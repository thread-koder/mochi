package compute

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"github.com/thread_koder/mochi/internal/database"
	"github.com/thread_koder/mochi/internal/kubernetes"
	"github.com/thread_koder/mochi/internal/logger"
)

// Applies a compute recommendation to the target workload
func ApplyRecommendation(ctx context.Context, rec *database.ComputeRecommendation) error {
	log := logger.WithComponent("compute")
	log.Info().
		Int64("recommendation_id", rec.ID).
		Str("workload_type", rec.WorkloadType).
		Str("workload_name", rec.WorkloadName).
		Str("namespace", rec.Namespace).
		Msg("Applying compute recommendation")

	// Parse recommendations from JSON
	var containerRecs []ContainerRecommendation
	if err := json.Unmarshal(rec.Recommendations, &containerRecs); err != nil {
		return fmt.Errorf("failed to parse recommendations: %w", err)
	}

	if len(containerRecs) == 0 {
		return fmt.Errorf("no container recommendations to apply")
	}

	// Route to appropriate apply function based on workload type
	switch rec.WorkloadType {
	case "deployment":
		return applyToDeployment(ctx, rec.Namespace, rec.WorkloadName, containerRecs)
	case "statefulset":
		return applyToStatefulSet(ctx, rec.Namespace, rec.WorkloadName, containerRecs)
	case "daemonset":
		return applyToDaemonSet(ctx, rec.Namespace, rec.WorkloadName, containerRecs)
	case "pod":
		return applyToPod()
	default:
		return fmt.Errorf("unsupported workload type: %s", rec.WorkloadType)
	}
}

