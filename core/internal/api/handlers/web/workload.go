package web

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/core/internal/api/handlers/common"
	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/database"
	"golang.org/x/sync/errgroup"
)

type WorkloadStatus struct {
	Replicas  *int    `json:"replicas,omitempty"`
	Ready     *int    `json:"ready,omitempty"`
	Active    *int    `json:"active,omitempty"`
	Succeeded *int    `json:"succeeded,omitempty"`
	Failed    *int    `json:"failed,omitempty"`
	Schedule  *string `json:"schedule,omitempty"`
	Suspend   *bool   `json:"suspend,omitempty"`
	Phase     *string `json:"phase,omitempty"`
}

type WorkloadResponse struct {
	Namespace  string         `json:"namespace"`
	Type       string         `json:"type"`
	Name       string         `json:"name"`
	Status     WorkloadStatus `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	Pods       []Pod          `json:"pods"`
	Containers []Container    `json:"containers"`
	Stats      WorkloadStats  `json:"stats"`
}

type WorkloadStats struct {
	Pods       int `json:"pods"`
	Containers int `json:"containers"`
}

type Pod struct {
	Name      string    `json:"name"`
	UID       string    `json:"uid"`
	Phase     string    `json:"phase"`
	Node      string    `json:"node"`
	CreatedAt time.Time `json:"created_at"`
}

// Container describes one unique container included in workload details.
type Container struct {
	Name          string `json:"name"`
	Image         string `json:"image"`
	CPURequest    string `json:"cpu_request"`
	CPULimit      string `json:"cpu_limit"`
	MemoryRequest string `json:"memory_request"`
	MemoryLimit   string `json:"memory_limit"`
}

func GetWorkload(c *gin.Context) {
	namespaceName := c.Param("namespace")
	workloadType := c.Param("type")
	workloadName := c.Param("name")

	if !common.ValidateWorkloadType(c, workloadType) {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	response := WorkloadResponse{
		Namespace: namespaceName,
		Type:      workloadType,
		Name:      workloadName,
	}

	var pods []*database.Pod
	switch workloadType {
	case "Deployment":
		dep, err := database.GetDeploymentByName(ctx, workloadName, namespaceName)
		if err != nil {
			c.Error(err)
			if errors.Is(err, &apperrors.NotFoundError{}) {
				common.WriteNotFoundError(c, "deployment_not_found", "Deployment not found.")
			} else {
				common.WriteInternalError(c, "Failed to get deployment.")
			}
			return
		}
		response.Status = WorkloadStatus{
			Replicas: new(dep.Replicas),
			Ready:    new(dep.ReadyReplicas),
		}
		response.CreatedAt = dep.CreatedAt
		pods, err = database.GetPodsByWorkload(ctx, "Deployment", workloadName, namespaceName)
		if err != nil {
			c.Error(err)
		}

	case "StatefulSet":
		sts, err := database.GetStatefulSetByName(ctx, workloadName, namespaceName)
		if err != nil {
			c.Error(err)
			if errors.Is(err, &apperrors.NotFoundError{}) {
				common.WriteNotFoundError(c, "statefulset_not_found", "StatefulSet not found.")
			} else {
				common.WriteInternalError(c, "Failed to get StatefulSet.")
			}
			return
		}
		response.Status = WorkloadStatus{
			Replicas: new(sts.Replicas),
			Ready:    new(sts.ReadyReplicas),
		}
		response.CreatedAt = sts.CreatedAt
		pods, err = database.GetPodsByWorkload(ctx, "StatefulSet", workloadName, namespaceName)
		if err != nil {
			c.Error(err)
		}

	case "DaemonSet":
		ds, err := database.GetDaemonSetByName(ctx, workloadName, namespaceName)
		if err != nil {
			c.Error(err)
			if errors.Is(err, &apperrors.NotFoundError{}) {
				common.WriteNotFoundError(c, "daemonset_not_found", "DaemonSet not found.")
			} else {
				common.WriteInternalError(c, "Failed to get DaemonSet.")
			}
			return
		}
		response.Status = WorkloadStatus{
			Replicas: new(ds.DesiredNumberScheduled),
			Ready:    new(ds.NumberReady),
		}
		response.CreatedAt = ds.CreatedAt
		pods, err = database.GetPodsByWorkload(ctx, "DaemonSet", workloadName, namespaceName)
		if err != nil {
			c.Error(err)
		}

	case "Job":
		job, err := database.GetJobByName(ctx, workloadName, namespaceName)
		if err != nil {
			c.Error(err)
			if errors.Is(err, &apperrors.NotFoundError{}) {
				common.WriteNotFoundError(c, "job_not_found", "Job not found.")
			} else {
				common.WriteInternalError(c, "Failed to get Job.")
			}
			return
		}
		response.Status = WorkloadStatus{
			Active:    new(job.Active),
			Succeeded: new(job.Succeeded),
			Failed:    new(job.Failed),
		}
		response.CreatedAt = job.CreatedAt
		pods, err = database.GetPodsByWorkload(ctx, "Job", workloadName, namespaceName)
		if err != nil {
			c.Error(err)
		}

	case "CronJob":
		cj, err := database.GetCronJobByName(ctx, workloadName, namespaceName)
		if err != nil {
			c.Error(err)
			if errors.Is(err, &apperrors.NotFoundError{}) {
				common.WriteNotFoundError(c, "cronjob_not_found", "CronJob not found.")
			} else {
				common.WriteInternalError(c, "Failed to get CronJob.")
			}
			return
		}
		response.Status = WorkloadStatus{
			Schedule: new(cj.Schedule),
			Suspend:  new(cj.Suspend),
		}
		response.CreatedAt = cj.CreatedAt
		pods, err = database.GetPodsByWorkload(ctx, "CronJob", workloadName, namespaceName)
		if err != nil {
			c.Error(err)
		}

	case "Pod":
		pod, err := database.GetPodByName(ctx, workloadName, namespaceName)
		if err != nil {
			c.Error(err)
			if errors.Is(err, &apperrors.NotFoundError{}) {
				common.WriteNotFoundError(c, "pod_not_found", "Pod not found.")
			} else {
				common.WriteInternalError(c, "Failed to get pod.")
			}
			return
		}
		if !common.ValidateStandalonePod(c, pod, workloadName) {
			return
		}
		response.Status = WorkloadStatus{Phase: new(pod.Phase)}
		response.CreatedAt = pod.CreatedAt
		pods = []*database.Pod{pod}
	}

	response.Stats.Pods = len(pods)

	type podData struct {
		pod        Pod
		containers []*database.Container
	}

	results := make([]podData, len(pods))
	if len(pods) > 0 {
		g, gctx := errgroup.WithContext(ctx)
		for i, pod := range pods {
			g.Go(func() error {
				nodeName := ""
				if pod.Node != nil {
					nodeName = *pod.Node
				}
				podDetail := Pod{
					Name:      pod.Name,
					UID:       pod.UID,
					Phase:     pod.Phase,
					Node:      nodeName,
					CreatedAt: pod.CreatedAt,
				}

				containers, err := database.GetContainersByPodUID(gctx, pod.UID)
				if err != nil {
					c.Error(err)
					results[i].pod = podDetail
					return nil
				}
				results[i] = podData{pod: podDetail, containers: containers}
				return nil
			})
		}
		_ = g.Wait()
	}

	response.Pods = make([]Pod, 0, len(pods))
	uniqueContainers := make(map[string]Container)
	for _, item := range results {
		response.Pods = append(response.Pods, item.pod)
		for _, container := range item.containers {
			if _, exists := uniqueContainers[container.Name]; !exists {
				cpuRequest := ""
				if container.CPURequest != nil {
					cpuRequest = *container.CPURequest
				}
				cpuLimit := ""
				if container.CPULimit != nil {
					cpuLimit = *container.CPULimit
				}
				memoryRequest := ""
				if container.MemoryRequest != nil {
					memoryRequest = *container.MemoryRequest
				}
				memoryLimit := ""
				if container.MemoryLimit != nil {
					memoryLimit = *container.MemoryLimit
				}

				uniqueContainers[container.Name] = Container{
					Name:          container.Name,
					Image:         container.Image,
					CPURequest:    cpuRequest,
					CPULimit:      cpuLimit,
					MemoryRequest: memoryRequest,
					MemoryLimit:   memoryLimit,
				}
			}
		}
	}

	response.Containers = make([]Container, 0, len(uniqueContainers))
	for _, container := range uniqueContainers {
		response.Containers = append(response.Containers, container)
	}
	response.Stats.Containers = len(response.Containers)

	c.JSON(http.StatusOK, response)
}
