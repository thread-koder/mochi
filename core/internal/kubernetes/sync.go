package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/thread_koder/mochi/core/internal/config"
	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/logger"
	"golang.org/x/sync/errgroup"
)

const namespaceSyncConcurrency = 4

func SyncResources(ctx context.Context, workerCfg *config.WorkerSyncConfig) {
	log := logger.WithComponent("kubernetes")

	if err := syncNodes(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to sync cluster nodes")
	}

	syncedNames, err := syncNamespaces(ctx, workerCfg)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to sync cluster namespaces")
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(namespaceSyncConcurrency)

	for _, name := range syncedNames {
		g.Go(func() error {
			syncNamespace(gctx, name)
			return nil
		})
	}
	_ = g.Wait()
}

func syncNodes(ctx context.Context) error {
	log := logger.WithComponent("kubernetes")
	start := time.Now()
	log.Info().Msg("Syncing cluster nodes...")

	nodes, err := Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list cluster nodes: %w", err)
	}

	dbNodes := make([]*database.Node, 0, len(nodes.Items))
	now := time.Now()

	nodeUIDs := make([]string, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		nodeUID := string(node.UID)
		nodeUIDs = append(nodeUIDs, nodeUID)
		labelsJSON, _ := mapToJSON(node.Labels)
		annotationsJSON, _ := mapToJSON(node.Annotations)

		var internalIP, externalIP *string
		for _, addr := range node.Status.Addresses {
			switch addr.Type {
			case corev1.NodeInternalIP:
				internalIP = new(addr.Address)
			case corev1.NodeExternalIP:
				externalIP = new(addr.Address)
			}
		}

		cpuCapacity := node.Status.Capacity[corev1.ResourceCPU]
		memoryCapacity := node.Status.Capacity[corev1.ResourceMemory]
		cpuAllocatable := node.Status.Allocatable[corev1.ResourceCPU]
		memoryAllocatable := node.Status.Allocatable[corev1.ResourceMemory]

		cpuCapacityStr := cpuCapacity.String()
		memoryCapacityStr := memoryCapacity.String()
		cpuAllocatableStr := cpuAllocatable.String()
		memoryAllocatableStr := memoryAllocatable.String()

		conditionsJSON, _ := sliceToJSON(node.Status.Conditions)

		dbNode := &database.Node{
			Name:                    node.Name,
			UID:                     nodeUID,
			InternalIP:              internalIP,
			ExternalIP:              externalIP,
			OSImage:                 node.Status.NodeInfo.OSImage,
			KernelVersion:           node.Status.NodeInfo.KernelVersion,
			ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
			KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
			CPUCapacity:             cpuCapacityStr,
			MemoryCapacity:          memoryCapacityStr,
			CPUAllocatable:          cpuAllocatableStr,
			MemoryAllocatable:       memoryAllocatableStr,
			Labels:                  labelsJSON,
			Annotations:             annotationsJSON,
			Conditions:              conditionsJSON,
			CreatedAt:               node.CreationTimestamp.Time,
			SyncedAt:                now,
		}

		dbNodes = append(dbNodes, dbNode)
	}

	if err := database.UpsertNodesBatch(ctx, dbNodes); err != nil {
		return err
	}

	if err := database.PruneNodes(ctx, nodeUIDs); err != nil {
		log.Warn().Err(err).Msg("Failed to prune cluster nodes not in current state")
	}
	log.Info().
		Str("duration", time.Since(start).Round(time.Millisecond).String()).
		Msg("Cluster nodes synced")
	return nil
}

