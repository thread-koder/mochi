package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/logger"
)

// InsertComputeRecommendation inserts a compute recommendation.
// AnalysisTimeRange is stored as an interval when the string parses as a duration, otherwise the column stays unset.
func InsertComputeRecommendation(ctx context.Context, rec *ComputeRecommendation) error {
	if rec.ID == uuid.Nil {
		return fmt.Errorf("compute recommendation ID is required")
	}

	log := logger.WithComponent("database")
	log.Debug().
		Str("id", rec.ID.String()).
		Str("workload_type", rec.WorkloadType).
		Str("workload_name", rec.WorkloadName).
		Str("namespace", rec.Namespace).
		Msg("Inserting compute recommendation")

	query := `
		INSERT INTO compute_recommendations (
			id, workload_type, workload_name, namespace, recommendation_mode,
			recommendations, status, analysis_time_range, created_at, updated_at, generated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	var analysisTimeRange *time.Duration
	if rec.AnalysisTimeRange != "" {
		duration, err := time.ParseDuration(rec.AnalysisTimeRange)
		if err == nil {
			analysisTimeRange = &duration
		}
	}

	_, err := Pool.Exec(ctx, query,
		rec.ID, rec.WorkloadType, rec.WorkloadName, rec.Namespace, rec.RecommendationMode,
		rec.Recommendations, rec.Status, analysisTimeRange,
		rec.CreatedAt, rec.UpdatedAt, rec.GeneratedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert compute recommendation: %w", err)
	}

	log.Debug().
		Str("id", rec.ID.String()).
		Str("workload_name", rec.WorkloadName).
		Str("namespace", rec.Namespace).
		Msg("Compute recommendation inserted successfully")

	return nil
}

// GetComputeRecommendationByID returns a compute recommendation by its ID.
func GetComputeRecommendationByID(ctx context.Context, id uuid.UUID) (*ComputeRecommendation, error) {
	query := `
		SELECT id, workload_type, workload_name, namespace, recommendation_mode,
		       recommendations, status, analysis_time_range, created_at, updated_at, generated_at
		FROM compute_recommendations
		WHERE id = $1
	`

	var rec ComputeRecommendation
	var analysisTimeRange *time.Duration

	err := Pool.QueryRow(ctx, query, id).Scan(
		&rec.ID, &rec.WorkloadType, &rec.WorkloadName, &rec.Namespace,
		&rec.RecommendationMode, &rec.Recommendations, &rec.Status,
		&analysisTimeRange, &rec.CreatedAt, &rec.UpdatedAt, &rec.GeneratedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("compute_recommendation", id.String())
		}
		return nil, fmt.Errorf("failed to query compute recommendation by ID: %w", err)
	}

	if analysisTimeRange != nil {
		rec.AnalysisTimeRange = analysisTimeRange.String()
	}

	return &rec, nil
}

// GetLatestComputeRecommendation returns the newest compute recommendation for the workload by created_at.
func GetLatestComputeRecommendation(ctx context.Context, workloadType, workloadName, namespace string) (*ComputeRecommendation, error) {
	query := `
		SELECT id, workload_type, workload_name, namespace, recommendation_mode,
		       recommendations, status, analysis_time_range, created_at, updated_at, generated_at
		FROM compute_recommendations
		WHERE namespace = $1 AND workload_type = $2 AND workload_name = $3
		ORDER BY created_at DESC
		LIMIT 1
	`

	var rec ComputeRecommendation
	var analysisTimeRange *time.Duration

	err := Pool.QueryRow(ctx, query, namespace, workloadType, workloadName).Scan(
		&rec.ID, &rec.WorkloadType, &rec.WorkloadName, &rec.Namespace,
		&rec.RecommendationMode, &rec.Recommendations, &rec.Status,
		&analysisTimeRange, &rec.CreatedAt, &rec.UpdatedAt, &rec.GeneratedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("compute_recommendation", fmt.Sprintf("%s/%s/%s", namespace, workloadType, workloadName))
		}
		return nil, fmt.Errorf("failed to query latest compute recommendation: %w", err)
	}

	if analysisTimeRange != nil {
		rec.AnalysisTimeRange = analysisTimeRange.String()
	}

	return &rec, nil
}

// GetComputeRecommendations filters by optional pointer fields (nil means no filter).
// workloadName uses ILIKE substring match. limit and offset append only when positive,
// limit 0 means no LIMIT clause (unbounded page).
func GetComputeRecommendations(ctx context.Context, namespace *string, status *string, mode *string, workloadType *string, workloadName *string, limit, offset int) ([]*ComputeRecommendation, int64, error) {
	whereClause := "WHERE 1=1"
	args := []any{}
	argIndex := 1

	if namespace != nil {
		whereClause += fmt.Sprintf(" AND namespace = $%d", argIndex)
		args = append(args, *namespace)
		argIndex++
	}

	if status != nil {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *status)
		argIndex++
	}

	if mode != nil {
		whereClause += fmt.Sprintf(" AND recommendation_mode = $%d", argIndex)
		args = append(args, *mode)
		argIndex++
	}

	if workloadType != nil {
		whereClause += fmt.Sprintf(" AND workload_type = $%d", argIndex)
		args = append(args, *workloadType)
		argIndex++
	}

	if workloadName != nil {
		whereClause += fmt.Sprintf(" AND workload_name ILIKE $%d", argIndex)
		args = append(args, "%"+*workloadName+"%")
		argIndex++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM compute_recommendations %s", whereClause)
	var total int64
	err := Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count compute recommendations: %w", err)
	}

	selectQuery := fmt.Sprintf(`
		SELECT id, workload_type, workload_name, namespace, recommendation_mode,
		       recommendations, status, analysis_time_range, created_at, updated_at, generated_at
		FROM compute_recommendations
		%s
		ORDER BY created_at DESC
	`, whereClause)

	selectArgs := make([]any, len(args))
	copy(selectArgs, args)
	selectArgIndex := argIndex

	if limit > 0 {
		selectQuery += fmt.Sprintf(" LIMIT $%d", selectArgIndex)
		selectArgs = append(selectArgs, limit)
		selectArgIndex++
	}

	if offset > 0 {
		selectQuery += fmt.Sprintf(" OFFSET $%d", selectArgIndex)
		selectArgs = append(selectArgs, offset)
		selectArgIndex++
	}

	rows, err := Pool.Query(ctx, selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query compute recommendations: %w", err)
	}
	defer rows.Close()

	recommendations := make([]*ComputeRecommendation, 0)
	for rows.Next() {
		var rec ComputeRecommendation
		var analysisTimeRange *time.Duration

		err := rows.Scan(
			&rec.ID, &rec.WorkloadType, &rec.WorkloadName, &rec.Namespace,
			&rec.RecommendationMode, &rec.Recommendations, &rec.Status,
			&analysisTimeRange, &rec.CreatedAt, &rec.UpdatedAt, &rec.GeneratedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan compute recommendation: %w", err)
		}

		if analysisTimeRange != nil {
			rec.AnalysisTimeRange = analysisTimeRange.String()
		}

		recommendations = append(recommendations, &rec)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate compute recommendations: %w", err)
	}

	return recommendations, total, nil
}

// UpdateComputeRecommendationStatus sets status and updates updated_at.
func UpdateComputeRecommendationStatus(ctx context.Context, id uuid.UUID, status string) error {
	log := logger.WithComponent("database")
	log.Debug().
		Str("id", id.String()).
		Str("status", status).
		Msg("Updating compute recommendation status")

	query := `
		UPDATE compute_recommendations
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := Pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update compute recommendation status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NewNotFound("compute_recommendation", id.String())
	}

	log.Debug().
		Str("id", id.String()).
		Str("status", status).
		Msg("Compute recommendation status updated successfully")

	return nil
}

