package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/thread_koder/mochi/internal/config"
	"github.com/thread_koder/mochi/internal/database"
	"github.com/thread_koder/mochi/internal/logger"
)

// SyncResources fetches cluster resources and stores a DB snapshot.
// Each stage logs and continues so one failed resource kind does not block the rest.
func SyncResources(ctx context.Context) error {
	log := logger.WithComponent("kubernetes")

	if err := SyncNodes(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to sync nodes")
	}

	namespaces, err := Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to list namespaces")
	}
	if err := SyncNamespaces(ctx, namespaces); err != nil {
		log.Warn().Err(err).Msg("Failed to sync namespaces")
	}

	for _, ns := range namespaces.Items {
		if !shouldSyncNamespace(ns.Name) {
			continue
		}

		if err := SyncDeployments(ctx, ns.Name); err != nil {
			log.Warn().Err(err).Msg("Failed to sync deployments")
		}

		if err := SyncReplicaSets(ctx, ns.Name); err != nil {
			log.Warn().Err(err).Msg("Failed to sync replicasets")
		}

		if err := SyncStatefulSets(ctx, ns.Name); err != nil {
			log.Warn().Err(err).Msg("Failed to sync statefulsets")
		}

		if err := SyncDaemonSets(ctx, ns.Name); err != nil {
			log.Warn().Err(err).Msg("Failed to sync daemonsets")
		}

		if err := SyncServices(ctx, ns.Name); err != nil {
			log.Warn().Err(err).Msg("Failed to sync services")
		}

		if err := SyncEndpointSlices(ctx, ns.Name); err != nil {
			log.Warn().Err(err).Msg("Failed to sync endpoint slices")
		}

		if err := SyncPods(ctx, ns.Name); err != nil {
			log.Warn().Err(err).Msg("Failed to sync pods")
		}

		if err := SyncContainers(ctx, ns.Name); err != nil {
			log.Warn().Err(err).Msg("Failed to sync containers")
		}

	}

	return nil
}

// SyncNodes syncs nodes and prunes stale rows.
func SyncNodes(ctx context.Context) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	nodes, err := Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	log.Debug().Int("count", len(nodes.Items)).Msg("Syncing nodes...")

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
			OSImage:                 &node.Status.NodeInfo.OSImage,
			KernelVersion:           &node.Status.NodeInfo.KernelVersion,
			ContainerRuntimeVersion: &node.Status.NodeInfo.ContainerRuntimeVersion,
			KubeletVersion:          &node.Status.NodeInfo.KubeletVersion,
			CPUCapacity:             &cpuCapacityStr,
			MemoryCapacity:          &memoryCapacityStr,
			CPUAllocatable:          &cpuAllocatableStr,
			MemoryAllocatable:       &memoryAllocatableStr,
			Labels:                  labelsJSON,
			Annotations:             annotationsJSON,
			Conditions:              conditionsJSON,
			CreatedAt:               node.CreationTimestamp.Time,
			SyncedAt:                now,
		}

		dbNodes = append(dbNodes, dbNode)
	}

	if err := database.UpsertNodesBatch(ctx, dbNodes); err != nil {
		return fmt.Errorf("failed to upsert nodes: %w", err)
	}

	if err := database.PruneNodes(ctx, nodeUIDs); err != nil {
		log.Warn().Err(err).Msg("Failed to delete nodes not in current state")
	}
	log.Debug().Int("count", len(dbNodes)).Msg("Nodes synced successfully")
	return nil
}

