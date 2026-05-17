package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"

	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/kubernetes"
	"github.com/thread_koder/mochi/core/internal/logger"
)

// Annotation keys written when applying recommendations so operators can trace origin and mode.
const (
	AnnotationManagedBy          = "mochi.io/managed-by"
	AnnotationRecommendationID   = "mochi.io/recommendation-id"
	AnnotationRecommendationMode = "mochi.io/recommendation-mode"
	AnnotationLastApplied        = "mochi.io/last-applied"
)

// ApplyRecommendation patches Deployment, StatefulSet, or DaemonSet pod templates with recommended
// requests/limits via server-side apply. Standalone Pods return an error because their spec is immutable.
func ApplyRecommendation(ctx context.Context, rec *database.ComputeRecommendation) error {
	log := logger.WithComponent("compute")
	log.Info().
		Int64("recommendation_id", rec.ID).
		Str("workload_type", rec.WorkloadType).
		Str("workload_name", rec.WorkloadName).
		Str("namespace", rec.Namespace).
		Msg("Applying compute recommendation")

	var containerRecs []ContainerRecommendation
	if err := json.Unmarshal(rec.Recommendations, &containerRecs); err != nil {
		return fmt.Errorf("failed to parse recommendations: %w", err)
	}

	if len(containerRecs) == 0 {
		return fmt.Errorf("no container recommendations to apply")
	}

	switch rec.WorkloadType {
	case "Deployment":
		return applyToDeployment(ctx, rec, containerRecs)
	case "StatefulSet":
		return applyToStatefulSet(ctx, rec, containerRecs)
	case "DaemonSet":
		return applyToDaemonSet(ctx, rec, containerRecs)
	case "Pod":
		return applyToPod()
	default:
		return fmt.Errorf("unsupported workload type: %s", rec.WorkloadType)
	}
}

