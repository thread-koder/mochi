package dependency

import (
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

// nodeRefFromPod maps a pod to a graph node from sync-time workload_kind / workload_name.
func nodeRefFromPod(pod *database.Pod) (NodeRef, bool) {
	if pod.WorkloadKind == nil {
		return NodeRef{
			Kind:      KindPod,
			Namespace: pod.Namespace,
			Name:      pod.Name,
		}, true
	}

	kind := *pod.WorkloadKind
	if kind == "Node" {
		return NodeRef{}, false
	}
	if pod.WorkloadName == nil {
		return NodeRef{}, false
	}
	name := *pod.WorkloadName

	switch kind {
	case KindDeployment, KindStatefulSet, KindDaemonSet, KindJob, KindCronJob:
		return NodeRef{
			Kind:      kind,
			Namespace: pod.Namespace,
			Name:      name,
		}, true
	default:
		return NodeRef{}, false
	}
}