// SyncNamespaces syncs namespaces and prunes stale rows.
func SyncNamespaces(ctx context.Context, namespaces *corev1.NamespaceList) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	log.Debug().Int("count", len(namespaces.Items)).Msg("Syncing namespaces...")

	dbNamespaces := make([]*database.Namespace, 0, len(namespaces.Items))
	now := time.Now()

	nsUIDs := make([]string, 0, len(namespaces.Items))
	for _, ns := range namespaces.Items {
		if !shouldSyncNamespace(ns.Name) {
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
	}

	if err := database.UpsertNamespacesBatch(ctx, dbNamespaces); err != nil {
		return fmt.Errorf("failed to upsert namespaces: %w", err)
	}

	if err := database.PruneNamespaces(ctx, nsUIDs); err != nil {
		log.Warn().Err(err).Msg("Failed to delete namespaces not in current state")
	}
	log.Debug().Int("count", len(dbNamespaces)).Msg("Namespaces synced successfully")
	return nil
}

// SyncDeployments syncs deployments in one namespace and prunes stale rows.
func SyncDeployments(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	deployments, err := Clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	log.Debug().Int("count", len(deployments.Items)).Msg("Syncing deployments...")

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
		return fmt.Errorf("failed to upsert deployments: %w", err)
	}

	if err := database.PruneDeployments(ctx, namespace, depUIDs); err != nil {
		log.Warn().Err(err).Msg("Failed to prune deployments not in current state")
	}
	log.Debug().Int("count", len(dbDeployments)).Msg("Deployments synced successfully")
	return nil
}

// SyncReplicaSets syncs active ReplicaSets in one namespace and prunes stale rows.
func SyncReplicaSets(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	replicasets, err := Clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list replicasets: %w", err)
	}

	log.Debug().Int("count", len(replicasets.Items)).Msg("Syncing replicasets...")

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
			ownerKind = new(owner.Kind)
			ownerName = new(owner.Name)
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
		return fmt.Errorf("failed to upsert replicasets: %w", err)
	}

	if err := database.PruneReplicaSets(ctx, namespace, rsUIDs); err != nil {
		log.Warn().Err(err).Msg("Failed to delete replicasets not in current state")
	}
	log.Debug().Int("count", len(dbReplicaSets)).Msg("Replicasets synced successfully")
	return nil
}

// SyncStatefulSets syncs statefulsets in one namespace and prunes stale rows.
func SyncStatefulSets(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	statefulsets, err := Clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list statefulsets: %w", err)
	}

	log.Debug().Int("count", len(statefulsets.Items)).Msg("Syncing statefulsets...")

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
		return fmt.Errorf("failed to upsert statefulsets: %w", err)
	}

	if err := database.PruneStatefulSets(ctx, namespace, stsUIDs); err != nil {
		log.Warn().Err(err).Msg("Failed to delete statefulsets not in current state")
	}
	log.Debug().Int("count", len(dbStatefulSets)).Msg("Statefulsets synced successfully")
	return nil
}

// SyncDaemonSets syncs daemonsets in one namespace and prunes stale rows.
func SyncDaemonSets(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	daemonsets, err := Clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list daemonsets: %w", err)
	}

	log.Debug().Int("count", len(daemonsets.Items)).Msg("Syncing daemonsets...")

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
		return fmt.Errorf("failed to upsert daemonsets: %w", err)
	}

	if err := database.PruneDaemonSets(ctx, namespace, dsUIDs); err != nil {
		log.Warn().Err(err).Msg("Failed to delete daemonsets not in current state")
	}
	log.Debug().Int("count", len(dbDaemonSets)).Msg("Daemonsets synced successfully")
	return nil
}

// SyncServices syncs services in one namespace and prunes stale rows.
func SyncServices(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	services, err := Clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	log.Debug().Int("count", len(services.Items)).Msg("Syncing services...")

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

		clusterIP := svc.Spec.ClusterIP
		var clusterIPPtr *string
		if clusterIP != "" {
			clusterIPPtr = new(clusterIP)
		}

		dbService := &database.Service{
			Name:        svc.Name,
			Namespace:   svc.Namespace,
			UID:         svcUID,
			Type:        string(svc.Spec.Type),
			ClusterIP:   clusterIPPtr,
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
		return fmt.Errorf("failed to upsert services: %w", err)
	}

	if err := database.PruneServices(ctx, namespace, svcUIDs); err != nil {
		log.Warn().Err(err).Msg("Failed to delete services not in current state")
	}
	log.Debug().Int("count", len(dbServices)).Msg("Services synced successfully")
	return nil
}

