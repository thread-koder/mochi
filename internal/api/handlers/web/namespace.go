package web

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/database"
)

// Represents namespace API response
type NamespaceResponse struct {
	Name       string          `json:"name"`
	Phase      string          `json:"phase"`
	Stats      NamespaceStats  `json:"stats"`
	Workloads  []Workload      `json:"workloads"`
	Standalone []StandalonePod `json:"standalone_pods"`
	System     []StandalonePod `json:"system_pods"`
}

// Represents namespace statistics
type NamespaceStats struct {
	Workloads  int `json:"workloads"`
	Pods       int `json:"pods"`
	Containers int `json:"containers"`
}

// Represents a workload
type Workload struct {
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Replicas  int       `json:"replicas"`
	Ready     int       `json:"ready"`
	CreatedAt time.Time `json:"created_at"`
}

// Represents a standalone pod
type StandalonePod struct {
	Name      string    `json:"name"`
	Phase     string    `json:"phase"`
	Node      string    `json:"node"`
	CreatedAt time.Time `json:"created_at"`
}

// Returns namespace page data
func GetNamespace(c *gin.Context) {
	namespaceName := c.Param("namespace")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get namespace
	namespace, err := database.GetNamespaceByName(ctx, namespaceName)
	if err != nil {
		c.Error(err)
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "namespace not found",
				"details": err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "failed to get namespace",
				"details": err.Error(),
			})
		}
		return
	}

	response := NamespaceResponse{
		Name:       namespace.Name,
		Phase:      namespace.Phase,
		Stats:      NamespaceStats{},
		Workloads:  make([]Workload, 0),
		Standalone: make([]StandalonePod, 0),
		System:     make([]StandalonePod, 0),
	}

	// Get workloads
	deployments, err := database.GetDeploymentsByNamespace(ctx, namespace.Name)
	if err != nil {
		c.Error(fmt.Errorf("failed to get deployments: %w", err))
	} else {
		for _, dep := range deployments {
			response.Workloads = append(response.Workloads, Workload{
				Type:      "Deployment",
				Name:      dep.Name,
				Replicas:  dep.Replicas,
				Ready:     dep.ReadyReplicas,
				CreatedAt: dep.CreatedAt,
			})
		}
		response.Stats.Workloads += len(deployments)
	}

	statefulsets, err := database.GetStatefulSetsByNamespace(ctx, namespace.Name)
	if err != nil {
		c.Error(fmt.Errorf("failed to get statefulsets: %w", err))
	} else {
		for _, sts := range statefulsets {
			response.Workloads = append(response.Workloads, Workload{
				Type:      "StatefulSet",
				Name:      sts.Name,
				Replicas:  sts.Replicas,
				Ready:     sts.ReadyReplicas,
				CreatedAt: sts.CreatedAt,
			})
		}
		response.Stats.Workloads += len(statefulsets)
	}

	daemonsets, err := database.GetDaemonSetsByNamespace(ctx, namespace.Name)
	if err != nil {
		c.Error(fmt.Errorf("failed to get daemonsets: %w", err))
	} else {
		for _, ds := range daemonsets {
			response.Workloads = append(response.Workloads, Workload{
				Type:      "DaemonSet",
				Name:      ds.Name,
				Replicas:  ds.DesiredNumberScheduled,
				Ready:     ds.NumberReady,
				CreatedAt: ds.CreatedAt,
			})
		}
		response.Stats.Workloads += len(daemonsets)
	}

	// Get standalone pods
	standalonePods, err := database.GetStandalonePodsByNamespace(ctx, namespace.Name)
	if err != nil {
		c.Error(fmt.Errorf("failed to get standalone pods: %w", err))
	} else {
		for _, pod := range standalonePods {
			nodeName := ""
			if pod.NodeName != nil {
				nodeName = *pod.NodeName
			}
			response.Standalone = append(response.Standalone, StandalonePod{
				Name:      pod.Name,
				Phase:     pod.Phase,
				Node:      nodeName,
				CreatedAt: pod.CreatedAt,
			})
		}
	}

	// Get system pods (Node-owned)
	systemPods, err := database.GetPodsByOwnerKind(ctx, "Node", namespace.Name)
	if err != nil {
		c.Error(fmt.Errorf("failed to get system pods: %w", err))
	} else {
		for _, pod := range systemPods {
			nodeName := ""
			if pod.NodeName != nil {
				nodeName = *pod.NodeName
			}
			response.System = append(response.System, StandalonePod{
				Name:      pod.Name,
				Phase:     pod.Phase,
				Node:      nodeName,
				CreatedAt: pod.CreatedAt,
			})
		}
	}

	// Get pod count
	podCount, err := database.GetPodCountByNamespace(ctx, namespace.Name)
	if err != nil {
		c.Error(fmt.Errorf("failed to get pod count: %w", err))
	} else {
		response.Stats.Pods = podCount
	}

	// Get container count
	containerCount, err := database.GetContainerCountByNamespace(ctx, namespace.Name)
	if err != nil {
		c.Error(fmt.Errorf("failed to get container count: %w", err))
	} else {
		response.Stats.Containers = containerCount
	}

	c.JSON(http.StatusOK, response)
}