func syncNamespaces(ctx context.Context, workerCfg *config.WorkerSyncConfig) ([]string, error) {
	log := logger.WithComponent("kubernetes")
	start := time.Now()
	log.Info().Msg("Syncing cluster namespaces...")

	namespaces, err := Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster namespaces: %w", err)
	}

	dbNamespaces := make([]*database.Namespace, 0, len(namespaces.Items))
	syncedNames := make([]string, 0, len(namespaces.Items))
	now := time.Now()

	nsUIDs := make([]string, 0, len(namespaces.Items))
	for _, ns := range namespaces.Items {
		if !workerCfg.ShouldSyncNamespace(ns.Name) {
			continue
		}
		nsUID := string(ns.UID)
		nsUIDs = append(nsUIDs, nsUID)
		labelsJSON, _ := mapToJSON(ns.Labels)
		annotationsJSON, _ := mapToJSON(ns.Annotations)

		dbNamespace := &database.Namespace{
			Name:        ns.Name,
			UID:         nsUID,
			Phase:       string(ns.Status.Phase),
			Labels:      labelsJSON,
			Annotations: annotationsJSON,
			CreatedAt:   ns.CreationTimestamp.Time,
			SyncedAt:    now,
		}

		dbNamespaces = append(dbNamespaces, dbNamespace)
		syncedNames = append(syncedNames, ns.Name)
	}

	if err := database.UpsertNamespacesBatch(ctx, dbNamespaces); err != nil {
		return nil, err
	}

	if err := database.PruneNamespaces(ctx, nsUIDs); err != nil {
		log.Warn().Err(err).Msg("Failed to prune cluster namespaces not in current state")
	}
	log.Info().
		Str("duration", time.Since(start).Round(time.Millisecond).String()).
		Msg("Cluster namespaces synced")
	return syncedNames, nil
}

func syncNamespace(ctx context.Context, namespace string) {
	log := logger.WithComponent("kubernetes")
	start := time.Now()
	log.Info().Str("namespace", namespace).Msg("Syncing namespace...")

	kinds := []struct {
		name string
		fn   func(context.Context, string) error
	}{
		{"deployments", syncDeployments},
		{"replicasets", syncReplicaSets},
		{"statefulsets", syncStatefulSets},
		{"daemonsets", syncDaemonSets},
		{"services", syncServices},
		{"endpoint slices", syncEndpointSlices},
		{"pods", syncPods},
		{"containers", syncContainers},
	}
	for _, kind := range kinds {
		if err := kind.fn(ctx, namespace); err != nil {
			log.Warn().Err(err).Str("namespace", namespace).Str("kind", kind.name).Msg("Namespace sync stopped")
			return
		}
	}

	log.Info().
		Str("namespace", namespace).
		Str("duration", time.Since(start).Round(time.Millisecond).String()).
		Msg("Namespace synced")
}

func syncDeployments(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	deployments, err := Clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	dbDeployments := make([]*database.Deployment, 0, len(deployments.Items))
	now := time.Now()

	depUIDs := make([]string, 0, len(deployments.Items))
	for _, dep := range deployments.Items {
		depUID := string(dep.UID)
		depUIDs = append(depUIDs, depUID)
		labelsJSON, _ := mapToJSON(dep.Labels)
		annotationsJSON, _ := mapToJSON(dep.Annotations)

		replicas := int32(0)
		if dep.Spec.Replicas != nil {
			replicas = *dep.Spec.Replicas
		}

		dbDeployment := &database.Deployment{
			Name:              dep.Name,
			Namespace:         dep.Namespace,
			UID:               depUID,
			Replicas:          int(replicas),
			ReadyReplicas:     int(dep.Status.ReadyReplicas),
			AvailableReplicas: int(dep.Status.AvailableReplicas),
			Labels:            labelsJSON,
			Annotations:       annotationsJSON,
			CreatedAt:         dep.CreationTimestamp.Time,
			SyncedAt:          now,
		}

		dbDeployments = append(dbDeployments, dbDeployment)
	}

	if err := database.UpsertDeploymentsBatch(ctx, dbDeployments); err != nil {
		return err
	}

	if err := database.PruneDeployments(ctx, namespace, depUIDs); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune deployments not in current state")
	} else if err := database.PruneComputeRecommendations(ctx, namespace, "Deployment"); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune compute recommendations for deleted deployments")
	}
	return nil
}

