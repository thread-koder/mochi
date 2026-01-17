package web

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/database"
)

// Represents data for the namespace page
type NamespaceData struct {
	Title      string
	Page       string
	Namespace  *database.Namespace
	Stats      NamespaceStats
	Workloads  []WorkloadItem
	Standalone []StandalonePodItem
	System     []StandalonePodItem
}

// Represents namespace statistics
type NamespaceStats struct {
	Workloads  int `json:"workloads"`
	Pods       int `json:"pods"`
	Containers int `json:"containers"`
}

// Represents a workload item for display
type WorkloadItem struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Replicas  int    `json:"replicas"`
	Ready     int    `json:"ready"`
	CreatedAt string `json:"created_at"`
}

// Represents a standalone pod item for display
type StandalonePodItem struct {
	Name      string `json:"name"`
	Phase     string `json:"phase"`
	Node      string `json:"node"`
	CreatedAt string `json:"created_at"`
}

// Renders the namespace page
func Namespace(c *gin.Context) {
	namespaceName := c.Param("namespace")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get namespace
	namespace, err := database.GetNamespaceByName(ctx, namespaceName)
	if err != nil {
		c.Error(fmt.Errorf("failed to get namespace: %w", err))
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"Title":   "Not Found",
			"Page":    "error",
			"Message": fmt.Sprintf("Namespace '%s' not found", namespaceName),
		})
		return
	}

	data := NamespaceData{
		Title:      namespace.Name,
		Page:       "namespace",
		Namespace:  namespace,
		Stats:      NamespaceStats{},
		Workloads:  make([]WorkloadItem, 0),
		Standalone: make([]StandalonePodItem, 0),
		System:     make([]StandalonePodItem, 0),
	}

	// Get workloads
	deployments, err := database.GetDeploymentsByNamespace(ctx, namespace.Name)
	if err != nil {
		c.Error(fmt.Errorf("failed to get deployments: %w", err))
	} else {
		for _, dep := range deployments {
			data.Workloads = append(data.Workloads, WorkloadItem{
				Type:      "Deployment",
				Name:      dep.Name,
				Replicas:  dep.Replicas,
				Ready:     dep.ReadyReplicas,
				CreatedAt: formatTimeAgo(dep.CreatedAt),
			})
		}
		data.Stats.Workloads += len(deployments)
	}

	statefulsets, err := database.GetStatefulSetsByNamespace(ctx, namespace.Name)
	if err != nil {
		c.Error(fmt.Errorf("failed to get statefulsets: %w", err))
	} else {
		for _, sts := range statefulsets {
			data.Workloads = append(data.Workloads, WorkloadItem{
				Type:      "StatefulSet",
				Name:      sts.Name,
				Replicas:  sts.Replicas,
				Ready:     sts.ReadyReplicas,
				CreatedAt: formatTimeAgo(sts.CreatedAt),
			})
		}
		data.Stats.Workloads += len(statefulsets)
	}

	daemonsets, err := database.GetDaemonSetsByNamespace(ctx, namespace.Name)
	if err != nil {
		c.Error(fmt.Errorf("failed to get daemonsets: %w", err))
	} else {
		for _, ds := range daemonsets {
			data.Workloads = append(data.Workloads, WorkloadItem{
				Type:      "DaemonSet",
				Name:      ds.Name,
				Replicas:  ds.DesiredNumberScheduled,
				Ready:     ds.NumberReady,
				CreatedAt: formatTimeAgo(ds.CreatedAt),
			})
		}
		data.Stats.Workloads += len(daemonsets)
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
			data.Standalone = append(data.Standalone, StandalonePodItem{
				Name:      pod.Name,
				Phase:     pod.Phase,
				Node:      nodeName,
				CreatedAt: formatTimeAgo(pod.CreatedAt),
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
			data.System = append(data.System, StandalonePodItem{
				Name:      pod.Name,
				Phase:     pod.Phase,
				Node:      nodeName,
				CreatedAt: formatTimeAgo(pod.CreatedAt),
			})
		}
	}

	// Get pod count
	podCount, err := database.GetPodCountByNamespace(ctx, namespace.Name)
	if err != nil {
		c.Error(fmt.Errorf("failed to get pod count: %w", err))
	} else {
		data.Stats.Pods = podCount
	}

	// Get container count
	containerCount, err := database.GetContainerCountByNamespace(ctx, namespace.Name)
	if err != nil {
		c.Error(fmt.Errorf("failed to get container count: %w", err))
	} else {
		data.Stats.Containers = containerCount
	}

	c.HTML(http.StatusOK, "namespace.html", data)
}
