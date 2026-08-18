package dependency

import (
	"context"

	"github.com/thread_koder/mochi/core/internal/database"
)

const (
	KindDeployment  = "Deployment"
	KindStatefulSet = "StatefulSet"
	KindDaemonSet   = "DaemonSet"
	KindJob         = "Job"
	KindCronJob     = "CronJob"
	KindPod         = "Pod"
	KindExternal    = "External"
)

// nodeRefFromPod maps a pod to a stable workload identity.
func nodeRefFromPod(ctx context.Context, pod *database.Pod, opts ResolveOptions) (NodeRef, bool, error) {
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
		rs, found, err := lookupReplicaSet(ctx, opts, pod.Namespace, ownerName)
		if err != nil {
			return NodeRef{}, false, err
		}
		if !found {
			return NodeRef{}, false, nil
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
	case "Job":
		job, found, err := lookupJob(ctx, opts, pod.Namespace, ownerName)
		if err != nil {
			return NodeRef{}, false, err
		}
		if !found {
			return NodeRef{}, false, nil
		}
		if job.OwnerKind != nil && job.OwnerName != nil && *job.OwnerKind == "CronJob" {
			return NodeRef{
				Kind:      KindCronJob,
				Namespace: pod.Namespace,
				Name:      *job.OwnerName,
			}, true, nil
		}
		return NodeRef{
			Kind:      KindJob,
			Namespace: pod.Namespace,
			Name:      ownerName,
		}, true, nil
	default:
		return NodeRef{}, false, nil
	}
}