func syncReplicaSets(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	replicasets, err := Clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list replicasets: %w", err)
	}

	dbReplicaSets := make([]*database.ReplicaSet, 0, len(replicasets.Items))
	now := time.Now()

	rsUIDs := make([]string, 0, len(replicasets.Items))
	for _, rs := range replicasets.Items {
		// Skip scaled-to-zero sets to avoid persisting historical rollout artifacts.
		if rs.Status.Replicas == 0 {
			continue
		}
		rsUID := string(rs.UID)
		rsUIDs = append(rsUIDs, rsUID)
		labelsJSON, _ := mapToJSON(rs.Labels)
		annotationsJSON, _ := mapToJSON(rs.Annotations)

		replicas := int32(0)
		if rs.Spec.Replicas != nil {
			replicas = *rs.Spec.Replicas
		}

		var ownerKind, ownerName *string
		for _, owner := range rs.OwnerReferences {
			ownerKind, ownerName = optionalOwner(owner.Kind, owner.Name)
			break // OwnerReferences are ordered by controller, so we only persist the primary owner (eg. Deployment).
		}

		dbReplicaSet := &database.ReplicaSet{
			Name:          rs.Name,
			Namespace:     rs.Namespace,
			UID:           rsUID,
			Replicas:      int(replicas),
			ReadyReplicas: int(rs.Status.ReadyReplicas),
			OwnerKind:     ownerKind,
			OwnerName:     ownerName,
			Labels:        labelsJSON,
			Annotations:   annotationsJSON,
			CreatedAt:     rs.CreationTimestamp.Time,
			SyncedAt:      now,
		}

		dbReplicaSets = append(dbReplicaSets, dbReplicaSet)
	}

	if err := database.UpsertReplicaSetsBatch(ctx, dbReplicaSets); err != nil {
		return err
	}

	if err := database.PruneReplicaSets(ctx, namespace, rsUIDs); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune replicasets not in current state")
	}
	return nil
}

func syncStatefulSets(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	statefulsets, err := Clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list statefulsets: %w", err)
	}

	dbStatefulSets := make([]*database.StatefulSet, 0, len(statefulsets.Items))
	now := time.Now()

	stsUIDs := make([]string, 0, len(statefulsets.Items))
	for _, sts := range statefulsets.Items {
		stsUID := string(sts.UID)
		stsUIDs = append(stsUIDs, stsUID)
		labelsJSON, _ := mapToJSON(sts.Labels)
		annotationsJSON, _ := mapToJSON(sts.Annotations)

		replicas := int32(0)
		if sts.Spec.Replicas != nil {
			replicas = *sts.Spec.Replicas
		}

		dbStatefulSet := &database.StatefulSet{
			Name:          sts.Name,
			Namespace:     sts.Namespace,
			UID:           stsUID,
			Replicas:      int(replicas),
			ReadyReplicas: int(sts.Status.ReadyReplicas),
			Labels:        labelsJSON,
			Annotations:   annotationsJSON,
			CreatedAt:     sts.CreationTimestamp.Time,
			SyncedAt:      now,
		}

		dbStatefulSets = append(dbStatefulSets, dbStatefulSet)
	}

	if err := database.UpsertStatefulSetsBatch(ctx, dbStatefulSets); err != nil {
		return err
	}

	if err := database.PruneStatefulSets(ctx, namespace, stsUIDs); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune statefulsets not in current state")
	} else if err := database.PruneComputeRecommendations(ctx, namespace, "StatefulSet"); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune compute recommendations for deleted statefulsets")
	}
	return nil
}

func syncDaemonSets(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	daemonsets, err := Clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list daemonsets: %w", err)
	}

	dbDaemonSets := make([]*database.DaemonSet, 0, len(daemonsets.Items))
	now := time.Now()

	dsUIDs := make([]string, 0, len(daemonsets.Items))
	for _, ds := range daemonsets.Items {
		dsUID := string(ds.UID)
		dsUIDs = append(dsUIDs, dsUID)
		labelsJSON, _ := mapToJSON(ds.Labels)
		annotationsJSON, _ := mapToJSON(ds.Annotations)

		dbDaemonSet := &database.DaemonSet{
			Name:                   ds.Name,
			Namespace:              ds.Namespace,
			UID:                    dsUID,
			DesiredNumberScheduled: int(ds.Status.DesiredNumberScheduled),
			NumberReady:            int(ds.Status.NumberReady),
			NumberAvailable:        int(ds.Status.NumberAvailable),
			Labels:                 labelsJSON,
			Annotations:            annotationsJSON,
			CreatedAt:              ds.CreationTimestamp.Time,
			SyncedAt:               now,
		}

		dbDaemonSets = append(dbDaemonSets, dbDaemonSet)
	}

	if err := database.UpsertDaemonSetsBatch(ctx, dbDaemonSets); err != nil {
		return err
	}

	if err := database.PruneDaemonSets(ctx, namespace, dsUIDs); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune daemonsets not in current state")
	} else if err := database.PruneComputeRecommendations(ctx, namespace, "DaemonSet"); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune compute recommendations for deleted daemonsets")
	}
	return nil
}

