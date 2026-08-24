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
		{"statefulsets", syncStatefulSets},
		{"daemonsets", syncDaemonSets},
		{"cronjobs", syncCronJobs},
		{"services", syncServices},
		{"endpoint slices", syncEndpointSlices},
	}
	for _, kind := range kinds {
		if err := kind.fn(ctx, namespace); err != nil {
			log.Warn().Err(err).Str("namespace", namespace).Str("kind", kind.name).Msg("Namespace sync stopped")
			return
		}
	}

	replicaSetToDeployment, err := indexReplicaSetDeployments(ctx, namespace)
	if err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Str("kind", "replicasets").Msg("Namespace sync stopped")
		return
	}

	jobToCronJob, err := syncJobs(ctx, namespace)
	if err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Str("kind", "jobs").Msg("Namespace sync stopped")
		return
	}

	pods, err := syncPods(ctx, namespace, replicaSetToDeployment, jobToCronJob)
	if err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Str("kind", "pods").Msg("Namespace sync stopped")
		return
	}

	if err := syncContainers(ctx, namespace, pods); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Str("kind", "containers").Msg("Namespace sync stopped")
		return
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

// indexReplicaSetDeployments lists ReplicaSets and returns ReplicaSet name -> Deployment name.
// ReplicaSets are not persisted.
func indexReplicaSetDeployments(ctx context.Context, namespace string) (map[string]string, error) {
	replicasets, err := Clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list replicasets: %w", err)
	}

	replicaSetToDeployment := make(map[string]string)
	for i := range replicasets.Items {
		rs := &replicasets.Items[i]
		kind, name := primaryOwner(rs)
		if kind == "Deployment" {
			replicaSetToDeployment[rs.Name] = name
		}
	}
	return replicaSetToDeployment, nil
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

func syncCronJobs(ctx context.Context, namespace string) error {
	log := logger.WithComponent("kubernetes")

	cronjobs, err := Clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list cronjobs: %w", err)
	}

	dbCronJobs := make([]*database.CronJob, 0, len(cronjobs.Items))
	now := time.Now()

	cjUIDs := make([]string, 0, len(cronjobs.Items))
	for _, cj := range cronjobs.Items {
		cjUID := string(cj.UID)
		cjUIDs = append(cjUIDs, cjUID)
		labelsJSON, _ := mapToJSON(cj.Labels)
		annotationsJSON, _ := mapToJSON(cj.Annotations)

		suspend := false
		if cj.Spec.Suspend != nil {
			suspend = *cj.Spec.Suspend
		}

		dbCronJob := &database.CronJob{
			Name:        cj.Name,
			Namespace:   cj.Namespace,
			UID:         cjUID,
			Schedule:    cj.Spec.Schedule,
			Suspend:     suspend,
			Labels:      labelsJSON,
			Annotations: annotationsJSON,
			CreatedAt:   cj.CreationTimestamp.Time,
			SyncedAt:    now,
		}

		dbCronJobs = append(dbCronJobs, dbCronJob)
	}

	if err := database.UpsertCronJobsBatch(ctx, dbCronJobs); err != nil {
		return err
	}

	if err := database.PruneCronJobs(ctx, namespace, cjUIDs); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune cronjobs not in current state")
	} else if err := database.PruneComputeRecommendations(ctx, namespace, "CronJob"); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune compute recommendations for deleted cronjobs")
	}
	return nil
}

