package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/apperrors"
)

func UpsertCronJobsBatch(ctx context.Context, cronjobs []*CronJob) error {
	if len(cronjobs) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO cronjobs (
			name, namespace, uid, schedule, suspend,
			labels, annotations, created_at, synced_at
		) VALUES (
			@name, @namespace, @uid, @schedule, @suspend,
			@labels, @annotations, @created_at, @synced_at
		)
		ON CONFLICT (namespace, name) DO UPDATE SET
			uid = EXCLUDED.uid,
			schedule = EXCLUDED.schedule,
			suspend = EXCLUDED.suspend,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, cronjob := range cronjobs {
		batch.Queue(query, pgx.StrictNamedArgs{
			"name":        cronjob.Name,
			"namespace":   cronjob.Namespace,
			"uid":         cronjob.UID,
			"schedule":    cronjob.Schedule,
			"suspend":     cronjob.Suspend,
			"labels":      cronjob.Labels,
			"annotations": cronjob.Annotations,
			"created_at":  cronjob.CreatedAt,
			"synced_at":   cronjob.SyncedAt,
		})
	}

	results := tx.SendBatch(ctx, batch)

	for i := range cronjobs {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for cronjob %d: %w", i, err)
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

func GetCronJobsByNamespace(ctx context.Context, namespace string) ([]*CronJob, error) {
	query := `
		SELECT id, name, namespace, uid, schedule, suspend,
		       labels, annotations, created_at, updated_at, synced_at
		FROM cronjobs
		WHERE namespace = @namespace
		ORDER BY name ASC
	`

	rows, err := Pool.Query(ctx, query,
		pgx.StrictNamedArgs{"namespace": namespace},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query cronjobs by namespace: %w", err)
	}
	defer rows.Close()

	cronjobs := make([]*CronJob, 0)
	for rows.Next() {
		var cj CronJob
		if err := rows.Scan(
			&cj.ID, &cj.Name, &cj.Namespace, &cj.UID,
			&cj.Schedule, &cj.Suspend,
			&cj.Labels, &cj.Annotations,
			&cj.CreatedAt, &cj.UpdatedAt, &cj.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan cronjob: %w", err)
		}
		cronjobs = append(cronjobs, &cj)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate cronjobs: %w", err)
	}

	return cronjobs, nil
}

func GetCronJobByName(ctx context.Context, name string, namespace string) (*CronJob, error) {
	query := `
		SELECT id, name, namespace, uid, schedule, suspend,
		       labels, annotations, created_at, updated_at, synced_at
		FROM cronjobs
		WHERE name = @name AND namespace = @namespace
		LIMIT 1
	`

	var cj CronJob
	err := Pool.QueryRow(ctx, query,
		pgx.StrictNamedArgs{
			"name":      name,
			"namespace": namespace,
		},
	).Scan(
		&cj.ID, &cj.Name, &cj.Namespace, &cj.UID,
		&cj.Schedule, &cj.Suspend,
		&cj.Labels, &cj.Annotations,
		&cj.CreatedAt, &cj.UpdatedAt, &cj.SyncedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("cronjob", fmt.Sprintf("%s/%s", namespace, name))
		}
		return nil, fmt.Errorf("failed to query cronjob by name: %w", err)
	}

	return &cj, nil
}

// PruneCronJobs deletes CronJobs not listed in uids.
// Empty uids deletes every CronJob in the namespace.
func PruneCronJobs(ctx context.Context, namespace string, uids []string) error {
	var err error
	if len(uids) == 0 {
		_, err = Pool.Exec(ctx,
			`DELETE FROM cronjobs WHERE namespace = @namespace`,
			pgx.StrictNamedArgs{"namespace": namespace},
		)
	} else {
		_, err = Pool.Exec(ctx,
			`DELETE FROM cronjobs WHERE namespace = @namespace AND NOT (uid = ANY(@uids::text[]))`,
			pgx.StrictNamedArgs{
				"namespace": namespace,
				"uids":      uids,
			},
		)
	}
	if err != nil {
		return fmt.Errorf("failed to prune cronjobs: %w", err)
	}
	return nil
}
