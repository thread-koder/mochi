package dependency

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/database"
	corev1 "k8s.io/api/core/v1"
)

const SourceMochiEBPF = "mochi-ebpf"

type NodeRef struct {
	Kind      string
	Namespace string
	Name      string
}

type ResolvedEdge struct {
	From                NodeRef
	To                  NodeRef
	Protocol            string
	Port                int
	ViaServiceNamespace *string
	ViaServiceName      *string
	ViaServicePort      *int
	Source              string
	Connects            float64
	TxBytes             float64
	RxBytes             float64
	ActiveConnections   float64
	Evidence            json.RawMessage
}

type ResolveOptions struct {
	IncludeDNS bool
	cache      resolveCache
}

type resolveCache struct {
	serviceByIP             map[string]*database.Service
	serviceByNodePort       map[string]*database.Service
	nodeIP                  map[string]bool
	serviceByEndpointIP     map[string]*database.Service
	nodeRefByServiceDest    map[string]cachedNodeRef
	podsByIP                map[string][]*database.Pod
	nodeRefByUID            map[string]cachedNodeRef
	servicePorts            map[string][]corev1.ServicePort
	endpointSlicesByService map[string][]*database.EndpointSlice
}

type cachedNodeRef struct {
	ref NodeRef
	ok  bool
}

// DefaultResolveOptions returns options with a ready pass-local cache.
// Callers of Resolve must use this (or equivalent initialized maps),
// a zero value panics on cache write.
func DefaultResolveOptions() ResolveOptions {
	return ResolveOptions{
		IncludeDNS: false,
		cache: resolveCache{
			serviceByIP:             make(map[string]*database.Service),
			serviceByNodePort:       make(map[string]*database.Service),
			nodeIP:                  make(map[string]bool),
			serviceByEndpointIP:     make(map[string]*database.Service),
			nodeRefByServiceDest:    make(map[string]cachedNodeRef),
			podsByIP:                make(map[string][]*database.Pod),
			nodeRefByUID:            make(map[string]cachedNodeRef),
			servicePorts:            make(map[string][]corev1.ServicePort),
			endpointSlicesByService: make(map[string][]*database.EndpointSlice),
		},
	}
}

// Resolve turns one connection series into a workload edge, or drops it as noise/unresolvable src.
func Resolve(ctx context.Context, series ConnectionSeries, opts ResolveOptions) (ResolvedEdge, bool, error) {
	if series.Connects <= 0 && series.ActiveConnections <= 0 {
		return ResolvedEdge{}, false, nil
	}
	if !isKnownProtocol(series.Protocol) {
		return ResolvedEdge{}, false, nil
	}
	if !isValidIPAddress(series.ActualDstIP) {
		return ResolvedEdge{}, false, nil
	}
	if isLinkLocalOrMetadata(series.ActualDstIP) {
		return ResolvedEdge{}, false, nil
	}

	from, ok, err := lookupNodeRefByUID(ctx, opts, series.SrcPodUID, "src")
	if err != nil {
		return ResolvedEdge{}, false, err
	}
	if !ok {
		return ResolvedEdge{}, false, nil
	}

	to, err := resolveDestination(ctx, series, opts)
	if err != nil {
		return ResolvedEdge{}, false, err
	}

	if sameNode(from, to) {
		return ResolvedEdge{}, false, nil
	}
	if !opts.IncludeDNS && isDNSNoise(to) {
		return ResolvedEdge{}, false, nil
	}

	via, err := viaService(ctx, series, opts)
	if err != nil {
		return ResolvedEdge{}, false, err
	}

	var viaNS, viaName *string
	var viaPort *int
	if via != nil {
		viaNS = new(via.namespace)
		viaName = new(via.name)
		viaPort = via.port
	}

	evidence, err := json.Marshal(map[string]string{
		"src_pod_uid":     series.SrcPodUID,
		"dst_pod_uid":     series.DstPodUID,
		"dst_ip":          series.DstIP,
		"actual_dst_ip":   series.ActualDstIP,
		"dst_port":        strconv.Itoa(series.DstPort),
		"actual_dst_port": strconv.Itoa(series.ActualDstPort),
		"dst_hostname":    series.DstHostname,
	})
	if err != nil {
		return ResolvedEdge{}, false, fmt.Errorf("marshal evidence: %w", err)
	}

	edge := ResolvedEdge{
		From:                from,
		To:                  to,
		Protocol:            series.Protocol,
		Port:                series.ActualDstPort,
		ViaServiceNamespace: viaNS,
		ViaServiceName:      viaName,
		ViaServicePort:      viaPort,
		Source:              SourceMochiEBPF,
		Connects:            series.Connects,
		TxBytes:             series.TxBytes,
		RxBytes:             series.RxBytes,
		ActiveConnections:   series.ActiveConnections,
		Evidence:            evidence,
	}
	return edge, true, nil
}

