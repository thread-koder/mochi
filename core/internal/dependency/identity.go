package dependency

import (
	"context"
	"errors"
	"fmt"

	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/database"
)

const (
	KindDeployment  = "Deployment"
	KindStatefulSet = "StatefulSet"
	KindDaemonSet   = "DaemonSet"
	KindPod         = "Pod"
	KindExternal    = "External"
)

// nodeRefFromPod maps a pod to a stable workload identity.
func nodeRefFromPod(ctx context.Context, pod *database.Pod) (NodeRef, bool, error) {
	if pod == nil {
		return NodeRef{}, false, fmt.Errorf("pod cannot be nil")
	}

	if pod.OwnerKind == nil || pod.OwnerName == nil {
		return NodeRef{
			Kind:      KindPod,
			Namespace: pod.Namespace,
			Name:      pod.Name,
		}, true, nil
	}

	ownerKind := *pod.OwnerKind
	ownerName := *pod.OwnerName

	if ownerKind == "Node" {
		return NodeRef{}, false, nil
	}

	switch ownerKind {
	case "ReplicaSet":
		rs, err := database.GetReplicaSetByName(ctx, ownerName, pod.Namespace)
		if err != nil {
			if errors.Is(err, &apperrors.NotFoundError{}) {
				return NodeRef{}, false, nil
			}
			return NodeRef{}, false, fmt.Errorf("resolve ReplicaSet owner for pod %s: %w", pod.UID, err)
		}
		if rs.OwnerKind != nil && rs.OwnerName != nil && *rs.OwnerKind == "Deployment" {
			return NodeRef{
				Kind:      KindDeployment,
				Namespace: pod.Namespace,
				Name:      *rs.OwnerName,
			}, true, nil
		}
		return NodeRef{}, false, nil
	case "StatefulSet":
		return NodeRef{
			Kind:      KindStatefulSet,
			Namespace: pod.Namespace,
			Name:      ownerName,
		}, true, nil
	case "DaemonSet":
		return NodeRef{
			Kind:      KindDaemonSet,
			Namespace: pod.Namespace,
			Name:      ownerName,
		}, true, nil
	default:
		// Job/CronJob and other controllers: treat as Pod-named node.
		return NodeRef{
			Kind:      KindPod,
			Namespace: pod.Namespace,
			Name:      pod.Name,
		}, true, nil
	}
}