// Applies recommendations to a deployment
func applyToDeployment(ctx context.Context, namespace, name string, containerRecs []ContainerRecommendation) error {
	if kubernetes.Clientset == nil {
		return fmt.Errorf("Kubernetes client not initialized")
	}

	// Get the deployment
	deployment, err := kubernetes.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Create a copy for patching
	originalBytes, err := json.Marshal(deployment)
	if err != nil {
		return fmt.Errorf("failed to marshal original deployment: %w", err)
	}

	// Apply container recommendations
	if err := updateContainerResources(&deployment.Spec.Template.Spec, containerRecs); err != nil {
		return fmt.Errorf("failed to update container resources: %w", err)
	}

	// Create patch
	modifiedBytes, err := json.Marshal(deployment)
	if err != nil {
		return fmt.Errorf("failed to marshal modified deployment: %w", err)
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(originalBytes, modifiedBytes, deployment)
	if err != nil {
		return fmt.Errorf("failed to create patch: %w", err)
	}

	// Apply the patch
	_, err = kubernetes.Clientset.AppsV1().Deployments(namespace).Patch(
		ctx,
		name,
		types.StrategicMergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to patch deployment: %w", err)
	}

	return nil
}

// Applies recommendations to a statefulset
func applyToStatefulSet(ctx context.Context, namespace, name string, containerRecs []ContainerRecommendation) error {
	if kubernetes.Clientset == nil {
		return fmt.Errorf("Kubernetes client not initialized")
	}

	// Get the statefulset
	statefulSet, err := kubernetes.Clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get statefulset: %w", err)
	}

	// Create a copy for patching
	originalBytes, err := json.Marshal(statefulSet)
	if err != nil {
		return fmt.Errorf("failed to marshal original statefulset: %w", err)
	}

	// Apply container recommendations
	if err := updateContainerResources(&statefulSet.Spec.Template.Spec, containerRecs); err != nil {
		return fmt.Errorf("failed to update container resources: %w", err)
	}

	// Create patch
	modifiedBytes, err := json.Marshal(statefulSet)
	if err != nil {
		return fmt.Errorf("failed to marshal modified statefulset: %w", err)
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(originalBytes, modifiedBytes, statefulSet)
	if err != nil {
		return fmt.Errorf("failed to create patch: %w", err)
	}

	// Apply the patch
	_, err = kubernetes.Clientset.AppsV1().StatefulSets(namespace).Patch(
		ctx,
		name,
		types.StrategicMergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to patch statefulset: %w", err)
	}

	return nil
}

// Applies recommendations to a daemonset
func applyToDaemonSet(ctx context.Context, namespace, name string, containerRecs []ContainerRecommendation) error {
	if kubernetes.Clientset == nil {
		return fmt.Errorf("Kubernetes client not initialized")
	}

	// Get the daemonset
	daemonSet, err := kubernetes.Clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get daemonset: %w", err)
	}

	// Create a copy for patching
	originalBytes, err := json.Marshal(daemonSet)
	if err != nil {
		return fmt.Errorf("failed to marshal original daemonset: %w", err)
	}

	// Apply container recommendations
	if err := updateContainerResources(&daemonSet.Spec.Template.Spec, containerRecs); err != nil {
		return fmt.Errorf("failed to update container resources: %w", err)
	}

	// Create patch
	modifiedBytes, err := json.Marshal(daemonSet)
	if err != nil {
		return fmt.Errorf("failed to marshal modified daemonset: %w", err)
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(originalBytes, modifiedBytes, daemonSet)
	if err != nil {
		return fmt.Errorf("failed to create patch: %w", err)
	}

	// Apply the patch
	_, err = kubernetes.Clientset.AppsV1().DaemonSets(namespace).Patch(
		ctx,
		name,
		types.StrategicMergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to patch daemonset: %w", err)
	}

	return nil
}

// Handles pod recommendation application
// Pods are immutable, return an error explaining the limitation
func applyToPod() error {
	return fmt.Errorf("cannot apply recommendations to standalone pods: pods are immutable and cannot be patched. To apply resource changes, you must delete and recreate the pod manually, or use a Deployment/StatefulSet/DaemonSet instead")
}

// Updates container resources in a pod spec based on recommendations
func updateContainerResources(podSpec *corev1.PodSpec, containerRecs []ContainerRecommendation) error {
	// Create a map of container name to recommendation for quick lookup
	recMap := make(map[string]ContainerRecommendation)
	for _, rec := range containerRecs {
		recMap[rec.ContainerName] = rec
	}

	// Update each container in the pod spec
	for i := range podSpec.Containers {
		container := &podSpec.Containers[i]
		rec, exists := recMap[container.Name]
		if !exists {
			continue // Skip containers without recommendations
		}

		// Initialize resources if nil
		if container.Resources.Requests == nil {
			container.Resources.Requests = make(corev1.ResourceList)
		}
		if container.Resources.Limits == nil {
			container.Resources.Limits = make(corev1.ResourceList)
		}

		// Apply CPU recommendations
		if rec.CPU.RecommendedRequest != nil {
			qty, err := resource.ParseQuantity(*rec.CPU.RecommendedRequest)
			if err != nil {
				return fmt.Errorf("failed to parse CPU request %s for container %s: %w", *rec.CPU.RecommendedRequest, container.Name, err)
			}
			container.Resources.Requests[corev1.ResourceCPU] = qty
		}

		if rec.CPU.RecommendedLimit != nil {
			qty, err := resource.ParseQuantity(*rec.CPU.RecommendedLimit)
			if err != nil {
				return fmt.Errorf("failed to parse CPU limit %s for container %s: %w", *rec.CPU.RecommendedLimit, container.Name, err)
			}
			container.Resources.Limits[corev1.ResourceCPU] = qty
		}

		// Apply memory recommendations
		if rec.Memory.RecommendedRequest != nil {
			qty, err := resource.ParseQuantity(*rec.Memory.RecommendedRequest)
			if err != nil {
				return fmt.Errorf("failed to parse memory request %s for container %s: %w", *rec.Memory.RecommendedRequest, container.Name, err)
			}
			container.Resources.Requests[corev1.ResourceMemory] = qty
		}

		if rec.Memory.RecommendedLimit != nil {
			qty, err := resource.ParseQuantity(*rec.Memory.RecommendedLimit)
			if err != nil {
				return fmt.Errorf("failed to parse memory limit %s for container %s: %w", *rec.Memory.RecommendedLimit, container.Name, err)
			}
			container.Resources.Limits[corev1.ResourceMemory] = qty
		}
	}

	return nil
}
