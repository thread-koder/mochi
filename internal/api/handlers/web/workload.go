package web

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/database"
)

// Represents workload API response
type WorkloadResponse struct {
	Namespace         string        `json:"namespace"`
	WorkloadType      string        `json:"workload_type"`
	WorkloadName      string        `json:"workload_name"`
	WorkloadReplicas  int           `json:"workload_replicas"`
	WorkloadReady     int           `json:"workload_ready"`
	WorkloadCreatedAt time.Time     `json:"workload_created_at"`
	Pods              []Pod         `json:"pods"`
	Containers        []Container   `json:"containers"`
	Stats             WorkloadStats `json:"stats"`
}

// Represents workload statistics
type WorkloadStats struct {
	PodCount       int `json:"pod_count"`
	ContainerCount int `json:"container_count"`
}

// Represents a pod
type Pod struct {
	Name      string    `json:"name"`
	UID       string    `json:"uid"`
	Phase     string    `json:"phase"`
	Node      string    `json:"node"`
	CreatedAt time.Time `json:"created_at"`
}

// Represents a container
type Container struct {
	Name          string `json:"name"`
	Image         string `json:"image"`
	CPURequest    string `json:"cpu_request"`
	CPULimit      string `json:"cpu_limit"`
	MemoryRequest string `json:"memory_request"`
	MemoryLimit   string `json:"memory_limit"`
}

// Returns workload page data
func GetWorkload(c *gin.Context) {
	namespaceName := c.Param("namespace")
	workloadType := c.Param("type")
	workloadName := c.Param("name")

	// Validate workload type
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
		Namespace:    namespaceName,
		WorkloadType: workloadType,
		WorkloadName: workloadName,
		Pods:         make([]Pod, 0),
		Containers:   make([]Container, 0),
		Stats:        WorkloadStats{},
	}

	// Get workload metadata based on type
	var pods []*database.Pod
	switch workloadType {
	case "Deployment":
		dep, err := database.GetDeploymentByName(ctx, workloadName, namespaceName)
		if err != nil {
			c.Error(err)
			if isNotFoundError(err) {
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
		response.WorkloadReplicas = dep.Replicas
		response.WorkloadReady = dep.ReadyReplicas
		response.WorkloadCreatedAt = dep.CreatedAt
		pods, err = database.GetPodsByWorkload(ctx, "Deployment", workloadName, namespaceName)
		if err != nil {
			c.Error(fmt.Errorf("failed to get pods: %w", err))
		}

	case "StatefulSet":
		sts, err := database.GetStatefulSetByName(ctx, workloadName, namespaceName)
		if err != nil {
			c.Error(err)
			if isNotFoundError(err) {
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
		response.WorkloadReplicas = sts.Replicas
		response.WorkloadReady = sts.ReadyReplicas
		response.WorkloadCreatedAt = sts.CreatedAt
		pods, err = database.GetPodsByWorkload(ctx, "StatefulSet", workloadName, namespaceName)
		if err != nil {
			c.Error(fmt.Errorf("failed to get pods: %w", err))
		}

	case "DaemonSet":
		ds, err := database.GetDaemonSetByName(ctx, workloadName, namespaceName)
		if err != nil {
			c.Error(err)
			if isNotFoundError(err) {
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
		response.WorkloadReplicas = ds.DesiredNumberScheduled
		response.WorkloadReady = ds.NumberReady
		response.WorkloadCreatedAt = ds.CreatedAt
		pods, err = database.GetPodsByWorkload(ctx, "DaemonSet", workloadName, namespaceName)
		if err != nil {
			c.Error(fmt.Errorf("failed to get pods: %w", err))
		}

	case "Pod":
		pod, err := database.GetPodByName(ctx, workloadName, namespaceName)
		if err != nil {
			c.Error(err)
			if isNotFoundError(err) {
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
		// Validate it's a standalone pod (no owner) or a system pod (Node-owned)
		if pod.OwnerKind != nil && *pod.OwnerKind != "" && *pod.OwnerKind != "Node" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid pod",
				"details": fmt.Sprintf("Pod '%s' belongs to %s/%s. Use the workload endpoint instead.", workloadName, *pod.OwnerKind, *pod.OwnerName),
			})
			return
		}
		response.WorkloadReplicas = 1
		if pod.Phase == "Running" {
			response.WorkloadReady = 1
		}
		response.WorkloadCreatedAt = pod.CreatedAt
		pods = []*database.Pod{pod}
	}

	// Process pods
	response.Stats.PodCount = len(pods)
	uniqueContainers := make(map[string]Container)

	for _, pod := range pods {
		nodeName := ""
		if pod.NodeName != nil {
			nodeName = *pod.NodeName
		}

		podDetail := Pod{
			Name:      pod.Name,
			UID:       pod.UID,
			Phase:     pod.Phase,
			Node:      nodeName,
			CreatedAt: pod.CreatedAt,
		}
		response.Pods = append(response.Pods, podDetail)

		// Get containers for this pod
		containers, err := database.GetContainersByPodUID(ctx, pod.UID)
		if err != nil {
			c.Error(fmt.Errorf("failed to get containers for pod %s: %w", pod.Name, err))
			continue
		}

		for _, container := range containers {
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

				containerDetail := Container{
					Name:          container.Name,
					Image:         container.Image,
					CPURequest:    cpuRequest,
					CPULimit:      cpuLimit,
					MemoryRequest: memoryRequest,
					MemoryLimit:   memoryLimit,
				}
				uniqueContainers[container.Name] = containerDetail
			}
		}
	}

	response.Containers = make([]Container, 0, len(uniqueContainers))
	for _, container := range uniqueContainers {
		response.Containers = append(response.Containers, container)
	}

	response.Stats.ContainerCount = len(response.Containers)

	c.JSON(http.StatusOK, response)
}
