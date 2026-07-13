package database

import (
	"context"
	"errors"
	"fmt"
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
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := Pool.Exec(ctx, query,
		rec.ID, rec.WorkloadType, rec.WorkloadName, rec.Namespace, rec.RecommendationMode,
		rec.Recommendations, rec.Status, rec.AnalysisTimeRange,
		rec.CreatedAt, rec.UpdatedAt,
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
		WHERE id = $1
	`

	var rec ComputeRecommendation

	err := Pool.QueryRow(ctx, query, id).Scan(
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
		WHERE namespace = $1 AND workload_type = $2 AND workload_name = $3
		ORDER BY created_at DESC
		LIMIT 1
	`

	var rec ComputeRecommendation

	err := Pool.QueryRow(ctx, query, namespace, workloadType, workloadName).Scan(
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
		       recommendations, status, analysis_time_range, created_at, updated_at
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
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := Pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to execute update compute recommendation status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NewNotFound("compute_recommendation", id.String())
	}

	return nil
}

func MarkRecommendationsSuperseded(ctx context.Context, workloadType, workloadName, namespace string, excludeID uuid.UUID) error {
	query := `
		UPDATE compute_recommendations
		SET status = 'superseded', updated_at = NOW()
		WHERE workload_type = $1
		  AND workload_name = $2
		  AND namespace = $3
		  AND id != $4
		  AND status != 'superseded'
	`

	_, err := Pool.Exec(ctx, query, workloadType, workloadName, namespace, excludeID)
	if err != nil {
		return fmt.Errorf("failed to execute update compute recommendations: %w", err)
	}

	return nil
}

func DeleteComputeRecommendationsOlderThan(ctx context.Context, since time.Time) error {
	query := `DELETE FROM compute_recommendations WHERE created_at < $1`
	_, err := Pool.Exec(ctx, query, since)
	return err
}

func DeleteComputeRecommendationsForDeletedWorkloads(ctx context.Context) error {
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

	_, err := Pool.Exec(ctx, query)
	return err
}
