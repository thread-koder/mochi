package database

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/apperrors"
)

func InsertComputeRecommendation(ctx context.Context, rec *ComputeRecommendation) error {
	if rec.ID == uuid.Nil {
		return fmt.Errorf("compute recommendation ID is required")
	}

	if _, err := time.ParseDuration(rec.AnalysisTimeRange); err != nil {
		return fmt.Errorf("invalid analysis time range %q: %w", rec.AnalysisTimeRange, err)
	}

	query := `
		INSERT INTO compute_recommendations (
			id, workload_type, workload_name, namespace, recommendation_mode,
			recommendations, status, analysis_time_range, created_at, updated_at
		) VALUES (
			@id, @workload_type, @workload_name, @namespace, @recommendation_mode,
			@recommendations, @status, @analysis_time_range, @created_at, @updated_at
		)
	`

	_, err := Pool.Exec(ctx, query,
		pgx.StrictNamedArgs{
			"id":                  rec.ID,
			"workload_type":       rec.WorkloadType,
			"workload_name":       rec.WorkloadName,
			"namespace":           rec.Namespace,
			"recommendation_mode": rec.RecommendationMode,
			"recommendations":     rec.Recommendations,
			"status":              rec.Status,
			"analysis_time_range": rec.AnalysisTimeRange,
			"created_at":          rec.CreatedAt,
			"updated_at":          rec.UpdatedAt,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to execute insert compute recommendation: %w", err)
	}

	return nil
}

func GetComputeRecommendationByID(ctx context.Context, id uuid.UUID) (*ComputeRecommendation, error) {
	query := `
		SELECT id, workload_type, workload_name, namespace, recommendation_mode,
		       recommendations, status, analysis_time_range, created_at, updated_at
		FROM compute_recommendations
		WHERE id = @id
	`

	var rec ComputeRecommendation

	err := Pool.QueryRow(ctx, query,
		pgx.StrictNamedArgs{"id": id},
	).Scan(
		&rec.ID, &rec.WorkloadType, &rec.WorkloadName, &rec.Namespace,
		&rec.RecommendationMode, &rec.Recommendations, &rec.Status,
		&rec.AnalysisTimeRange, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("compute_recommendation", id.String())
		}
		return nil, fmt.Errorf("failed to query compute recommendation by ID: %w", err)
	}

	return &rec, nil
}

func GetLatestComputeRecommendation(ctx context.Context, workloadType, workloadName, namespace string) (*ComputeRecommendation, error) {
	query := `
		SELECT id, workload_type, workload_name, namespace, recommendation_mode,
		       recommendations, status, analysis_time_range, created_at, updated_at
		FROM compute_recommendations
		WHERE namespace = @namespace AND workload_type = @workload_type AND workload_name = @workload_name
		ORDER BY created_at DESC
		LIMIT 1
	`

	var rec ComputeRecommendation

	err := Pool.QueryRow(ctx, query,
		pgx.StrictNamedArgs{
			"namespace":     namespace,
			"workload_type": workloadType,
			"workload_name": workloadName,
		},
	).Scan(
		&rec.ID, &rec.WorkloadType, &rec.WorkloadName, &rec.Namespace,
		&rec.RecommendationMode, &rec.Recommendations, &rec.Status,
		&rec.AnalysisTimeRange, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("compute_recommendation", fmt.Sprintf("%s/%s/%s", namespace, workloadType, workloadName))
		}
		return nil, fmt.Errorf("failed to query compute recommendation by workload: %w", err)
	}

	return &rec, nil
}

