package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/internal/logger"
)

// Upserts a container recommendation into the database
func UpsertContainerRecommendation(ctx context.Context, rec *ContainerRecommendation) error {
	log := logger.WithComponent("database")
	log.Debug().
		Int64("container_id", rec.ContainerID).
		Str("container_name", rec.ContainerName).
		Str("namespace", rec.Namespace).
		Msg("Upserting recommendation")

	query := `
		INSERT INTO container_recommendations (
			container_id, pod_uid, container_name, namespace,
			current_cpu_request, current_cpu_limit, current_memory_request, current_memory_limit,
			recommended_cpu_request, recommended_cpu_limit, recommended_memory_request, recommended_memory_limit,
			recommendation_mode, confidence_score, status, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
		ON CONFLICT (container_id) DO UPDATE SET
			pod_uid = EXCLUDED.pod_uid,
			container_name = EXCLUDED.container_name,
			namespace = EXCLUDED.namespace,
			current_cpu_request = EXCLUDED.current_cpu_request,
			current_cpu_limit = EXCLUDED.current_cpu_limit,
			current_memory_request = EXCLUDED.current_memory_request,
			current_memory_limit = EXCLUDED.current_memory_limit,
			recommended_cpu_request = EXCLUDED.recommended_cpu_request,
			recommended_cpu_limit = EXCLUDED.recommended_cpu_limit,
			recommended_memory_request = EXCLUDED.recommended_memory_request,
			recommended_memory_limit = EXCLUDED.recommended_memory_limit,
			recommendation_mode = EXCLUDED.recommendation_mode,
			confidence_score = EXCLUDED.confidence_score,
			status = CASE 
				WHEN container_recommendations.status = 'applied' THEN container_recommendations.status
				ELSE EXCLUDED.status
			END,
			updated_at = NOW()
	`

	_, err := Pool.Exec(ctx, query,
		rec.ContainerID, rec.PodUID, rec.ContainerName, rec.Namespace,
		rec.CurrentCPURequest, rec.CurrentCPULimit, rec.CurrentMemoryRequest, rec.CurrentMemoryLimit,
		rec.RecommendedCPURequest, rec.RecommendedCPULimit, rec.RecommendedMemoryRequest, rec.RecommendedMemoryLimit,
		rec.RecommendationMode, rec.ConfidenceScore, rec.Status, rec.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert recommendation: %w", err)
	}

	log.Debug().
		Int64("container_id", rec.ContainerID).
		Str("container_name", rec.ContainerName).
		Str("namespace", rec.Namespace).
		Msg("Recommendation upserted successfully")

	return nil
}

// Gets a container recommendation by ID
func GetContainerRecommendationByID(ctx context.Context, id int64) (*ContainerRecommendation, error) {
	query := `
		SELECT id, container_id, pod_uid, container_name, namespace,
		       current_cpu_request, current_cpu_limit, current_memory_request, current_memory_limit,
		       recommended_cpu_request, recommended_cpu_limit, recommended_memory_request, recommended_memory_limit,
		       recommendation_mode, confidence_score, status, created_at, updated_at, applied_at
		FROM container_recommendations
		WHERE id = $1
	`

	var rec ContainerRecommendation
	err := Pool.QueryRow(ctx, query, id).Scan(
		&rec.ID, &rec.ContainerID, &rec.PodUID, &rec.ContainerName, &rec.Namespace,
		&rec.CurrentCPURequest, &rec.CurrentCPULimit, &rec.CurrentMemoryRequest, &rec.CurrentMemoryLimit,
		&rec.RecommendedCPURequest, &rec.RecommendedCPULimit, &rec.RecommendedMemoryRequest, &rec.RecommendedMemoryLimit,
		&rec.RecommendationMode, &rec.ConfidenceScore, &rec.Status, &rec.CreatedAt, &rec.UpdatedAt, &rec.AppliedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("recommendation not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get recommendation: %w", err)
	}

	return &rec, nil
}

