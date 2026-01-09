package web

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/thread_koder/mochi/internal/database"
)

// Helper function to parse integer from string
func parseInt(s string) (int, error) {
	result, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("failed to parse integer: %w", err)
	}
	return result, nil
}

// Formats timestamp as "X minutes/hours/days ago"
func formatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// Gets statistics data (shared by home handler and stats API endpoint)
func GetStatsData(ctx context.Context) (Stats, error) {
	stats := Stats{}

	if count, err := database.GetNamespaceCount(ctx); err == nil {
		stats.Namespaces = count
	}

	if count, err := database.GetWorkloadCount(ctx); err == nil {
		stats.Workloads = count
	}

	if count, err := database.GetPodCount(ctx); err == nil {
		stats.Pods = count
	}

	return stats, nil
}

// Gets activity items (shared by home handler and activity API endpoint)
func GetActivityItems(ctx context.Context, limit int) ([]ActivityItem, error) {
	computeRecommendations, err := database.GetComputeRecommendations(
		ctx,
		nil, nil, nil, nil, nil,
		limit,
		0,
	)
	if err != nil {
		return nil, err
	}

	activities := make([]ActivityItem, 0, len(computeRecommendations))
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

		activities = append(activities, ActivityItem{
			Type:      activityType,
			Message:   message,
			Timestamp: rec.CreatedAt,
		})
	}

	return activities, nil
}