func applyToDeployment(ctx context.Context, rec *database.ComputeRecommendation, containerRecs []ContainerRecommendation) error {
	if kubernetes.Clientset == nil {
		return fmt.Errorf("Kubernetes client not initialized")
	}

	_, err := kubernetes.Clientset.AppsV1().Deployments(rec.Namespace).Get(ctx, rec.WorkloadName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("deployment not found: cannot apply recommendations to non-existent workload")
		}
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	depApplyConfig := applyappsv1.Deployment(rec.WorkloadName, rec.Namespace)
	depApplyConfig.WithAnnotations(getMochiAnnotations(rec))
	podSpecConfig := buildPodSpecConfig(containerRecs)
	depSpecConfig := applyappsv1.DeploymentSpec().WithTemplate(applycorev1.PodTemplateSpec().WithSpec(podSpecConfig))
	depApplyConfig.WithSpec(depSpecConfig)

	// Force lets this field manager take ownership of conflicting container resource keys on re-apply.
	_, err = kubernetes.Clientset.AppsV1().Deployments(rec.Namespace).Apply(
		ctx,
		depApplyConfig,
		metav1.ApplyOptions{
			FieldManager: "mochi-controller",
			Force:        true,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to apply recommendation to deployment: %w", err)
	}

	return nil
}

func applyToStatefulSet(ctx context.Context, rec *database.ComputeRecommendation, containerRecs []ContainerRecommendation) error {
	if kubernetes.Clientset == nil {
		return fmt.Errorf("Kubernetes client not initialized")
	}

	_, err := kubernetes.Clientset.AppsV1().StatefulSets(rec.Namespace).Get(ctx, rec.WorkloadName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("statefulset not found: cannot apply recommendations to non-existent workload")
		}
		return fmt.Errorf("failed to get statefulset: %w", err)
	}

	stsApplyConfig := applyappsv1.StatefulSet(rec.WorkloadName, rec.Namespace)
	stsApplyConfig.WithAnnotations(getMochiAnnotations(rec))
	podSpecConfig := buildPodSpecConfig(containerRecs)
	stsSpecConfig := applyappsv1.StatefulSetSpec().WithTemplate(applycorev1.PodTemplateSpec().WithSpec(podSpecConfig))
	stsApplyConfig.WithSpec(stsSpecConfig)

	_, err = kubernetes.Clientset.AppsV1().StatefulSets(rec.Namespace).Apply(
		ctx,
		stsApplyConfig,
		metav1.ApplyOptions{
			FieldManager: "mochi-controller",
			Force:        true,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to apply recommendation to statefulset: %w", err)
	}

	return nil
}

func applyToDaemonSet(ctx context.Context, rec *database.ComputeRecommendation, containerRecs []ContainerRecommendation) error {
	if kubernetes.Clientset == nil {
		return fmt.Errorf("Kubernetes client not initialized")
	}

	_, err := kubernetes.Clientset.AppsV1().DaemonSets(rec.Namespace).Get(ctx, rec.WorkloadName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("daemonset not found: cannot apply recommendations to non-existent workload")
		}
		return fmt.Errorf("failed to get daemonset: %w", err)
	}

	dsApplyConfig := applyappsv1.DaemonSet(rec.WorkloadName, rec.Namespace)
	dsApplyConfig.WithAnnotations(getMochiAnnotations(rec))
	podSpecConfig := buildPodSpecConfig(containerRecs)
	dsSpecConfig := applyappsv1.DaemonSetSpec().WithTemplate(applycorev1.PodTemplateSpec().WithSpec(podSpecConfig))
	dsApplyConfig.WithSpec(dsSpecConfig)

	_, err = kubernetes.Clientset.AppsV1().DaemonSets(rec.Namespace).Apply(
		ctx,
		dsApplyConfig,
		metav1.ApplyOptions{
			FieldManager: "mochi-controller",
			Force:        true,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to apply recommendation to daemonset: %w", err)
	}

	return nil
}

func applyToPod() error {
	return fmt.Errorf("cannot apply recommendations to standalone pod: pods are immutable. Resource changes require pod recreation or management via Deployment/StatefulSet/DaemonSet")
}

func getMochiAnnotations(rec *database.ComputeRecommendation) map[string]string {
	return map[string]string{
		AnnotationManagedBy:          "mochi",
		AnnotationRecommendationID:   fmt.Sprintf("%d", rec.ID),
		AnnotationRecommendationMode: rec.RecommendationMode,
		AnnotationLastApplied:        time.Now().UTC().Format(time.RFC3339),
	}
}

func buildPodSpecConfig(containerRecs []ContainerRecommendation) *applycorev1.PodSpecApplyConfiguration {
	containers := make([]*applycorev1.ContainerApplyConfiguration, 0, len(containerRecs))
	for _, containerRec := range containerRecs {
		requests := corev1.ResourceList{}
		limits := corev1.ResourceList{}

		if containerRec.CPU.RecommendedRequest != nil {
			requests[corev1.ResourceCPU] = resource.MustParse(*containerRec.CPU.RecommendedRequest)
		}
		if containerRec.Memory.RecommendedRequest != nil {
			requests[corev1.ResourceMemory] = resource.MustParse(*containerRec.Memory.RecommendedRequest)
		}

		if containerRec.CPU.RecommendedLimit != nil {
			limits[corev1.ResourceCPU] = resource.MustParse(*containerRec.CPU.RecommendedLimit)
		}
		if containerRec.Memory.RecommendedLimit != nil {
			limits[corev1.ResourceMemory] = resource.MustParse(*containerRec.Memory.RecommendedLimit)
		}

		res := applycorev1.ResourceRequirements()
		if len(requests) > 0 {
			res.WithRequests(requests)
		}
		if len(limits) > 0 {
			res.WithLimits(limits)
		}

		containerApply := applycorev1.Container().WithName(containerRec.ContainerName).WithResources(res)
		containers = append(containers, containerApply)
	}
	return applycorev1.PodSpec().WithContainers(containers...)
}
