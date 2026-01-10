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

// Represents data for the home page
type HomeData struct {
	Title        string
	Page         string
	ClusterName  string
	Stats        Stats
	HealthChecks map[string]bool
	Activities   []ActivityItemDisplay
}

// Represents activity item with formatted time
type ActivityItemDisplay struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	TimeAgo   string `json:"time_ago"`
	Icon      string `json:"icon"`
	IconColor string `json:"icon_color"`
}

// Renders the home page
func Home(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data := HomeData{
		Title:        "Home",
		Page:         "home",
		ClusterName:  "Kubernetes Cluster",
		HealthChecks: make(map[string]bool),
		Activities:   make([]ActivityItemDisplay, 0),
	}

	// Get cluster info for name
	if info, err := kubernetes.GetClusterInfo(ctx); err != nil {
		c.Error(fmt.Errorf("failed to get cluster info: %w", err))
	} else {
		if info.ClusterName != "" {
			data.ClusterName = info.ClusterName
		}
	}

	// Get stats
	if stats, err := GetStatsData(ctx); err != nil {
		c.Error(fmt.Errorf("failed to get stats: %w", err))
	} else {
		data.Stats = stats
	}

	// Get health checks
	data.HealthChecks["database"] = database.HealthCheck(ctx) == nil
	data.HealthChecks["kubernetes"] = kubernetes.HealthCheck(ctx) == nil
	data.HealthChecks["prometheus"] = prometheus.HealthCheck(ctx) == nil
	data.HealthChecks["redis"] = redis.HealthCheck(ctx) == nil

	// Calculate health score
	healthyCount := 0
	for _, healthy := range data.HealthChecks {
		if healthy {
			healthyCount++
		}
	}
	data.Stats.HealthScore = (healthyCount * 100) / len(data.HealthChecks)

	// Get recent activity
	if activities, err := GetActivityItems(ctx, 10); err != nil {
		c.Error(fmt.Errorf("failed to get activity items: %w", err))
	} else {
		// Format activities for display
		displayActivities := make([]ActivityItemDisplay, 0, len(activities))
		for _, activity := range activities {
			display := ActivityItemDisplay{
				Type:    activity.Type,
				Message: activity.Message,
				TimeAgo: formatTimeAgo(activity.Timestamp),
			}

			switch activity.Type {
			case "recommendation_applied":
				display.Icon = "✓"
				display.IconColor = "text-green-400"
			case "recommendation_generated":
				display.Icon = "💡"
				display.IconColor = "text-purple-400"
			default:
				display.Icon = "📊"
				display.IconColor = "text-blue-400"
			}

			displayActivities = append(displayActivities, display)
		}
		data.Activities = displayActivities
	}

	c.HTML(http.StatusOK, "home.html", data)
}
