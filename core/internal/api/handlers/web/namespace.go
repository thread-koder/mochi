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

// NamespaceResponse is the payload for namespace detail pages.
type NamespaceResponse struct {
	Name       string          `json:"name"`
	Phase      string          `json:"phase"`
	Stats      NamespaceStats  `json:"stats"`
	Workloads  []Workload      `json:"workloads"`
	Standalone []StandalonePod `json:"standalone_pods"`
	System     []StandalonePod `json:"system_pods"`
}

// NamespaceStats holds namespace-level workload and resource counts.
type NamespaceStats struct {
	Workloads  int `json:"workloads"`
	Pods       int `json:"pods"`
	Containers int `json:"containers"`
}

// Workload represents one workload row in namespace details.
type Workload struct {
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Replicas  int       `json:"replicas"`
	Ready     int       `json:"ready"`
	CreatedAt time.Time `json:"created_at"`
}

// StandalonePod represents a pod that is not managed by a workload controller.
type StandalonePod struct {
	Name      string    `json:"name"`
	Phase     string    `json:"phase"`
	Node      string    `json:"node"`
	CreatedAt time.Time `json:"created_at"`
}

// GetNamespace returns metadata, workloads, and pods for a namespace.
func GetNamespace(c *gin.Context) {
	namespaceName := c.Param("namespace")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	namespace, err := database.GetNamespaceByName(ctx, namespaceName)
	if err != nil {
		c.Error(err)
		if common.IsNotFoundError(err) {
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

	var (
		deployments    []*database.Deployment
		statefulsets   []*database.StatefulSet
		daemonsets     []*database.DaemonSet
		standalonePods []*database.Pod
		systemPods     []*database.Pod
		podCount       int
		containerCount int
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		deployments, err = database.GetDeploymentsByNamespace(gctx, namespaceName)
		if err != nil {
			c.Error(fmt.Errorf("failed to get deployments: %w", err))
		}
		return nil
	})

	g.Go(func() error {
		var err error
		statefulsets, err = database.GetStatefulSetsByNamespace(gctx, namespaceName)
		if err != nil {
			c.Error(fmt.Errorf("failed to get statefulsets: %w", err))
		}
		return nil
	})

	g.Go(func() error {
		var err error
		daemonsets, err = database.GetDaemonSetsByNamespace(gctx, namespaceName)
		if err != nil {
			c.Error(fmt.Errorf("failed to get daemonsets: %w", err))
		}
		return nil
	})

	g.Go(func() error {
		var err error
		standalonePods, err = database.GetStandalonePodsByNamespace(gctx, namespaceName)
		if err != nil {
			c.Error(fmt.Errorf("failed to get standalone pods: %w", err))
		}
		return nil
	})

	g.Go(func() error {
		var err error
		systemPods, err = database.GetPodsByOwnerKind(gctx, "Node", namespaceName)
		if err != nil {
			c.Error(fmt.Errorf("failed to get system pods: %w", err))
		}
		return nil
	})

	g.Go(func() error {
		var err error
		podCount, err = database.GetPodCountByNamespace(gctx, namespaceName)
		if err != nil {
			c.Error(fmt.Errorf("failed to get pod count: %w", err))
		}
		return nil
	})

	g.Go(func() error {
		var err error
		containerCount, err = database.GetContainerCountByNamespace(gctx, namespaceName)
		if err != nil {
			c.Error(fmt.Errorf("failed to get container count: %w", err))
		}
		return nil
	})

	_ = g.Wait()

	workloads := make([]Workload, 0, len(deployments)+len(statefulsets)+len(daemonsets))
	for _, dep := range deployments {
		workloads = append(workloads, Workload{
			Type:      "Deployment",
			Name:      dep.Name,
			Replicas:  dep.Replicas,
			Ready:     dep.ReadyReplicas,
			CreatedAt: dep.CreatedAt,
		})
	}
	for _, sts := range statefulsets {
		workloads = append(workloads, Workload{
			Type:      "StatefulSet",
			Name:      sts.Name,
			Replicas:  sts.Replicas,
			Ready:     sts.ReadyReplicas,
			CreatedAt: sts.CreatedAt,
		})
	}
	for _, ds := range daemonsets {
		workloads = append(workloads, Workload{
			Type:      "DaemonSet",
			Name:      ds.Name,
			Replicas:  ds.DesiredNumberScheduled,
			Ready:     ds.NumberReady,
			CreatedAt: ds.CreatedAt,
		})
	}

	standalone := make([]StandalonePod, 0, len(standalonePods))
	for _, pod := range standalonePods {
		nodeName := ""
		if pod.Node != nil {
			nodeName = *pod.Node
		}
		standalone = append(standalone, StandalonePod{
			Name:      pod.Name,
			Phase:     pod.Phase,
			Node:      nodeName,
			CreatedAt: pod.CreatedAt,
		})
	}

	// Node-owned pods are shown separately from standalone user pods.
	system := make([]StandalonePod, 0, len(systemPods))
	for _, pod := range systemPods {
		nodeName := ""
		if pod.Node != nil {
			nodeName = *pod.Node
		}
		system = append(system, StandalonePod{
			Name:      pod.Name,
			Phase:     pod.Phase,
			Node:      nodeName,
			CreatedAt: pod.CreatedAt,
		})
	}

	response := NamespaceResponse{
		Name:       namespace.Name,
		Phase:      namespace.Phase,
		Workloads:  workloads,
		Standalone: standalone,
		System:     system,
		Stats: NamespaceStats{
			Workloads:  len(deployments) + len(statefulsets) + len(daemonsets),
			Pods:       podCount,
			Containers: containerCount,
		},
	}

	c.JSON(http.StatusOK, response)
}
