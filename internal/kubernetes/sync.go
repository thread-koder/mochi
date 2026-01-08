package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/thread_koder/mochi/internal/database"
	"github.com/thread_koder/mochi/internal/logger"
)

// Syncs all Kubernetes resources to PostgreSQL
func SyncAllResources(ctx context.Context) error {
	log := logger.WithComponent("kubernetes")
	log.Debug().Msg("Starting full resource sync")

	// Sync namespaces first (they are referenced by other resources)
	if err := SyncNamespaces(ctx); err != nil {
		return fmt.Errorf("failed to sync namespaces: %w", err)
	}

	// Sync nodes
	if err := SyncNodes(ctx); err != nil {
		return fmt.Errorf("failed to sync nodes: %w", err)
	}

	// Sync deployments
	if err := SyncDeployments(ctx); err != nil {
		return fmt.Errorf("failed to sync deployments: %w", err)
	}

	// Sync statefulsets
	if err := SyncStatefulSets(ctx); err != nil {
		return fmt.Errorf("failed to sync statefulsets: %w", err)
	}

	// Sync daemonsets
	if err := SyncDaemonSets(ctx); err != nil {
		return fmt.Errorf("failed to sync daemonsets: %w", err)
	}

	// Sync services
	if err := SyncServices(ctx); err != nil {
		return fmt.Errorf("failed to sync services: %w", err)
	}

	// Sync endpoints
	if err := SyncEndpoints(ctx); err != nil {
		return fmt.Errorf("failed to sync endpoints: %w", err)
	}

	// Sync pods last (they reference other resources)
	if err := SyncPods(ctx); err != nil {
		return fmt.Errorf("failed to sync pods: %w", err)
	}

	// Sync containers (they reference pods)
	if err := SyncContainers(ctx); err != nil {
		return fmt.Errorf("failed to sync containers: %w", err)
	}

	log.Debug().Msg("Full resource sync completed")
	return nil
}