// syncJobs persists standalone Jobs only and returns job name → CronJob name for climb.
func syncJobs(ctx context.Context, namespace string) (map[string]string, error) {
	log := logger.WithComponent("kubernetes")

	jobs, err := Clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	jobToCronJob := make(map[string]string)
	dbJobs := make([]*database.Job, 0)
	now := time.Now()
	jobUIDs := make([]string, 0)

	for i := range jobs.Items {
		job := &jobs.Items[i]
		kind, name := primaryOwner(job)
		if kind == "CronJob" {
			jobToCronJob[job.Name] = name
			continue
		}

		jobUID := string(job.UID)
		jobUIDs = append(jobUIDs, jobUID)
		labelsJSON, _ := mapToJSON(job.Labels)
		annotationsJSON, _ := mapToJSON(job.Annotations)

		dbJobs = append(dbJobs, &database.Job{
			Name:        job.Name,
			Namespace:   job.Namespace,
			UID:         jobUID,
			Active:      int(job.Status.Active),
			Succeeded:   int(job.Status.Succeeded),
			Failed:      int(job.Status.Failed),
			Labels:      labelsJSON,
			Annotations: annotationsJSON,
			CreatedAt:   job.CreationTimestamp.Time,
			SyncedAt:    now,
		})
	}

	if err := database.UpsertJobsBatch(ctx, dbJobs); err != nil {
		return nil, err
	}

	if err := database.PruneJobs(ctx, namespace, jobUIDs); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune jobs not in current state")
	} else if err := database.PruneComputeRecommendations(ctx, namespace, "Job"); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune compute recommendations for deleted jobs")
	}
	return jobToCronJob, nil
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
	for i := range endpointSlices.Items {
		es := &endpointSlices.Items[i]
		esUID := string(es.UID)
		esUIDs = append(esUIDs, esUID)
		labelsJSON, _ := mapToJSON(es.Labels)
		annotationsJSON, _ := mapToJSON(es.Annotations)

		ownerKind, ownerName := primaryOwner(es)

		endpointsJSON, _ := sliceToJSON(es.Endpoints)
		portsJSON, _ := sliceToJSON(es.Ports)

		dbEndpointSlices = append(dbEndpointSlices, &database.EndpointSlice{
			Name:        es.Name,
			Namespace:   es.Namespace,
			UID:         esUID,
			AddressType: string(es.AddressType),
			OwnerKind:   optionalString(ownerKind),
			OwnerName:   optionalString(ownerName),
			Endpoints:   endpointsJSON,
			Ports:       portsJSON,
			Labels:      labelsJSON,
			Annotations: annotationsJSON,
			CreatedAt:   es.CreationTimestamp.Time,
			SyncedAt:    now,
		})
	}

	if err := database.UpsertEndpointSlicesBatch(ctx, dbEndpointSlices); err != nil {
		return err
	}

	if err := database.PruneEndpointSlices(ctx, namespace, esUIDs); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune endpoint slices not in current state")
	}
	return nil
}

func syncPods(ctx context.Context, namespace string, replicaSetToDeployment, jobToCronJob map[string]string) ([]corev1.Pod, error) {
	log := logger.WithComponent("kubernetes")

	pods, err := Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	dbPods := make([]*database.Pod, 0, len(pods.Items))
	attributions := make([]*database.PodAttribution, 0, len(pods.Items))
	now := time.Now()

	podUIDs := make([]string, 0, len(pods.Items))
	for i := range pods.Items {
		pod := &pods.Items[i]
		podUID := string(pod.UID)
		podUIDs = append(podUIDs, podUID)
		labelsJSON, _ := mapToJSON(pod.Labels)
		annotationsJSON, _ := mapToJSON(pod.Annotations)

		controllerKind, controllerName := primaryOwner(pod)
		workloadKind, workloadName := workloadIdentity(controllerKind, controllerName, replicaSetToDeployment, jobToCronJob)

		dbPods = append(dbPods, &database.Pod{
			Name:          pod.Name,
			Namespace:     pod.Namespace,
			UID:           podUID,
			Node:          optionalString(pod.Spec.NodeName),
			PodIP:         optionalString(pod.Status.PodIP),
			Phase:         string(pod.Status.Phase),
			RestartPolicy: string(pod.Spec.RestartPolicy),
			Labels:        labelsJSON,
			Annotations:   annotationsJSON,
			WorkloadKind:  workloadKind,
			WorkloadName:  workloadName,
			CreatedAt:     pod.CreationTimestamp.Time,
			SyncedAt:      now,
		})

		if !isAttributableWorkload(workloadKind) {
			continue
		}

		containersJSON, err := attributionContainersJSON(pod)
		if err != nil {
			return nil, fmt.Errorf("failed to encode attribution containers for pod %s: %w", pod.Name, err)
		}

		attr := &database.PodAttribution{
			UID:          podUID,
			Name:         pod.Name,
			Namespace:    pod.Namespace,
			WorkloadKind: workloadKind,
			WorkloadName: workloadName,
			Phase:        string(pod.Status.Phase),
			Node:         optionalString(pod.Spec.NodeName),
			Containers:   containersJSON,
			FirstSeenAt:  now,
			LastSeenAt:   now,
		}
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			attr.FinishedAt = &now
		}
		attributions = append(attributions, attr)
	}

	if err := database.UpsertPodsBatch(ctx, dbPods); err != nil {
		return nil, err
	}

	if err := database.UpsertPodAttributionsBatch(ctx, attributions); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to upsert pod attributions")
	} else if err := database.FinishUnlistedPodAttributions(ctx, namespace, podUIDs, now); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to finish unlisted pod attributions")
	}

	if err := database.PrunePods(ctx, namespace, podUIDs); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune pods not in current state")
	} else if err := database.PruneComputeRecommendations(ctx, namespace, "Pod"); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune compute recommendations for deleted pods")
	}
	return pods.Items, nil
}

