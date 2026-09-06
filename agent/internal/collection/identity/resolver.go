package identity

import (
	"context"
	"fmt"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thread_koder/mochi/agent/internal/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const unresolvedWarnInterval = 1 * time.Minute
const cgroupReconcileInterval = 1 * time.Minute

type PodInfo struct {
	UID       string
	Namespace string
	Name      string

	nodeName    string
	hostNetwork bool
	ips         []string
}

// Resolver maps host pid/cgroup to pod identity (this node) and dest IPs to
// unique non-hostNetwork pods (cluster-wide informer for podsByIP).
type Resolver struct {
	nodeName string

	mapsMu       sync.RWMutex
	podsByUID    map[string]PodInfo  // informer: UID -> identity
	podsByIP     map[string]string   // normalized IP -> UID (unique non-hostNetwork)
	uidByCgroup  map[uint64]string   // cgroupfs: inode -> pod UID
	nonPodCgroup map[uint64]struct{} // negative cache: known non-pod inodes
	factory      informers.SharedInformerFactory
	stopOnce     sync.Once
	rebuildMu    sync.Mutex

	unresolvedCount atomic.Uint64
	onPodDeleted    func(string)
}

func NewResolver(nodeName string, onPodDeleted func(string)) *Resolver {
	return &Resolver{
		nodeName:     nodeName,
		podsByUID:    make(map[string]PodInfo),
		podsByIP:     make(map[string]string),
		uidByCgroup:  make(map[uint64]string),
		nonPodCgroup: make(map[uint64]struct{}),
		onPodDeleted: onPodDeleted,
	}
}

func (r *Resolver) Start(ctx context.Context) error {
	log := logger.WithComponent("identity")

	clientset, err := newClientset()
	if err != nil {
		return fmt.Errorf("create kubernetes client: %w", err)
	}

	r.factory = informers.NewSharedInformerFactoryWithOptions(
		clientset,
		0,
		informers.WithTransform(slimPod),
	)

	informer := r.factory.Core().V1().Pods().Informer()
	handle, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    r.onPodApply,
		UpdateFunc: func(_, newObj any) { r.onPodApply(newObj) },
		DeleteFunc: r.onPodRemove,
	})
	if err != nil {
		r.Stop()
		return fmt.Errorf("add pod event handler: %w", err)
	}

	r.factory.StartWithContext(ctx)
	if !cache.WaitFor(ctx, "", handle.HasSyncedChecker()) {
		r.Stop()
		return fmt.Errorf("wait for pod event handler sync: %w", context.Cause(ctx))
	}

	go r.warnUnresolved(ctx)
	go r.watchCgroups(ctx)
	go r.reconcileCgroups(ctx)

	log.Info().Str("node", r.nodeName).Msg("Pod identity started")
	return nil
}

func (r *Resolver) Stop() {
	r.stopOnce.Do(func() {
		if r.factory != nil {
			r.factory.Shutdown()
		}
	})
}

// ResolvePID returns pod labels for a host pid, or ok=false if unknown.
func (r *Resolver) ResolvePID(pid uint32) (PodInfo, bool) {
	return r.Resolve(pid, 0)
}

// Resolve maps a host pid and/or cgroup id to pod identity.
// BPF stashes cgroup id at connect time so attribution still works after
// /proc/<pid> and the cgroup dir are gone.
func (r *Resolver) Resolve(pid uint32, cgroupID uint64) (PodInfo, bool) {
	if cgroupID != 0 {
		if info, ok := r.lookupCgroup(cgroupID); ok {
			return info, true
		}
	}

	pidGone := false
	if pid != 0 {
		pidCgroup := pidCgroupLookup(pid)
		switch pidCgroup.kind {
		case pidCgroupPod:
			r.learn(pidCgroup)
			if info, ok := r.lookupUID(pidCgroup.podUID); ok {
				return info, true
			}
			r.unresolvedCount.Add(1)
			return PodInfo{}, false
		case pidCgroupHost:
			return PodInfo{}, false
		case pidCgroupGone:
			pidGone = true
		}
	}

	// Gone pid (or pid 0): fall through to the stashed cgroup inode.
	if cgroupID != 0 && (pid == 0 || pidGone) {
		r.mapsMu.RLock()
		_, knownNonPod := r.nonPodCgroup[cgroupID]
		r.mapsMu.RUnlock()
		if knownNonPod {
			return PodInfo{}, false
		}

		r.rebuild()

		if info, ok := r.lookupCgroup(cgroupID); ok {
			return info, true
		}

		r.mapsMu.Lock()
		r.nonPodCgroup[cgroupID] = struct{}{}
		r.mapsMu.Unlock()
	}

	return PodInfo{}, false
}

