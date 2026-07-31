package common

import (
	"context"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/database"
)

var supportedWorkloadTypes = map[string]bool{
	"Deployment":  true,
	"StatefulSet": true,
	"DaemonSet":   true,
	"Pod":         true,
}

func IsValidWorkloadType(workloadType string) bool {
	_, ok := supportedWorkloadTypes[workloadType]
	return ok
}

func ValidateWorkloadType(c *gin.Context, workloadType string) bool {
	if IsValidWorkloadType(workloadType) {
		return true
	}
	err := fmt.Errorf("workload type must be one of: Deployment, StatefulSet, DaemonSet, Pod")
	c.Error(err)
	WriteValidationError(c, "invalid_workload_type", "Workload type must be one of Deployment, StatefulSet, DaemonSet, or Pod.")
	return false
}

// ValidateStandalonePod rejects controller-owned pods so callers use the owning workload route.
func ValidateStandalonePod(c *gin.Context, pod *database.Pod, podName string) bool {
	if pod.OwnerKind == nil || *pod.OwnerKind == "Node" {
		return true
	}
	err := fmt.Errorf("pod %s belongs to %s/%s, use workload endpoint with type %s instead",
		podName, *pod.OwnerKind, *pod.OwnerName, *pod.OwnerKind)
	c.Error(err)
	WriteValidationError(c, "pod_belongs_to_workload", "Requested pod is managed by a workload. Use the owning workload endpoint.")
	return false
}

// EnsureWorkloadExists verifies the database record exists.
// For Pod, returns the loaded pod, and for controllers returns nil pod on success.
func EnsureWorkloadExists(c *gin.Context, ctx context.Context, workloadType, workloadName, namespace string) (*database.Pod, bool) {
	if workloadType == "Pod" {
		pod, err := database.GetPodByName(ctx, workloadName, namespace)
		if err != nil {
			c.Error(err)
			if errors.Is(err, &apperrors.NotFoundError{}) {
				WriteNotFoundError(c, "pod_not_found", "Pod not found.")
			} else {
				WriteInternalError(c, "Failed to get pod.")
			}
			return nil, false
		}
		if !ValidateStandalonePod(c, pod, workloadName) {
			return nil, false
		}
		return pod, true
	}

	switch workloadType {
	case "Deployment":
		if _, err := database.GetDeploymentByName(ctx, workloadName, namespace); err != nil {
			c.Error(err)
			if errors.Is(err, &apperrors.NotFoundError{}) {
				WriteNotFoundError(c, "deployment_not_found", "Deployment not found.")
			} else {
				WriteInternalError(c, "Failed to get deployment.")
			}
			return nil, false
		}
	case "StatefulSet":
		if _, err := database.GetStatefulSetByName(ctx, workloadName, namespace); err != nil {
			c.Error(err)
			if errors.Is(err, &apperrors.NotFoundError{}) {
				WriteNotFoundError(c, "statefulset_not_found", "StatefulSet not found.")
			} else {
				WriteInternalError(c, "Failed to get StatefulSet.")
			}
			return nil, false
		}
	case "DaemonSet":
		if _, err := database.GetDaemonSetByName(ctx, workloadName, namespace); err != nil {
			c.Error(err)
			if errors.Is(err, &apperrors.NotFoundError{}) {
				WriteNotFoundError(c, "daemonset_not_found", "DaemonSet not found.")
			} else {
				WriteInternalError(c, "Failed to get DaemonSet.")
			}
			return nil, false
		}
	}

	return nil, true
}

func ResolveWorkloadPods(c *gin.Context, ctx context.Context, workloadType, workloadName, namespace string) ([]*database.Pod, bool) {
	pod, ok := EnsureWorkloadExists(c, ctx, workloadType, workloadName, namespace)
	if !ok {
		return nil, false
	}

	if workloadType == "Pod" {
		return []*database.Pod{pod}, true
	}

	pods, err := database.GetPodsByWorkload(ctx, workloadType, workloadName, namespace)
	if err != nil {
		c.Error(err)
		WriteInternalError(c, "Failed to get pods for workload.")
		return nil, false
	}
	if len(pods) == 0 {
		err := fmt.Errorf("no pods found for workload %s/%s", workloadName, namespace)
		c.Error(err)
		WriteNotFoundError(c, "pods_not_found", "No pods found for workload.")
		return nil, false
	}

	return pods, true
}