// MarkRecommendationsSuperseded changes the status of active recommendations (eg. not superseded)
// for the same workload to superseded, excluding the applied row (excludeID).
func MarkRecommendationsSuperseded(ctx context.Context, workloadType, workloadName, namespace string, excludeID uuid.UUID) error {
	log := logger.WithComponent("database")
	log.Debug().
		Str("workload_type", workloadType).
		Str("workload_name", workloadName).
		Str("namespace", namespace).
		Str("exclude_id", excludeID.String()).
		Msg("Marking recommendations as superseded")

	query := `
		UPDATE compute_recommendations
		SET status = 'superseded', updated_at = NOW()
		WHERE workload_type = $1
		  AND workload_name = $2
		  AND namespace = $3
		  AND id != $4
		  AND status != 'superseded'
	`

	result, err := Pool.Exec(ctx, query, workloadType, workloadName, namespace, excludeID)
	if err != nil {
		return fmt.Errorf("failed to mark recommendations as superseded: %w", err)
	}

	rowsAffected := result.RowsAffected()
	log.Debug().
		Int64("rows_affected", rowsAffected).
		Str("workload_type", workloadType).
		Str("workload_name", workloadName).
		Str("namespace", namespace).
		Msg("Recommendations marked as superseded successfully")

	return nil
}

// DeleteComputeRecommendationsOlderThan deletes compute recommendations with created_at before since (retention cleanup).
func DeleteComputeRecommendationsOlderThan(ctx context.Context, since time.Time) error {
	log := logger.WithComponent("database")

	query := `DELETE FROM compute_recommendations WHERE created_at < $1`
	result, err := Pool.Exec(ctx, query, since)
	if err != nil {
		return fmt.Errorf("failed to delete old compute recommendations: %w", err)
	}

	deleted := result.RowsAffected()
	if deleted > 0 {
		log.Debug().Int64("count", deleted).Msg("Old compute recommendations deleted")
	}

	return nil
}

// DeleteComputeRecommendationsForDeletedWorkloads deletes compute recommendations whose workload no longer appears
// in the synced tables: Deployment/StatefulSet/DaemonSet by name, or Pod when workload_type is Pod and
// the pod has no owner (i.e. standalone pods).
func DeleteComputeRecommendationsForDeletedWorkloads(ctx context.Context) error {
	log := logger.WithComponent("database")

	query := `
		DELETE FROM compute_recommendations cr
		WHERE NOT EXISTS (
			SELECT 1 FROM deployments d
			WHERE d.namespace = cr.namespace AND d.name = cr.workload_name
			AND cr.workload_type = 'Deployment'
		)
		AND NOT EXISTS (
			SELECT 1 FROM statefulsets s
			WHERE s.namespace = cr.namespace AND s.name = cr.workload_name
			AND cr.workload_type = 'StatefulSet'
		)
		AND NOT EXISTS (
			SELECT 1 FROM daemonsets ds
			WHERE ds.namespace = cr.namespace AND ds.name = cr.workload_name
			AND cr.workload_type = 'DaemonSet'
		)
		AND NOT EXISTS (
			SELECT 1 FROM pods p
			WHERE p.namespace = cr.namespace AND p.name = cr.workload_name
			AND cr.workload_type = 'Pod'
			AND (p.owner_kind IS NULL OR p.owner_name IS NULL)
		)
	`

	result, err := Pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to cleanup compute recommendations for deleted workloads: %w", err)
	}

	deleted := result.RowsAffected()
	if deleted > 0 {
		log.Debug().Int64("count", deleted).Msg("Compute recommendations cleaned up for deleted workloads")
	}

	return nil
}
