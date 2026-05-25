package web

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/core/internal/api/handlers/common"
	"github.com/thread_koder/mochi/core/internal/database"
	"golang.org/x/sync/errgroup"
)

// WorkloadResponse is the payload for workload detail pages.
type WorkloadResponse struct {
	Namespace  string        `json:"namespace"`
	Type       string        `json:"type"`
	Name       string        `json:"name"`
	Replicas   int           `json:"replicas"`
	Ready      int           `json:"ready"`
	CreatedAt  time.Time     `json:"created_at"`
	Pods       []Pod         `json:"pods"`
	Containers []Container   `json:"containers"`
	Stats      WorkloadStats `json:"stats"`
}

// WorkloadStats summarizes pod and container totals for a workload.
type WorkloadStats struct {
	Pods       int `json:"pods"`
	Containers int `json:"containers"`
}

// Pod describes one pod included in workload details.
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

// GetWorkload returns metadata, pods, and containers for a workload.
func GetWorkload(c *gin.Context) {
	namespaceName := c.Param("namespace")
	workloadType := c.Param("type")
	workloadName := c.Param("name")

	validTypes := map[string]bool{
		"Deployment":  true,
		"StatefulSet": true,
		"DaemonSet":   true,
		"Pod":         true,
	}
	if !validTypes[workloadType] {
		err := fmt.Errorf("workload type must be one of: Deployment, StatefulSet, DaemonSet, Pod")
		c.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid workload type",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
			if common.IsNotFoundError(err) {
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "deployment not found",
					"details": err.Error(),
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "failed to get deployment",
					"details": err.Error(),
				})
			}
			return
		}
		response.Replicas = dep.Replicas
		response.Ready = dep.ReadyReplicas
		response.CreatedAt = dep.CreatedAt
		pods, err = database.GetPodsByWorkload(ctx, "Deployment", workloadName, namespaceName)
		if err != nil {
			c.Error(fmt.Errorf("failed to get pods: %w", err))
		}

	case "StatefulSet":
		sts, err := database.GetStatefulSetByName(ctx, workloadName, namespaceName)
		if err != nil {
			c.Error(err)
			if common.IsNotFoundError(err) {
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "statefulset not found",
					"details": err.Error(),
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "failed to get statefulset",
					"details": err.Error(),
				})
			}
			return
		}
		response.Replicas = sts.Replicas
		response.Ready = sts.ReadyReplicas
		response.CreatedAt = sts.CreatedAt
		pods, err = database.GetPodsByWorkload(ctx, "StatefulSet", workloadName, namespaceName)
		if err != nil {
			c.Error(fmt.Errorf("failed to get pods: %w", err))
		}

	case "DaemonSet":
		ds, err := database.GetDaemonSetByName(ctx, workloadName, namespaceName)
		if err != nil {
			c.Error(err)
			if common.IsNotFoundError(err) {
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "daemonset not found",
					"details": err.Error(),
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "failed to get daemonset",
					"details": err.Error(),
				})
			}
			return
		}
		response.Replicas = ds.DesiredNumberScheduled
		response.Ready = ds.NumberReady
		response.CreatedAt = ds.CreatedAt
		pods, err = database.GetPodsByWorkload(ctx, "DaemonSet", workloadName, namespaceName)
		if err != nil {
			c.Error(fmt.Errorf("failed to get pods: %w", err))
		}

	case "Pod":
		pod, err := database.GetPodByName(ctx, workloadName, namespaceName)
		if err != nil {
			c.Error(err)
			if common.IsNotFoundError(err) {
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "pod not found",
					"details": err.Error(),
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "failed to get pod",
					"details": err.Error(),
				})
			}
			return
		}
		// Reject controller-owned pods so the caller uses the correct workload type.
		if pod.OwnerKind != nil && *pod.OwnerKind != "" && *pod.OwnerKind != "Node" {
			err := fmt.Errorf("pod %s belongs to %s/%s, use workload endpoint with type %s instead",
				workloadName, *pod.OwnerKind, *pod.OwnerName, *pod.OwnerKind)
			c.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid pod",
				"details": err.Error(),
			})
			return
		}
		response.Replicas = 1
		if pod.Phase == "Running" {
			response.Ready = 1
		}
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
					c.Error(fmt.Errorf("failed to get containers for pod %s: %w", pod.Name, err))
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