func syncServices(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	services, err := Clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	dbServices := make([]*database.Service, 0, len(services.Items))
	now := time.Now()

	svcUIDs := make([]string, 0, len(services.Items))
	for _, svc := range services.Items {
		svcUID := string(svc.UID)
		svcUIDs = append(svcUIDs, svcUID)
		labelsJSON, _ := mapToJSON(svc.Labels)
		annotationsJSON, _ := mapToJSON(svc.Annotations)
		selectorJSON, _ := mapToJSON(svc.Spec.Selector)
		portsJSON, _ := sliceToJSON(svc.Spec.Ports)

		dbService := &database.Service{
			Name:        svc.Name,
			Namespace:   svc.Namespace,
			UID:         svcUID,
			Type:        string(svc.Spec.Type),
			ClusterIP:   optionalString(svc.Spec.ClusterIP),
			Ports:       portsJSON,
			Selector:    selectorJSON,
			Labels:      labelsJSON,
			Annotations: annotationsJSON,
			CreatedAt:   svc.CreationTimestamp.Time,
			SyncedAt:    now,
		}

		dbServices = append(dbServices, dbService)
	}

	if err := database.UpsertServicesBatch(ctx, dbServices); err != nil {
		return err
	}

	if err := database.PruneServices(ctx, namespace, svcUIDs); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune services not in current state")
	}
	return nil
}

func syncEndpointSlices(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	endpointSlices, err := Clientset.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list endpoint slices: %w", err)
	}

	dbEndpointSlices := make([]*database.EndpointSlice, 0, len(endpointSlices.Items))
	now := time.Now()

	esUIDs := make([]string, 0, len(endpointSlices.Items))
	for _, es := range endpointSlices.Items {
		esUID := string(es.UID)
		esUIDs = append(esUIDs, esUID)
		labelsJSON, _ := mapToJSON(es.Labels)
		annotationsJSON, _ := mapToJSON(es.Annotations)

		var ownerKind, ownerName *string
		for _, owner := range es.OwnerReferences {
			ownerKind, ownerName = optionalOwner(owner.Kind, owner.Name)
			break // OwnerReferences are ordered by controller, so we only persist the primary owner (eg. Service).
		}

		endpointsJSON, _ := sliceToJSON(es.Endpoints)
		portsJSON, _ := sliceToJSON(es.Ports)

		dbEndpointSlice := &database.EndpointSlice{
			Name:        es.Name,
			Namespace:   es.Namespace,
			UID:         esUID,
			AddressType: string(es.AddressType),
			OwnerKind:   ownerKind,
			OwnerName:   ownerName,
			Endpoints:   endpointsJSON,
			Ports:       portsJSON,
			Labels:      labelsJSON,
			Annotations: annotationsJSON,
			CreatedAt:   es.CreationTimestamp.Time,
			SyncedAt:    now,
		}

		dbEndpointSlices = append(dbEndpointSlices, dbEndpointSlice)
	}

	if err := database.UpsertEndpointSlicesBatch(ctx, dbEndpointSlices); err != nil {
		return err
	}

	if err := database.PruneEndpointSlices(ctx, namespace, esUIDs); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune endpoint slices not in current state")
	}
	return nil
}