func (r *Resolver) LookupByIP(addr netip.Addr) (PodInfo, bool) {
	key := AddrKey(addr)
	if key == "" {
		return PodInfo{}, false
	}
	r.mapsMu.RLock()
	uid, ok := r.podsByIP[key]
	if !ok {
		r.mapsMu.RUnlock()
		return PodInfo{}, false
	}
	info, ok := r.podsByUID[uid]
	r.mapsMu.RUnlock()
	return info, ok
}

func (r *Resolver) lookupCgroup(cgroupID uint64) (PodInfo, bool) {
	r.mapsMu.RLock()
	podUID, ok := r.uidByCgroup[cgroupID]
	if !ok {
		r.mapsMu.RUnlock()
		return PodInfo{}, false
	}
	info, ok := r.podsByUID[podUID]
	r.mapsMu.RUnlock()
	if !ok {
		r.unresolvedCount.Add(1)
		r.unindex(cgroupID)
		return PodInfo{}, false
	}
	return info, true
}

func (r *Resolver) lookupUID(podUID string) (PodInfo, bool) {
	r.mapsMu.RLock()
	info, ok := r.podsByUID[podUID]
	r.mapsMu.RUnlock()
	return info, ok
}

func (r *Resolver) learn(pidCgroup pidCgroupResult) {
	inode, ok := inodeOf(pidCgroup.cgroupDir)
	if !ok {
		return
	}
	r.index(inode, pidCgroup.podUID)
}

func (r *Resolver) index(inode uint64, podUID string) {
	r.rebuildMu.Lock()
	r.mapsMu.Lock()
	r.uidByCgroup[inode] = podUID
	delete(r.nonPodCgroup, inode)
	r.mapsMu.Unlock()
	r.rebuildMu.Unlock()
}

func (r *Resolver) unindex(inode uint64) {
	r.rebuildMu.Lock()
	r.mapsMu.Lock()
	delete(r.uidByCgroup, inode)
	r.mapsMu.Unlock()
	r.rebuildMu.Unlock()
}

func (r *Resolver) rebuild() {
	r.rebuildMu.Lock()
	defer r.rebuildMu.Unlock()

	walked := make(map[uint64]string)
	walkPodCgroupInodes(func(inode uint64, podUID string) {
		walked[inode] = podUID
	})

	r.mapsMu.Lock()
	for inode, uid := range walked {
		if _, ok := r.podsByUID[uid]; !ok {
			delete(walked, inode)
		}
	}
	r.uidByCgroup = walked
	r.mapsMu.Unlock()
}

func (r *Resolver) reconcileCgroups(ctx context.Context) {
	ticker := time.NewTicker(cgroupReconcileInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.rebuild()
		}
	}
}

func (r *Resolver) onPodApply(obj any) {
	if info := podInfoFromObject(obj); info != nil {
		r.apply(info)
	}
}

func (r *Resolver) onPodRemove(obj any) {
	if info := podInfoFromObject(obj); info != nil {
		r.remove(info)
	}
}

func (r *Resolver) apply(info *PodInfo) {
	r.mapsMu.Lock()
	defer r.mapsMu.Unlock()

	if prev, ok := r.podsByUID[info.UID]; ok {
		for _, ip := range prev.ips {
			if uid, ok := r.podsByIP[ip]; ok && uid == prev.UID {
				delete(r.podsByIP, ip)
			}
		}
	}
	r.podsByUID[info.UID] = *info
	if info.hostNetwork {
		return
	}
	for _, ip := range info.ips {
		if existing, ok := r.podsByIP[ip]; ok && existing != info.UID {
			delete(r.podsByIP, ip)
			continue
		}
		r.podsByIP[ip] = info.UID
	}
}

