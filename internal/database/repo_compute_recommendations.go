package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/internal/logger"
)

// Inserts a new compute recommendation into the database
func InsertComputeRecommendation(ctx context.Context, rec *ComputeRecommendation) error {
	log := logger.WithComponent("database")
	log.Debug().
		Str("workload_type", rec.WorkloadType).
		Str("workload_name", rec.WorkloadName).
		Str("namespace", rec.Namespace).
		Msg("Inserting compute recommendation")

	query := `
		INSERT INTO compute_recommendations (
			workload_type, workload_name, namespace, recommendation_mode,
			recommendations, status, analysis_time_range, created_at, updated_at, generated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	var analysisTimeRange *time.Duration
	if rec.AnalysisTimeRange != "" {
		duration, err := time.ParseDuration(rec.AnalysisTimeRange)
		if err == nil {
			analysisTimeRange = &duration
		}
	}

	err := Pool.QueryRow(ctx, query,
		rec.WorkloadType, rec.WorkloadName, rec.Namespace, rec.RecommendationMode,
		rec.Recommendations, rec.Status, analysisTimeRange,
		rec.CreatedAt, rec.UpdatedAt, rec.GeneratedAt,
	).Scan(&rec.ID)
	if err != nil {
		return fmt.Errorf("failed to insert compute recommendation: %w", err)
	}

	log.Debug().
		Int64("id", rec.ID).
		Str("workload_name", rec.WorkloadName).
		Str("namespace", rec.Namespace).
		Msg("Compute recommendation inserted successfully")

	return nil
}

// Gets a compute recommendation by ID
func GetComputeRecommendationByID(ctx context.Context, id int64) (*ComputeRecommendation, error) {
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
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("compute recommendation %d not found", id)
		}
		return nil, fmt.Errorf("failed to query compute recommendation by ID: %w", err)
	}

	if analysisTimeRange != nil {
		durationStr := analysisTimeRange.String()
		rec.AnalysisTimeRange = durationStr
	}

	return &rec, nil
}

// Gets the latest compute recommendation for a workload
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
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("compute recommendation for %s/%s/%s not found", namespace, workloadType, workloadName)
		}
		return nil, fmt.Errorf("failed to query latest compute recommendation: %w", err)
	}

	if analysisTimeRange != nil {
		durationStr := analysisTimeRange.String()
		rec.AnalysisTimeRange = durationStr
	}

	return &rec, nil
}

// Gets all compute recommendations with optional filters
func GetComputeRecommendations(ctx context.Context, namespace *string, status *string, mode *string, workloadType *string, workloadName *string, limit, offset int) ([]*ComputeRecommendation, int64, error) {
	// Build WHERE clause for both count and select queries
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

	// Get total count (without LIMIT/OFFSET)
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM compute_recommendations %s", whereClause)
	var total int64
	err := Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count compute recommendations: %w", err)
	}

	// Get paginated results
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
			durationStr := analysisTimeRange.String()
			rec.AnalysisTimeRange = durationStr
		}

		recommendations = append(recommendations, &rec)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating compute recommendations: %w", err)
	}

	return recommendations, total, nil
}

// Updates the status of a compute recommendation
func UpdateComputeRecommendationStatus(ctx context.Context, id int64, status string) error {
	log := logger.WithComponent("database")
	log.Debug().
		Int64("id", id).
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
		return fmt.Errorf("compute recommendation %d not found", id)
	}

	log.Debug().
		Int64("id", id).
		Str("status", status).
		Msg("Compute recommendation status updated successfully")

	return nil
}

// Marks all recommendations for a workload as superseded (except the one being applied)
func MarkRecommendationsSuperseded(ctx context.Context, workloadType, workloadName, namespace string, excludeID int64) error {
	log := logger.WithComponent("database")
	log.Debug().
		Str("workload_type", workloadType).
		Str("workload_name", workloadName).
		Str("namespace", namespace).
		Int64("exclude_id", excludeID).
		Msg("Marking recommendations as superseded")

	query := `
		UPDATE compute_recommendations
		SET status = 'superseded', updated_at = NOW()
		WHERE workload_type = $1
		  AND workload_name = $2
		  AND namespace = $3
		  AND id != $4
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

// Deletes a compute recommendation by ID
func DeleteComputeRecommendation(ctx context.Context, id int64) error {
	log := logger.WithComponent("database")
	log.Debug().
		Int64("id", id).
		Msg("Deleting compute recommendation")

	query := `DELETE FROM compute_recommendations WHERE id = $1`

	result, err := Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete compute recommendation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("compute recommendation %d not found", id)
	}

	log.Debug().
		Int64("id", id).
		Msg("Compute recommendation deleted successfully")

	return nil
}

// Deletes compute recommendations older than the specified time
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

// Deletes compute recommendations for workloads that no longer exist
func DeleteComputeRecommendationsForDeletedWorkloads(ctx context.Context) error {
	log := logger.WithComponent("database")

	// Delete recommendations for workloads that don't exist in any workload table
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