func resolveDestination(ctx context.Context, series ConnectionSeries, opts ResolveOptions) (NodeRef, error) {
	if series.DstPodUID != "" {
		ref, ok, err := lookupNodeRefByUID(ctx, opts, series.DstPodUID, "dest")
		if err != nil {
			return NodeRef{}, err
		}
		if ok {
			return ref, nil
		}
	} else {
		isNode, err := nodeIPExists(ctx, opts, series.ActualDstIP)
		if err != nil {
			return NodeRef{}, err
		}
		if !isNode {
			pods, err := lookupPodsByIP(ctx, opts, series.ActualDstIP)
			if err != nil {
				return NodeRef{}, err
			}
			if len(pods) > 0 {
				ref, ok := resolvePodNodeRef(opts, pods[0])
				if ok {
					return ref, nil
				}
			}
		}
	}

	// Service VIP/NodePort dest (including node-address skip of GetPodsByIP) maps through EndpointSlice backends.
	ref, ok, err := resolveViaServiceEndpoints(ctx, opts, series.ActualDstIP, series.ActualDstPort, series.Protocol)
	if err != nil {
		return NodeRef{}, err
	}
	if ok {
		return ref, nil
	}
	if series.DstIP != "" && series.DstIP != series.ActualDstIP {
		ref, ok, err := resolveViaServiceEndpoints(ctx, opts, series.DstIP, series.DstPort, series.Protocol)
		if err != nil {
			return NodeRef{}, err
		}
		if ok {
			return ref, nil
		}
	}

	name := series.ActualDstIP
	if hostname := normalizeHostname(series.DstHostname); hostname != "" {
		name = hostname
	}
	return NodeRef{
		Kind:      KindExternal,
		Namespace: "",
		Name:      name,
	}, nil
}

func resolvePodNodeRef(opts ResolveOptions, pod *database.Pod) (NodeRef, bool) {
	if cached, hit := opts.cache.nodeRefByUID[pod.UID]; hit {
		return cached.ref, cached.ok
	}
	ref, ok := nodeRefFromPod(pod)
	opts.cache.nodeRefByUID[pod.UID] = cachedNodeRef{
		ref: ref,
		ok:  ok,
	}
	return ref, ok
}

func lookupNodeRefByUID(ctx context.Context, opts ResolveOptions, uid, what string) (NodeRef, bool, error) {
	if cached, hit := opts.cache.nodeRefByUID[uid]; hit {
		return cached.ref, cached.ok, nil
	}

	pod, err := database.GetPodIdentityByUID(ctx, uid)
	if err != nil {
		if errors.Is(err, &apperrors.NotFoundError{}) {
			opts.cache.nodeRefByUID[uid] = cachedNodeRef{}
			return NodeRef{}, false, nil
		}
		return NodeRef{}, false, fmt.Errorf("lookup %s pod %s: %w", what, uid, err)
	}

	ref, ok := resolvePodNodeRef(opts, pod)
	return ref, ok, nil
}

func lookupPodsByIP(ctx context.Context, opts ResolveOptions, ip string) ([]*database.Pod, error) {
	if pods, hit := opts.cache.podsByIP[ip]; hit {
		return pods, nil
	}
	pods, err := database.GetPodsByIP(ctx, ip)
	if err != nil {
		return nil, fmt.Errorf("lookup pods by IP %s: %w", ip, err)
	}
	opts.cache.podsByIP[ip] = pods
	return pods, nil
}

func nodeIPExists(ctx context.Context, opts ResolveOptions, ip string) (bool, error) {
	if exists, hit := opts.cache.nodeIP[ip]; hit {
		return exists, nil
	}
	exists, err := database.NodeIPExists(ctx, ip)
	if err != nil {
		return false, fmt.Errorf("lookup node IP %s: %w", ip, err)
	}
	opts.cache.nodeIP[ip] = exists
	return exists, nil
}

