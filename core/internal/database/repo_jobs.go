package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/apperrors"
)

func UpsertJobsBatch(ctx context.Context, jobs []*Job) error {
	if len(jobs) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO jobs (
			name, namespace, uid, active, succeeded, failed,
			labels, annotations, created_at, synced_at
		) VALUES (
			@name, @namespace, @uid, @active, @succeeded, @failed,
			@labels, @annotations, @created_at, @synced_at
		)
		ON CONFLICT (namespace, name) DO UPDATE SET
			uid = EXCLUDED.uid,
			active = EXCLUDED.active,
			succeeded = EXCLUDED.succeeded,
			failed = EXCLUDED.failed,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, job := range jobs {
		batch.Queue(query, pgx.StrictNamedArgs{
			"name":        job.Name,
			"namespace":   job.Namespace,
			"uid":         job.UID,
			"active":      job.Active,
			"succeeded":   job.Succeeded,
			"failed":      job.Failed,
			"labels":      job.Labels,
			"annotations": job.Annotations,
			"created_at":  job.CreatedAt,
			"synced_at":   job.SyncedAt,
		})
	}

	results := tx.SendBatch(ctx, batch)

	for i := range jobs {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for job %d: %w", i, err)
		}
	}

	if err := results.Close(); err != nil {
		return fmt.Errorf("failed to close batch results: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func GetJobsByNamespace(ctx context.Context, namespace string) ([]*Job, error) {
	query := `
		SELECT id, name, namespace, uid, active, succeeded, failed,
		       labels, annotations, created_at, updated_at, synced_at
		FROM jobs
		WHERE namespace = @namespace
		ORDER BY name ASC
	`

	rows, err := Pool.Query(ctx, query,
		pgx.StrictNamedArgs{"namespace": namespace},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs by namespace: %w", err)
	}
	defer rows.Close()

	jobs := make([]*Job, 0)
	for rows.Next() {
		var job Job
		if err := rows.Scan(
			&job.ID, &job.Name, &job.Namespace, &job.UID,
			&job.Active, &job.Succeeded, &job.Failed,
			&job.Labels, &job.Annotations,
			&job.CreatedAt, &job.UpdatedAt, &job.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, &job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate jobs: %w", err)
	}

	return jobs, nil
}

func GetJobByName(ctx context.Context, name string, namespace string) (*Job, error) {
	query := `
		SELECT id, name, namespace, uid, active, succeeded, failed,
		       labels, annotations, created_at, updated_at, synced_at
		FROM jobs
		WHERE name = @name AND namespace = @namespace
		LIMIT 1
	`

	var job Job
	err := Pool.QueryRow(ctx, query,
		pgx.StrictNamedArgs{
			"name":      name,
			"namespace": namespace,
		},
	).Scan(
		&job.ID, &job.Name, &job.Namespace, &job.UID,
		&job.Active, &job.Succeeded, &job.Failed,
		&job.Labels, &job.Annotations,
		&job.CreatedAt, &job.UpdatedAt, &job.SyncedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("job", fmt.Sprintf("%s/%s", namespace, name))
		}
		return nil, fmt.Errorf("failed to query job by name: %w", err)
	}

	return &job, nil
}

// PruneJobs deletes Jobs not listed in uids.
// Empty uids deletes every Job in the namespace.
func PruneJobs(ctx context.Context, namespace string, uids []string) error {
	var err error
	if len(uids) == 0 {
		_, err = Pool.Exec(ctx,
			`DELETE FROM jobs WHERE namespace = @namespace`,
			pgx.StrictNamedArgs{"namespace": namespace},
		)
	} else {
		_, err = Pool.Exec(ctx,
			`DELETE FROM jobs WHERE namespace = @namespace AND NOT (uid = ANY(@uids::text[]))`,
			pgx.StrictNamedArgs{
				"namespace": namespace,
				"uids":      uids,
			},
		)
	}
	if err != nil {
		return fmt.Errorf("failed to prune jobs: %w", err)
	}
	return nil
}