// Syncs namespaces to PostgreSQL
func SyncNamespaces(ctx context.Context) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("Kubernetes client not initialized")
	}

	namespaces, err := Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list namespaces: %w", err)
	}

	log.Debug().Int("count", len(namespaces.Items)).Msg("Syncing namespaces")

	dbNamespaces := make([]*database.Namespace, 0, len(namespaces.Items))
	now := time.Now()

	for _, ns := range namespaces.Items {
		labelsJSON, _ := mapToJSON(ns.Labels)
		annotationsJSON, _ := mapToJSON(ns.Annotations)

		dbNamespace := &database.Namespace{
			Name:        ns.Name,
			UID:         string(ns.UID),
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

	log.Debug().Int("count", len(dbNamespaces)).Msg("Namespaces synced successfully")
	return nil
}

// Syncs nodes to PostgreSQL
func SyncNodes(ctx context.Context) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("Kubernetes client not initialized")
	}

	nodes, err := Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	log.Debug().Int("count", len(nodes.Items)).Msg("Syncing nodes")

	dbNodes := make([]*database.Node, 0, len(nodes.Items))
	now := time.Now()

	for _, node := range nodes.Items {
		labelsJSON, _ := mapToJSON(node.Labels)
		annotationsJSON, _ := mapToJSON(node.Annotations)

		// Extract IP addresses
		var internalIP, externalIP *string
		for _, addr := range node.Status.Addresses {
			switch addr.Type {
			case corev1.NodeInternalIP:
				ip := addr.Address
				internalIP = &ip
			case corev1.NodeExternalIP:
				ip := addr.Address
				externalIP = &ip
			}
		}

		// Extract capacity and allocatable
		cpuCapacity := node.Status.Capacity[corev1.ResourceCPU]
		memoryCapacity := node.Status.Capacity[corev1.ResourceMemory]
		cpuAllocatable := node.Status.Allocatable[corev1.ResourceCPU]
		memoryAllocatable := node.Status.Allocatable[corev1.ResourceMemory]

		cpuCapacityStr := cpuCapacity.String()
		memoryCapacityStr := memoryCapacity.String()
		cpuAllocatableStr := cpuAllocatable.String()
		memoryAllocatableStr := memoryAllocatable.String()

		// Convert conditions to JSON
		conditionsJSON, _ := sliceToJSON(node.Status.Conditions)

		dbNode := &database.Node{
			Name:                    node.Name,
			UID:                     string(node.UID),
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

	log.Debug().Int("count", len(dbNodes)).Msg("Nodes synced successfully")
	return nil
}

// Syncs deployments to PostgreSQL
func SyncDeployments(ctx context.Context) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("Kubernetes client not initialized")
	}

	deployments, err := Clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	log.Debug().Int("count", len(deployments.Items)).Msg("Syncing deployments")

	dbDeployments := make([]*database.Deployment, 0, len(deployments.Items))
	now := time.Now()

	for _, dep := range deployments.Items {
		labelsJSON, _ := mapToJSON(dep.Labels)
		annotationsJSON, _ := mapToJSON(dep.Annotations)

		replicas := int32(0)
		if dep.Spec.Replicas != nil {
			replicas = *dep.Spec.Replicas
		}

		dbDeployment := &database.Deployment{
			Name:              dep.Name,
			Namespace:         dep.Namespace,
			UID:               string(dep.UID),
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

	log.Debug().Int("count", len(dbDeployments)).Msg("Deployments synced successfully")
	return nil
}

// Syncs statefulsets to PostgreSQL
func SyncStatefulSets(ctx context.Context) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("Kubernetes client not initialized")
	}

	statefulsets, err := Clientset.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list statefulsets: %w", err)
	}

	log.Debug().Int("count", len(statefulsets.Items)).Msg("Syncing statefulsets")

	dbStatefulSets := make([]*database.StatefulSet, 0, len(statefulsets.Items))
	now := time.Now()

	for _, sts := range statefulsets.Items {
		labelsJSON, _ := mapToJSON(sts.Labels)
		annotationsJSON, _ := mapToJSON(sts.Annotations)

		replicas := int32(0)
		if sts.Spec.Replicas != nil {
			replicas = *sts.Spec.Replicas
		}

		dbStatefulSet := &database.StatefulSet{
			Name:          sts.Name,
			Namespace:     sts.Namespace,
			UID:           string(sts.UID),
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

	log.Debug().Int("count", len(dbStatefulSets)).Msg("Statefulsets synced successfully")
	return nil
}

// Syncs daemonsets to PostgreSQL
func SyncDaemonSets(ctx context.Context) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("Kubernetes client not initialized")
	}

	daemonsets, err := Clientset.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list daemonsets: %w", err)
	}

	log.Debug().Int("count", len(daemonsets.Items)).Msg("Syncing daemonsets")

	dbDaemonSets := make([]*database.DaemonSet, 0, len(daemonsets.Items))
	now := time.Now()

	for _, ds := range daemonsets.Items {
		labelsJSON, _ := mapToJSON(ds.Labels)
		annotationsJSON, _ := mapToJSON(ds.Annotations)

		dbDaemonSet := &database.DaemonSet{
			Name:                   ds.Name,
			Namespace:              ds.Namespace,
			UID:                    string(ds.UID),
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

	log.Debug().Int("count", len(dbDaemonSets)).Msg("Daemonsets synced successfully")
	return nil
}

// Syncs services to PostgreSQL
func SyncServices(ctx context.Context) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("Kubernetes client not initialized")
	}

	services, err := Clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	log.Debug().Int("count", len(services.Items)).Msg("Syncing services")

	dbServices := make([]*database.Service, 0, len(services.Items))
	now := time.Now()

	for _, svc := range services.Items {
		labelsJSON, _ := mapToJSON(svc.Labels)
		annotationsJSON, _ := mapToJSON(svc.Annotations)
		selectorJSON, _ := mapToJSON(svc.Spec.Selector)
		portsJSON, _ := sliceToJSON(svc.Spec.Ports)

		clusterIP := svc.Spec.ClusterIP
		var clusterIPPtr *string
		if clusterIP != "" {
			clusterIPPtr = &clusterIP
		}

		dbService := &database.Service{
			Name:        svc.Name,
			Namespace:   svc.Namespace,
			UID:         string(svc.UID),
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

	log.Debug().Int("count", len(dbServices)).Msg("Services synced successfully")
	return nil
}

// Syncs endpoints to PostgreSQL
func SyncEndpoints(ctx context.Context) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("Kubernetes client not initialized")
	}

	endpoints, err := Clientset.CoreV1().Endpoints("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list endpoints: %w", err)
	}

	log.Debug().Int("count", len(endpoints.Items)).Msg("Syncing endpoints")

	dbEndpoints := make([]*database.Endpoint, 0, len(endpoints.Items))
	now := time.Now()

	for _, ep := range endpoints.Items {
		labelsJSON, _ := mapToJSON(ep.Labels)
		annotationsJSON, _ := mapToJSON(ep.Annotations)
		// Extract addresses and ports from subsets
		var addresses []string
		var ports []corev1.EndpointPort
		for _, subset := range ep.Subsets {
			for _, addr := range subset.Addresses {
				addresses = append(addresses, addr.IP)
			}
			ports = append(ports, subset.Ports...)
		}
		addressesJSON, _ := sliceToJSON(addresses)
		portsJSON, _ := sliceToJSON(ports)

		dbEndpoint := &database.Endpoint{
			Name:        ep.Name,
			Namespace:   ep.Namespace,
			UID:         string(ep.UID),
			Addresses:   addressesJSON,
			Ports:       portsJSON,
			Labels:      labelsJSON,
			Annotations: annotationsJSON,
			CreatedAt:   ep.CreationTimestamp.Time,
			SyncedAt:    now,
		}

		dbEndpoints = append(dbEndpoints, dbEndpoint)
	}

	if err := database.UpsertEndpointsBatch(ctx, dbEndpoints); err != nil {
		return fmt.Errorf("failed to upsert endpoints: %w", err)
	}

	log.Debug().Int("count", len(dbEndpoints)).Msg("Endpoints synced successfully")
	return nil
}