// Gets container recommendations by container ID
func GetContainerRecommendationsByContainerID(ctx context.Context, containerID int64) ([]*ContainerRecommendation, error) {
	query := `
		SELECT id, container_id, pod_uid, container_name, namespace,
		       current_cpu_request, current_cpu_limit, current_memory_request, current_memory_limit,
		       recommended_cpu_request, recommended_cpu_limit, recommended_memory_request, recommended_memory_limit,
		       recommendation_mode, confidence_score, status, created_at, updated_at, applied_at
		FROM container_recommendations
		WHERE container_id = $1
		ORDER BY created_at DESC
	`

	rows, err := Pool.Query(ctx, query, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query container recommendations by container ID: %w", err)
	}
	defer rows.Close()

	recommendations := make([]*ContainerRecommendation, 0)
	for rows.Next() {
		var rec ContainerRecommendation
		err := rows.Scan(
			&rec.ID, &rec.ContainerID, &rec.PodUID, &rec.ContainerName, &rec.Namespace,
			&rec.CurrentCPURequest, &rec.CurrentCPULimit, &rec.CurrentMemoryRequest, &rec.CurrentMemoryLimit,
			&rec.RecommendedCPURequest, &rec.RecommendedCPULimit, &rec.RecommendedMemoryRequest, &rec.RecommendedMemoryLimit,
			&rec.RecommendationMode, &rec.ConfidenceScore, &rec.Status, &rec.CreatedAt, &rec.UpdatedAt, &rec.AppliedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recommendation: %w", err)
		}
		recommendations = append(recommendations, &rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recommendations: %w", err)
	}

	return recommendations, nil
}

// Gets container recommendations by namespace
func GetContainerRecommendationsByNamespace(ctx context.Context, namespace string) ([]*ContainerRecommendation, error) {
	query := `
		SELECT id, container_id, pod_uid, container_name, namespace,
		       current_cpu_request, current_cpu_limit, current_memory_request, current_memory_limit,
		       recommended_cpu_request, recommended_cpu_limit, recommended_memory_request, recommended_memory_limit,
		       recommendation_mode, confidence_score, status, created_at, updated_at, applied_at
		FROM container_recommendations
		WHERE namespace = $1
		ORDER BY created_at DESC
	`

	rows, err := Pool.Query(ctx, query, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to query container recommendations by namespace: %w", err)
	}
	defer rows.Close()

	recommendations := make([]*ContainerRecommendation, 0)
	for rows.Next() {
		var rec ContainerRecommendation
		err := rows.Scan(
			&rec.ID, &rec.ContainerID, &rec.PodUID, &rec.ContainerName, &rec.Namespace,
			&rec.CurrentCPURequest, &rec.CurrentCPULimit, &rec.CurrentMemoryRequest, &rec.CurrentMemoryLimit,
			&rec.RecommendedCPURequest, &rec.RecommendedCPULimit, &rec.RecommendedMemoryRequest, &rec.RecommendedMemoryLimit,
			&rec.RecommendationMode, &rec.ConfidenceScore, &rec.Status, &rec.CreatedAt, &rec.UpdatedAt, &rec.AppliedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recommendation: %w", err)
		}
		recommendations = append(recommendations, &rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recommendations: %w", err)
	}

	return recommendations, nil
}

// Gets container recommendations by status
func GetContainerRecommendationsByStatus(ctx context.Context, status string) ([]*ContainerRecommendation, error) {
	query := `
		SELECT id, container_id, pod_uid, container_name, namespace,
		       current_cpu_request, current_cpu_limit, current_memory_request, current_memory_limit,
		       recommended_cpu_request, recommended_cpu_limit, recommended_memory_request, recommended_memory_limit,
		       recommendation_mode, confidence_score, status, created_at, updated_at, applied_at
		FROM container_recommendations
		WHERE status = $1
		ORDER BY created_at DESC
	`

	rows, err := Pool.Query(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query container recommendations by status: %w", err)
	}
	defer rows.Close()

	recommendations := make([]*ContainerRecommendation, 0)
	for rows.Next() {
		var rec ContainerRecommendation
		err := rows.Scan(
			&rec.ID, &rec.ContainerID, &rec.PodUID, &rec.ContainerName, &rec.Namespace,
			&rec.CurrentCPURequest, &rec.CurrentCPULimit, &rec.CurrentMemoryRequest, &rec.CurrentMemoryLimit,
			&rec.RecommendedCPURequest, &rec.RecommendedCPULimit, &rec.RecommendedMemoryRequest, &rec.RecommendedMemoryLimit,
			&rec.RecommendationMode, &rec.ConfidenceScore, &rec.Status, &rec.CreatedAt, &rec.UpdatedAt, &rec.AppliedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recommendation: %w", err)
		}
		recommendations = append(recommendations, &rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recommendations: %w", err)
	}

	return recommendations, nil
}

// Updates container recommendation status
func UpdateContainerRecommendationStatus(ctx context.Context, id int64, status string) error {
	log := logger.WithComponent("database")
	log.Debug().
		Int64("id", id).
		Str("status", status).
		Msg("Updating container recommendation status")

	query := `
		UPDATE container_recommendations
		SET status = $1,
		    updated_at = NOW(),
		    applied_at = CASE WHEN $1 = 'applied' THEN NOW() ELSE applied_at END
		WHERE id = $2
	`

	result, err := Pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update recommendation status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("recommendation not found")
	}

	log.Debug().
		Int64("id", id).
		Str("status", status).
		Msg("Recommendation status updated successfully")

	return nil
}

// Deletes container recommendations for a container
func DeleteContainerRecommendationsByContainerID(ctx context.Context, containerID int64) error {
	log := logger.WithComponent("database")

	query := `DELETE FROM container_recommendations WHERE container_id = $1`
	result, err := Pool.Exec(ctx, query, containerID)
	if err != nil {
		return fmt.Errorf("failed to delete recommendations: %w", err)
	}

	deleted := result.RowsAffected()
	if deleted > 0 {
		log.Debug().Int64("container_id", containerID).Int64("count", deleted).Msg("Recommendations deleted")
	}

	return nil
}

// Deletes container recommendations that haven't been updated since the specified time
func DeleteContainerRecommendationsNotUpdatedSince(ctx context.Context, since time.Time) error {
	log := logger.WithComponent("database")

	query := `DELETE FROM container_recommendations WHERE updated_at < $1 AND status = 'pending'`
	result, err := Pool.Exec(ctx, query, since)
	if err != nil {
		return fmt.Errorf("failed to delete stale recommendations: %w", err)
	}

	deleted := result.RowsAffected()
	if deleted > 0 {
		log.Debug().Int64("count", deleted).Msg("Stale recommendations deleted")
	}

	return nil
}
