package web

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thread_koder/mochi/internal/database"
	"github.com/thread_koder/mochi/internal/kubernetes"
	"github.com/thread_koder/mochi/internal/prometheus"
	"github.com/thread_koder/mochi/internal/redis"
)

// Represents home page API response
type HomeResponse struct {
	ClusterName  string          `json:"cluster_name"`
	Stats        Stats           `json:"stats"`
	HealthChecks map[string]bool `json:"health_checks"`
	Activities   []Activity      `json:"activities"`
}

// Returns home page data
func GetHome(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response := HomeResponse{
		ClusterName:  "Kubernetes Cluster",
		HealthChecks: make(map[string]bool),
		Activities:   make([]Activity, 0),
	}

	// Get cluster info for name
	if info, err := kubernetes.GetClusterInfo(ctx); err != nil {
		c.Error(fmt.Errorf("failed to get cluster info: %w", err))
	} else {
		if info.ClusterName != "" {
			response.ClusterName = info.ClusterName
		}
	}

	// Get stats
	if stats, err := GetStats(ctx); err != nil {
		c.Error(fmt.Errorf("failed to get stats: %w", err))
	} else {
		response.Stats = stats
	}

	// Get health checks
	response.HealthChecks["database"] = database.HealthCheck(ctx) == nil
	response.HealthChecks["kubernetes"] = kubernetes.HealthCheck(ctx) == nil
	response.HealthChecks["prometheus"] = prometheus.HealthCheck(ctx) == nil
	response.HealthChecks["redis"] = redis.HealthCheck(ctx) == nil

	// Calculate health score
	healthyCount := 0
	for _, healthy := range response.HealthChecks {
		if healthy {
			healthyCount++
		}
	}
	if len(response.HealthChecks) > 0 {
		response.Stats.HealthScore = (healthyCount * 100) / len(response.HealthChecks)
	}

	// Get recent activities
	if activities, err := GetActivities(ctx, 10); err != nil {
		c.Error(fmt.Errorf("failed to get activities: %w", err))
	} else {
		response.Activities = activities
	}

	c.JSON(http.StatusOK, response)
}