// Syncs pods to PostgreSQL
func SyncPods(ctx context.Context) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("Kubernetes client not initialized")
	}

	pods, err := Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	log.Debug().Int("count", len(pods.Items)).Msg("Syncing pods")

	dbPods := make([]*database.Pod, 0, len(pods.Items))
	now := time.Now()

	for _, pod := range pods.Items {
		labelsJSON, _ := mapToJSON(pod.Labels)
		annotationsJSON, _ := mapToJSON(pod.Annotations)

		// Extract owner information
		var ownerKind, ownerName *string
		for _, owner := range pod.OwnerReferences {
			kind := owner.Kind
			name := owner.Name
			ownerKind = &kind
			ownerName = &name
			break // Take first owner
		}

		restartPolicy := string(pod.Spec.RestartPolicy)
		var nodeName *string
		if pod.Spec.NodeName != "" {
			n := pod.Spec.NodeName
			nodeName = &n
		}

		dbPod := &database.Pod{
			Name:          pod.Name,
			Namespace:     pod.Namespace,
			UID:           string(pod.UID),
			NodeName:      nodeName,
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

	log.Debug().Int("count", len(dbPods)).Msg("Pods synced successfully")
	return nil
}

// Syncs containers to PostgreSQL
func SyncContainers(ctx context.Context) error {
	log := logger.WithComponent("kubernetes")

	if Clientset == nil {
		return fmt.Errorf("Kubernetes client not initialized")
	}

	pods, err := Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	dbContainers := make([]*database.Container, 0)
	now := time.Now()

	for _, pod := range pods.Items {
		podUID := string(pod.UID)
		podName := pod.Name
		namespace := pod.Namespace

		// Extract containers from pod spec
		for _, container := range pod.Spec.Containers {
			var cpuRequest, cpuLimit, memoryRequest, memoryLimit *string
			var imagePullPolicy *string

			// Extract resource requests and limits
			if container.Resources.Requests != nil {
				if cpu := container.Resources.Requests[corev1.ResourceCPU]; !cpu.IsZero() {
					cpuStr := cpu.String()
					cpuRequest = &cpuStr
				}
				if mem := container.Resources.Requests[corev1.ResourceMemory]; !mem.IsZero() {
					memStr := mem.String()
					memoryRequest = &memStr
				}
			}
			if container.Resources.Limits != nil {
				if cpu := container.Resources.Limits[corev1.ResourceCPU]; !cpu.IsZero() {
					cpuStr := cpu.String()
					cpuLimit = &cpuStr
				}
				if mem := container.Resources.Limits[corev1.ResourceMemory]; !mem.IsZero() {
					memStr := mem.String()
					memoryLimit = &memStr
				}
			}

			// Extract image pull policy
			if container.ImagePullPolicy != "" {
				policy := string(container.ImagePullPolicy)
				imagePullPolicy = &policy
			}

			// Extract container ports
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
		log.Debug().Int("count", len(dbContainers)).Msg("Syncing containers")
		if err := database.UpsertContainersBatch(ctx, dbContainers); err != nil {
			return fmt.Errorf("failed to upsert containers: %w", err)
		}
		log.Debug().Int("count", len(dbContainers)).Msg("Containers synced successfully")
	}

	return nil
}

// Helper function to convert map to JSON bytes
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

// Helper function to convert slice to JSON bytes
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
