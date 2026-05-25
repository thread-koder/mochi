package web

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/kubernetes"
	"github.com/thread_koder/mochi/core/internal/prometheus"
	"github.com/thread_koder/mochi/core/internal/redis"
	"golang.org/x/sync/errgroup"
)

// HomeResponse is the payload for the home page.
type HomeResponse struct {
	ClusterName  string          `json:"cluster_name"`
	Stats        Stats           `json:"stats"`
	HealthChecks map[string]bool `json:"health_checks"`
	Activities   []Activity      `json:"activities"`
}

// GetHome builds the dashboard response for the home page.
func GetHome(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var (
		clusterInfo  *kubernetes.ClusterInfo
		stats        Stats
		dbHealthy    bool
		kubeHealthy  bool
		promHealthy  bool
		redisHealthy bool
		activities   []Activity
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		info, err := kubernetes.GetClusterInfo(gctx)
		if err != nil {
			c.Error(fmt.Errorf("failed to get cluster info: %w", err))
			return nil
		}
		clusterInfo = info
		return nil
	})

	g.Go(func() error {
		statsData, err := GetStats(gctx)
		if err != nil {
			c.Error(fmt.Errorf("failed to get stats: %w", err))
			return nil
		}
		stats = statsData
		return nil
	})

	g.Go(func() error {
		dbHealthy = database.HealthCheck(gctx) == nil
		return nil
	})

	g.Go(func() error {
		kubeHealthy = kubernetes.HealthCheck(gctx) == nil
		return nil
	})

	g.Go(func() error {
		promHealthy = prometheus.HealthCheck(gctx) == nil
		return nil
	})

	g.Go(func() error {
		redisHealthy = redis.HealthCheck(gctx) == nil
		return nil
	})

	g.Go(func() error {
		activityList, err := GetActivities(gctx, 10)
		if err != nil {
			c.Error(fmt.Errorf("failed to get activities: %w", err))
			return nil
		}
		activities = activityList
		return nil
	})

	_ = g.Wait()

	if activities == nil {
		activities = make([]Activity, 0)
	}

	response := HomeResponse{
		ClusterName: "Kubernetes Cluster",
		Stats:       stats,
		HealthChecks: map[string]bool{
			"database":   dbHealthy,
			"kubernetes": kubeHealthy,
			"prometheus": promHealthy,
			"redis":      redisHealthy,
		},
		Activities: activities,
	}

	if clusterInfo != nil && clusterInfo.ClusterName != "" {
		response.ClusterName = clusterInfo.ClusterName
	}

	healthyCount := 0
	for _, healthy := range response.HealthChecks {
		if healthy {
			healthyCount++
		}
	}
	if len(response.HealthChecks) > 0 {
		response.Stats.HealthScore = (healthyCount * 100) / len(response.HealthChecks)
	}

	c.JSON(http.StatusOK, response)
}
