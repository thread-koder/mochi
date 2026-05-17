package web

import (
	"context"
	"fmt"
	"time"

	"github.com/thread_koder/mochi/core/internal/database"
)

// Stats summarizes top-level cluster counts for the home endpoint.
type Stats struct {
	Namespaces  int `json:"namespaces"`
	Workloads   int `json:"workloads"`
	Pods        int `json:"pods"`
	HealthScore int `json:"health_score"`
}

// GetStats loads summary counters for namespaces, workloads, and pods.
func GetStats(ctx context.Context) (Stats, error) {
	stats := Stats{}

	if count, err := database.GetNamespaceCount(ctx); err == nil {
		stats.Namespaces = count
	} else {
		return stats, fmt.Errorf("failed to get namespace count: %w", err)
	}

	if count, err := database.GetWorkloadCount(ctx); err == nil {
		stats.Workloads = count
	} else {
		return stats, fmt.Errorf("failed to get workload count: %w", err)
	}

	if count, err := database.GetPodCount(ctx); err == nil {
		stats.Pods = count
	} else {
		return stats, fmt.Errorf("failed to get pod count: %w", err)
	}

	return stats, nil
}

// Activity describes a recent recommendation event shown on the home page.
type Activity struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// GetActivities converts recent recommendations into UI activity entries.
func GetActivities(ctx context.Context, limit int) ([]Activity, error) {
	computeRecommendations, _, err := database.GetComputeRecommendations(
		ctx,
		nil, nil, nil, nil, nil,
		limit,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get compute recommendations: %w", err)
	}

	activities := make([]Activity, 0, len(computeRecommendations))
	for _, rec := range computeRecommendations {
		var activityType string
		var message string

		switch rec.Status {
		case "applied":
			activityType = "recommendation_applied"
			message = fmt.Sprintf("Compute recommendation applied to %s/%s in %s", rec.WorkloadType, rec.WorkloadName, rec.Namespace)
		case "pending":
			activityType = "recommendation_generated"
			message = fmt.Sprintf("Compute recommendations generated for %s/%s in %s", rec.WorkloadType, rec.WorkloadName, rec.Namespace)
		default:
			continue
		}

		activities = append(activities, Activity{
			Type:      activityType,
			Message:   message,
			Timestamp: rec.CreatedAt,
		})
	}

	return activities, nil
}