func lookupService(ctx context.Context, opts ResolveOptions, ip string, port int, protocol string) (*database.Service, bool, error) {
	if ip == "" {
		return nil, false, nil
	}

	var vipSvc *database.Service
	if cached, hit := opts.cache.serviceByIP[ip]; hit {
		vipSvc = cached
	} else {
		svc, err := database.GetServiceByIP(ctx, ip)
		if err != nil {
			if errors.Is(err, &apperrors.NotFoundError{}) {
				opts.cache.serviceByIP[ip] = nil
			} else {
				return nil, false, fmt.Errorf("lookup service by IP %s: %w", ip, err)
			}
		} else {
			opts.cache.serviceByIP[ip] = svc
			vipSvc = svc
		}
	}

	// kube-proxy portals are IP+spec.port. A unique portal match applies everywhere
	// (ClusterIP, ExternalIP/LB on a node address) without consulting NodePort.
	if vipSvc != nil {
		ports, err := parsedServicePorts(opts, vipSvc)
		if err != nil {
			return nil, false, err
		}
		if uniqueServiceSpecPortMatch(ports, protocol, port) {
			return vipSvc, true, nil
		}
	}

	isNode, err := nodeIPExists(ctx, opts, ip)
	if err != nil {
		return nil, false, err
	}
	if vipSvc != nil && !isNode {
		return vipSvc, true, nil
	}
	if !isNode {
		return nil, false, nil
	}

	npKey := nodePortCacheKey(protocol, port)
	if svc, hit := opts.cache.serviceByNodePort[npKey]; hit {
		return svc, svc != nil, nil
	}

	svc, err := database.GetServiceByNodePort(ctx, protocol, port)
	if err != nil {
		if errors.Is(err, &apperrors.NotFoundError{}) {
			opts.cache.serviceByNodePort[npKey] = nil
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("lookup service by node port %s: %w", npKey, err)
	}
	opts.cache.serviceByNodePort[npKey] = svc
	return svc, true, nil
}

func lookupHeadlessService(ctx context.Context, opts ResolveOptions, ip string) (*database.Service, bool, error) {
	if svc, hit := opts.cache.serviceByEndpointIP[ip]; hit {
		return svc, svc != nil, nil
	}

	services, err := database.GetHeadlessServicesByEndpointIP(ctx, ip)
	if err != nil {
		return nil, false, fmt.Errorf("lookup headless service by endpoint IP %s: %w", ip, err)
	}
	if len(services) != 1 {
		opts.cache.serviceByEndpointIP[ip] = nil
		return nil, false, nil
	}

	opts.cache.serviceByEndpointIP[ip] = services[0]
	return services[0], true, nil
}

func lookupEndpointSlices(ctx context.Context, opts ResolveOptions, namespace, serviceName string) ([]*database.EndpointSlice, error) {
	key := serviceCacheKey(namespace, serviceName)
	if slices, hit := opts.cache.endpointSlicesByService[key]; hit {
		return slices, nil
	}

	slices, err := database.GetEndpointSlicesByService(ctx, namespace, serviceName)
	if err != nil {
		return nil, fmt.Errorf("lookup endpoint slices for %s/%s: %w", namespace, serviceName, err)
	}
	opts.cache.endpointSlicesByService[key] = slices
	return slices, nil
}

func resolveViaServiceEndpoints(ctx context.Context, opts ResolveOptions, destIP string, destPort int, protocol string) (NodeRef, bool, error) {
	if destIP == "" {
		return NodeRef{}, false, nil
	}
	destKey := serviceDestCacheKey(destIP, protocol, destPort)
	if cached, hit := opts.cache.nodeRefByServiceDest[destKey]; hit {
		return cached.ref, cached.ok, nil
	}

	svc, found, err := lookupService(ctx, opts, destIP, destPort, protocol)
	if err != nil {
		return NodeRef{}, false, err
	}
	if !found {
		opts.cache.nodeRefByServiceDest[destKey] = cachedNodeRef{}
		return NodeRef{}, false, nil
	}

	slices, err := lookupEndpointSlices(ctx, opts, svc.Namespace, svc.Name)
	if err != nil {
		return NodeRef{}, false, err
	}

	for _, es := range slices {
		addrs, err := endpointAddresses(es.Endpoints)
		if err != nil {
			return NodeRef{}, false, fmt.Errorf("parse endpoints for %s/%s: %w", es.Namespace, es.Name, err)
		}
		for _, addr := range addrs {
			pods, err := lookupPodsByIP(ctx, opts, addr)
			if err != nil {
				return NodeRef{}, false, err
			}
			if len(pods) == 0 {
				continue
			}
			ref, ok := resolvePodNodeRef(opts, pods[0])
			if ok {
				opts.cache.nodeRefByServiceDest[destKey] = cachedNodeRef{
					ref: ref,
					ok:  true,
				}
				return ref, true, nil
			}
		}
	}

	opts.cache.nodeRefByServiceDest[destKey] = cachedNodeRef{}
	return NodeRef{}, false, nil
}

func endpointAddresses(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var endpoints []struct {
		Addresses []string `json:"addresses"`
	}
	if err := json.Unmarshal(raw, &endpoints); err != nil {
		return nil, err
	}
	var results []string
	for _, ep := range endpoints {
		results = append(results, ep.Addresses...)
	}
	return results, nil
}

func nodePortCacheKey(protocol string, port int) string {
	return strings.Join([]string{strings.ToLower(protocol), strconv.Itoa(port)}, "\x00")
}

func serviceDestCacheKey(ip, protocol string, port int) string {
	return strings.Join([]string{ip, strings.ToLower(protocol), strconv.Itoa(port)}, "\x00")
}