func (r *Resolver) remove(info *PodInfo) {
	r.mapsMu.Lock()

	prev, ok := r.podsByUID[info.UID]
	if !ok {
		prev = *info
	} else {
		delete(r.podsByUID, info.UID)
	}
	for _, ip := range prev.ips {
		if uid, ok := r.podsByIP[ip]; ok && uid == prev.UID {
			delete(r.podsByIP, ip)
		}
	}

	if prev.nodeName == r.nodeName {
		for cgroupID, indexedUID := range r.uidByCgroup {
			if indexedUID == info.UID {
				delete(r.uidByCgroup, cgroupID)
			}
		}
	}
	r.mapsMu.Unlock()

	// After unlock so the callback does not nest under mapsMu.
	r.onPodDeleted(info.UID)
}

func (r *Resolver) warnUnresolved(ctx context.Context) {
	log := logger.WithComponent("identity")
	ticker := time.NewTicker(unresolvedWarnInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count := r.unresolvedCount.Swap(0)
			if count > 0 {
				log.Warn().Uint64("count", count).Msg("Unresolved pod sources dropped in the last minute")
			}
		}
	}
}

// slimPod keeps a *corev1.Pod so MetaNamespaceKeyFunc can still key the store.
func slimPod(obj any) (any, error) {
	switch t := obj.(type) {
	case *corev1.Pod:
		return slimPodObject(t), nil
	case cache.DeletedFinalStateUnknown:
		if pod, ok := t.Obj.(*corev1.Pod); ok {
			return cache.DeletedFinalStateUnknown{Key: t.Key, Obj: slimPodObject(pod)}, nil
		}
		return obj, nil
	default:
		return obj, nil
	}
}

func slimPodObject(pod *corev1.Pod) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              pod.Name,
			Namespace:         pod.Namespace,
			UID:               pod.UID,
			ResourceVersion:   pod.ResourceVersion,
			DeletionTimestamp: pod.DeletionTimestamp,
		},
		Spec: corev1.PodSpec{
			NodeName:    pod.Spec.NodeName,
			HostNetwork: pod.Spec.HostNetwork,
		},
		Status: corev1.PodStatus{
			PodIP:  pod.Status.PodIP,
			PodIPs: pod.Status.PodIPs,
		},
	}
}

func podInfoFromObject(obj any) *PodInfo {
	switch t := obj.(type) {
	case *corev1.Pod:
		return podInfoFromPod(t)
	case cache.DeletedFinalStateUnknown:
		return podInfoFromObject(t.Obj)
	default:
		return nil
	}
}

func podInfoFromPod(pod *corev1.Pod) *PodInfo {
	return &PodInfo{
		UID:         string(pod.UID),
		Namespace:   pod.Namespace,
		Name:        pod.Name,
		nodeName:    pod.Spec.NodeName,
		hostNetwork: pod.Spec.HostNetwork,
		ips:         podAddrKeys(pod),
	}
}

func podAddrKeys(pod *corev1.Pod) []string {
	var ips []string
	for _, podIP := range pod.Status.PodIPs {
		if key := addrKeyFromString(podIP.IP); key != "" {
			ips = append(ips, key)
		}
	}
	if len(ips) == 0 {
		if key := addrKeyFromString(pod.Status.PodIP); key != "" {
			ips = append(ips, key)
		}
	}
	return ips
}

func addrKeyFromString(raw string) string {
	addr, err := netip.ParseAddr(raw)
	if err != nil || !addr.IsGlobalUnicast() {
		return ""
	}
	return AddrKey(addr)
}

func AddrKey(addr netip.Addr) string {
	if !addr.IsValid() {
		return ""
	}
	return addr.Unmap().String()
}

func newClientset() (*kubernetes.Clientset, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules,
			&clientcmd.ConfigOverrides{},
		)
		cfg, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("in-cluster and kubeconfig failed: %w", err)
		}
	}
	return kubernetes.NewForConfig(cfg)
}