// SyncEndpointSlices syncs EndpointSlices in one namespace and prunes stale rows.
func SyncEndpointSlices(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	endpointSlices, err := Clientset.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list endpoint slices: %w", err)
	}

	log.Debug().Int("count", len(endpointSlices.Items)).Msg("Syncing endpoint slices...")

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
			ownerKind = &owner.Kind
			ownerName = &owner.Name
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
		return fmt.Errorf("failed to upsert endpoint slices: %w", err)
	}

	if err := database.PruneEndpointSlices(ctx, namespace, esUIDs); err != nil {
		log.Warn().Err(err).Msg("Failed to delete endpoint slices not in current state")
	}
	log.Debug().Int("count", len(dbEndpointSlices)).Msg("Endpoint slices synced successfully")
	return nil
}

// SyncPods syncs pods in one namespace and prunes stale rows.
func SyncPods(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	pods, err := Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	log.Debug().Int("count", len(pods.Items)).Msg("Syncing pods...")

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
			ownerKind = new(owner.Kind)
			ownerName = new(owner.Name)
			break // OwnerReferences are ordered by controller, so we only persist the primary owner (eg. Deployment).
		}

		restartPolicy := string(pod.Spec.RestartPolicy)
		var node *string
		if pod.Spec.NodeName != "" {
			node = new(pod.Spec.NodeName)
		}

		dbPod := &database.Pod{
			Name:          pod.Name,
			Namespace:     pod.Namespace,
			UID:           podUID,
			Node:          node,
			Phase:         string(pod.Status.Phase),
			RestartPolicy: &restartPolicy,
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
		return fmt.Errorf("failed to upsert pods: %w", err)
	}

	if err := database.PrunePods(ctx, namespace, podUIDs); err != nil {
		log.Warn().Err(err).Msg("Failed to delete pods not in current state")
	}
	log.Debug().Int("count", len(dbPods)).Msg("Pods synced successfully")
	return nil
}

// SyncContainers syncs pod container specs in one namespace and prunes stale rows.
func SyncContainers(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

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
			var imagePullPolicy *string

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

			if container.ImagePullPolicy != "" {
				imagePullPolicy = new(string(container.ImagePullPolicy))
			}

			portsJSON, _ := sliceToJSON(container.Ports)

			dbContainer := &database.Container{
				Name:            container.Name,
				PodUID:          podUID,
				PodName:         podName,
				Namespace:       namespace,
				Image:           container.Image,
				ImagePullPolicy: imagePullPolicy,
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
		log.Debug().Int("count", len(dbContainers)).Msg("Syncing containers...")
		if err := database.UpsertContainersBatch(ctx, dbContainers); err != nil {
			return fmt.Errorf("failed to upsert containers: %w", err)
		}
		log.Debug().Int("count", len(dbContainers)).Msg("Containers synced successfully")
	}

	if err := database.PruneContainers(ctx, namespace, podUIDs); err != nil {
		log.Warn().Err(err).Msg("Failed to delete containers not in current state")
	}
	return nil
}

// shouldSyncNamespace returns whether a namespace should be included in the sync.
// IncludeNamespaces takes precedence over ExcludeNamespaces for explicit allowlists.
func shouldSyncNamespace(namespace string) bool {
	cfg := config.AppConfig.Workers

	if len(cfg.IncludeNamespaces) > 0 {
		return slices.Contains(cfg.IncludeNamespaces, namespace)
	}

	if len(cfg.ExcludeNamespaces) > 0 {
		return !slices.Contains(cfg.ExcludeNamespaces, namespace)
	}

	return true
}

// mapToJSON encodes a string map for jsonb columns.
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

// sliceToJSON encodes arbitrary slices/structs for jsonb columns.
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