func syncPods(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	pods, err := Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	dbPods := make([]*database.Pod, 0, len(pods.Items))
	now := time.Now()

	podUIDs := make([]string, 0, len(pods.Items))
	for _, pod := range pods.Items {
		podUID := string(pod.UID)
		podUIDs = append(podUIDs, podUID)
		labelsJSON, _ := mapToJSON(pod.Labels)
		annotationsJSON, _ := mapToJSON(pod.Annotations)

		var ownerKind, ownerName *string
		for _, owner := range pod.OwnerReferences {
			ownerKind, ownerName = optionalOwner(owner.Kind, owner.Name)
			break // OwnerReferences are ordered by controller, so we only persist the primary owner (eg. Deployment).
		}

		dbPod := &database.Pod{
			Name:          pod.Name,
			Namespace:     pod.Namespace,
			UID:           podUID,
			Node:          optionalString(pod.Spec.NodeName),
			PodIP:         optionalString(pod.Status.PodIP),
			Phase:         string(pod.Status.Phase),
			RestartPolicy: string(pod.Spec.RestartPolicy),
			Labels:        labelsJSON,
			Annotations:   annotationsJSON,
			OwnerKind:     ownerKind,
			OwnerName:     ownerName,
			CreatedAt:     pod.CreationTimestamp.Time,
			SyncedAt:      now,
		}

		dbPods = append(dbPods, dbPod)
	}

	if err := database.UpsertPodsBatch(ctx, dbPods); err != nil {
		return err
	}

	if err := database.PrunePods(ctx, namespace, podUIDs); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune pods not in current state")
	} else if err := database.PruneComputeRecommendations(ctx, namespace, "Pod"); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune compute recommendations for deleted pods")
	}
	return nil
}

func syncContainers(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	pods, err := Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	dbContainers := make([]*database.Container, 0)
	now := time.Now()

	podUIDs := make([]string, 0, len(pods.Items))
	for _, pod := range pods.Items {
		podUID := string(pod.UID)
		podUIDs = append(podUIDs, podUID)
		podName := pod.Name
		namespace := pod.Namespace

		for _, container := range pod.Spec.Containers {
			var cpuRequest, cpuLimit, memoryRequest, memoryLimit *string

			if container.Resources.Requests != nil {
				if cpu := container.Resources.Requests[corev1.ResourceCPU]; !cpu.IsZero() {
					cpuRequest = new(cpu.String())
				}
				if mem := container.Resources.Requests[corev1.ResourceMemory]; !mem.IsZero() {
					memoryRequest = new(mem.String())
				}
			}
			if container.Resources.Limits != nil {
				if cpu := container.Resources.Limits[corev1.ResourceCPU]; !cpu.IsZero() {
					cpuLimit = new(cpu.String())
				}
				if mem := container.Resources.Limits[corev1.ResourceMemory]; !mem.IsZero() {
					memoryLimit = new(mem.String())
				}
			}

			portsJSON, _ := sliceToJSON(container.Ports)

			dbContainer := &database.Container{
				Name:            container.Name,
				PodUID:          podUID,
				PodName:         podName,
				Namespace:       namespace,
				Image:           container.Image,
				ImagePullPolicy: string(container.ImagePullPolicy),
				Ports:           portsJSON,
				CPURequest:      cpuRequest,
				CPULimit:        cpuLimit,
				MemoryRequest:   memoryRequest,
				MemoryLimit:     memoryLimit,
				CreatedAt:       pod.CreationTimestamp.Time,
				SyncedAt:        now,
			}

			dbContainers = append(dbContainers, dbContainer)
		}
	}

	if len(dbContainers) > 0 {
		if err := database.UpsertContainersBatch(ctx, dbContainers); err != nil {
			return err
		}
	}

	if err := database.PruneContainers(ctx, namespace, podUIDs); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune containers not in current state")
	}
	return nil
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return new(s)
}

func optionalOwner(kind, name string) (ownerKind, ownerName *string) {
	ownerKind = optionalString(kind)
	ownerName = optionalString(name)
	if ownerKind == nil || ownerName == nil {
		return nil, nil
	}
	return ownerKind, ownerName
}

func mapToJSON(m map[string]string) (json.RawMessage, error) {
	if len(m) == 0 {
		return json.RawMessage("{}"), nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal map to JSON: %w", err)
	}
	return json.RawMessage(data), nil
}

func sliceToJSON(s any) (json.RawMessage, error) {
	if s == nil {
		return json.RawMessage("[]"), nil
	}
	data, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal slice to JSON: %w", err)
	}
	return json.RawMessage(data), nil
}
