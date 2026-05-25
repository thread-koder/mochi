package web

import (
	"context"
	"fmt"
	"time"

	"github.com/thread_koder/mochi/core/internal/database"
	"golang.org/x/sync/errgroup"
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
	var namespaceCount, workloadCount, podCount int

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		namespaceCount, err = database.GetNamespaceCount(gctx)
		if err != nil {
			return fmt.Errorf("failed to get namespace count: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		workloadCount, err = database.GetWorkloadCount(gctx)
		if err != nil {
			return fmt.Errorf("failed to get workload count: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		podCount, err = database.GetPodCount(gctx)
		if err != nil {
			return fmt.Errorf("failed to get pod count: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return Stats{}, err
	}

	return Stats{
		Namespaces: namespaceCount,
		Workloads:  workloadCount,
		Pods:       podCount,
	}, nil
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