func syncContainers(ctx context.Context, namespace string, pods []corev1.Pod) error {
	log := logger.WithComponent("kubernetes")

	dbContainers := make([]*database.Container, 0)
	now := time.Now()

	podUIDs := make([]string, 0, len(pods))
	for _, pod := range pods {
		podUID := string(pod.UID)
		podUIDs = append(podUIDs, podUID)

		for _, container := range pod.Spec.Containers {
			cpuRequest, cpuLimit, memoryRequest, memoryLimit := containerResourceStrings(container)
			portsJSON, _ := sliceToJSON(container.Ports)

			dbContainers = append(dbContainers, &database.Container{
				Name:            container.Name,
				PodUID:          podUID,
				PodName:         pod.Name,
				Namespace:       pod.Namespace,
				Image:           container.Image,
				ImagePullPolicy: string(container.ImagePullPolicy),
				Ports:           portsJSON,
				CPURequest:      cpuRequest,
				CPULimit:        cpuLimit,
				MemoryRequest:   memoryRequest,
				MemoryLimit:     memoryLimit,
				CreatedAt:       pod.CreationTimestamp.Time,
				SyncedAt:        now,
			})
		}
	}

	if err := database.UpsertContainersBatch(ctx, dbContainers); err != nil {
		return err
	}

	if err := database.PruneContainers(ctx, namespace, podUIDs); err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to prune containers not in current state")
	}
	return nil
}

func containerResourceStrings(container corev1.Container) (cpuRequest, cpuLimit, memoryRequest, memoryLimit *string) {
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
	return cpuRequest, cpuLimit, memoryRequest, memoryLimit
}

func attributionContainersJSON(pod *corev1.Pod) (json.RawMessage, error) {
	specs := make([]database.AttributionContainerSpec, 0, len(pod.Spec.Containers))
	for _, container := range pod.Spec.Containers {
		cpuRequest, cpuLimit, memoryRequest, memoryLimit := containerResourceStrings(container)
		specs = append(specs, database.AttributionContainerSpec{
			Name:          container.Name,
			Image:         container.Image,
			CPURequest:    cpuRequest,
			CPULimit:      cpuLimit,
			MemoryRequest: memoryRequest,
			MemoryLimit:   memoryLimit,
		})
	}
	data, err := json.Marshal(specs)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// isAttributableWorkload matches first-class kinds plus standalone (nil kind).
func isAttributableWorkload(kind *string) bool {
	if kind == nil {
		return true
	}
	switch *kind {
	case "Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob":
		return true
	default:
		return false
	}
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return new(s)
}

// primaryOwner returns the controller OwnerReference, or the first ref if none is marked controller.
func primaryOwner(obj metav1.Object) (kind, name string) {
	if ref := metav1.GetControllerOf(obj); ref != nil {
		return ref.Kind, ref.Name
	}
	refs := obj.GetOwnerReferences()
	if len(refs) == 0 {
		return "", ""
	}
	return refs[0].Kind, refs[0].Name
}

// workloadIdentity climbs ReplicaSet->Deployment and Job->CronJob for Mochi workload identity.
func workloadIdentity(controllerKind, controllerName string, replicaSetToDeployment, jobToCronJob map[string]string) (kind, name *string) {
	if controllerKind == "" {
		return nil, nil
	}
	switch controllerKind {
	case "ReplicaSet":
		if deploy, ok := replicaSetToDeployment[controllerName]; ok {
			return optionalString("Deployment"), optionalString(deploy)
		}
		return optionalString(controllerKind), optionalString(controllerName)
	case "Job":
		if cron, ok := jobToCronJob[controllerName]; ok {
			return optionalString("CronJob"), optionalString(cron)
		}
		return optionalString(controllerKind), optionalString(controllerName)
	default:
		return optionalString(controllerKind), optionalString(controllerName)
	}
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
	data, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal slice to JSON: %w", err)
	}
	if string(data) == "null" {
		return json.RawMessage("[]"), nil
	}
	return json.RawMessage(data), nil
}