func GetComputeRecommendations(ctx context.Context, namespace *string, status *string, mode *string, workloadType *string, workloadName *string, limit, offset int) ([]*ComputeRecommendation, int64, error) {
	whereClause := "WHERE 1=1"
	args := pgx.StrictNamedArgs{}

	if namespace != nil {
		whereClause += " AND namespace = @namespace"
		args["namespace"] = *namespace
	}

	if status != nil {
		whereClause += " AND status = @status"
		args["status"] = *status
	}

	if mode != nil {
		whereClause += " AND recommendation_mode = @recommendation_mode"
		args["recommendation_mode"] = *mode
	}

	if workloadType != nil {
		whereClause += " AND workload_type = @workload_type"
		args["workload_type"] = *workloadType
	}

	if workloadName != nil {
		whereClause += " AND workload_name ILIKE @workload_name"
		args["workload_name"] = "%" + *workloadName + "%"
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM compute_recommendations %s", whereClause)
	var total int64
	var err error
	if len(args) == 0 {
		err = Pool.QueryRow(ctx, countQuery).Scan(&total)
	} else {
		err = Pool.QueryRow(ctx, countQuery, args).Scan(&total)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count compute recommendations: %w", err)
	}

	selectQuery := fmt.Sprintf(`
		SELECT id, workload_type, workload_name, namespace, recommendation_mode,
		       recommendations, status, analysis_time_range, created_at, updated_at
		FROM compute_recommendations
		%s
		ORDER BY created_at DESC
	`, whereClause)

	selectArgs := pgx.StrictNamedArgs{}
	maps.Copy(selectArgs, args)

	if limit > 0 {
		selectQuery += " LIMIT @limit"
		selectArgs["limit"] = limit
	}

	if offset > 0 {
		selectQuery += " OFFSET @offset"
		selectArgs["offset"] = offset
	}

	var rows pgx.Rows
	if len(selectArgs) == 0 {
		rows, err = Pool.Query(ctx, selectQuery)
	} else {
		rows, err = Pool.Query(ctx, selectQuery, selectArgs)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query compute recommendations: %w", err)
	}
	defer rows.Close()

	recommendations := make([]*ComputeRecommendation, 0)
	for rows.Next() {
		var rec ComputeRecommendation

		err := rows.Scan(
			&rec.ID, &rec.WorkloadType, &rec.WorkloadName, &rec.Namespace,
			&rec.RecommendationMode, &rec.Recommendations, &rec.Status,
			&rec.AnalysisTimeRange, &rec.CreatedAt, &rec.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan compute recommendation: %w", err)
		}

		recommendations = append(recommendations, &rec)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate compute recommendations: %w", err)
	}

	return recommendations, total, nil
}

func UpdateComputeRecommendationStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE compute_recommendations
		SET status = @status, updated_at = NOW()
		WHERE id = @id
	`

	result, err := Pool.Exec(ctx, query,
		pgx.StrictNamedArgs{
			"status": status,
			"id":     id,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to execute update compute recommendation status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NewNotFound("compute_recommendation", id.String())
	}

	return nil
}

func SupersedeComputeRecommendations(ctx context.Context, workloadType, workloadName, namespace string, excludeID uuid.UUID) error {
	query := `
		UPDATE compute_recommendations
		SET status = 'superseded', updated_at = NOW()
		WHERE workload_type = @workload_type
		  AND workload_name = @workload_name
		  AND namespace = @namespace
		  AND id != @id
		  AND status != 'superseded'
	`

	_, err := Pool.Exec(ctx, query,
		pgx.StrictNamedArgs{
			"workload_type": workloadType,
			"workload_name": workloadName,
			"namespace":     namespace,
			"id":            excludeID,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to execute update compute recommendations: %w", err)
	}

	return nil
}

func PruneExpiredComputeRecommendations(ctx context.Context, since time.Time) error {
	query := `DELETE FROM compute_recommendations WHERE created_at < @created_at`
	_, err := Pool.Exec(ctx, query,
		pgx.StrictNamedArgs{"created_at": since},
	)
	return err
}

// PruneComputeRecommendations deletes recommendations whose matching workload row is gone.
func PruneComputeRecommendations(ctx context.Context, namespace, workloadType string) error {
	var existsClause string
	switch workloadType {
	case "Deployment":
		existsClause = `
			SELECT 1 FROM deployments d
			WHERE d.namespace = cr.namespace AND d.name = cr.workload_name`
	case "StatefulSet":
		existsClause = `
			SELECT 1 FROM statefulsets s
			WHERE s.namespace = cr.namespace AND s.name = cr.workload_name`
	case "DaemonSet":
		existsClause = `
			SELECT 1 FROM daemonsets ds
			WHERE ds.namespace = cr.namespace AND ds.name = cr.workload_name`
	case "Pod":
		existsClause = `
			SELECT 1 FROM pods p
			WHERE p.namespace = cr.namespace AND p.name = cr.workload_name
			AND (p.owner_kind IS NULL OR p.owner_name IS NULL)`
	default:
		return fmt.Errorf("unsupported workload type for prune compute recommendations: %s", workloadType)
	}

	query := fmt.Sprintf(`
		DELETE FROM compute_recommendations cr
		WHERE cr.namespace = @namespace
		  AND cr.workload_type = @workload_type
		  AND NOT EXISTS (%s)
	`, existsClause)

	_, err := Pool.Exec(ctx, query,
		pgx.StrictNamedArgs{
			"namespace":     namespace,
			"workload_type": workloadType,
		},
	)
	return err
}
