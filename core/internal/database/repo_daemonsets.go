package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/apperrors"
)

func UpsertDaemonSetsBatch(ctx context.Context, daemonsets []*DaemonSet) error {
	if len(daemonsets) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO daemonsets (
			name, namespace, uid, desired_number_scheduled, number_ready, number_available,
			labels, annotations, created_at, synced_at
		) VALUES (
			@name, @namespace, @uid, @desired_number_scheduled, @number_ready, @number_available,
			@labels, @annotations, @created_at, @synced_at
		)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
			namespace = EXCLUDED.namespace,
			desired_number_scheduled = EXCLUDED.desired_number_scheduled,
			number_ready = EXCLUDED.number_ready,
			number_available = EXCLUDED.number_available,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, daemonset := range daemonsets {
		batch.Queue(query, pgx.StrictNamedArgs{
			"name":                     daemonset.Name,
			"namespace":                daemonset.Namespace,
			"uid":                      daemonset.UID,
			"desired_number_scheduled": daemonset.DesiredNumberScheduled,
			"number_ready":             daemonset.NumberReady,
			"number_available":         daemonset.NumberAvailable,
			"labels":                   daemonset.Labels,
			"annotations":              daemonset.Annotations,
			"created_at":               daemonset.CreatedAt,
			"synced_at":                daemonset.SyncedAt,
		})
	}

	results := tx.SendBatch(ctx, batch)

	for i := range daemonsets {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for daemonset %d: %w", i, err)
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

func GetDaemonSetsByNamespace(ctx context.Context, namespace string) ([]*DaemonSet, error) {
	query := `
		SELECT id, name, namespace, uid, desired_number_scheduled, number_ready, number_available,
		       labels, annotations, created_at, updated_at, synced_at
		FROM daemonsets
		WHERE namespace = @namespace
		ORDER BY name ASC
	`

	rows, err := Pool.Query(ctx, query,
		pgx.StrictNamedArgs{"namespace": namespace},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query daemonsets by namespace: %w", err)
	}
	defer rows.Close()

	daemonsets := make([]*DaemonSet, 0)
	for rows.Next() {
		var ds DaemonSet
		if err := rows.Scan(
			&ds.ID, &ds.Name, &ds.Namespace, &ds.UID,
			&ds.DesiredNumberScheduled, &ds.NumberReady, &ds.NumberAvailable,
			&ds.Labels, &ds.Annotations,
			&ds.CreatedAt, &ds.UpdatedAt, &ds.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan daemonset: %w", err)
		}
		daemonsets = append(daemonsets, &ds)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate daemonsets: %w", err)
	}

	return daemonsets, nil
}

func GetDaemonSetByName(ctx context.Context, name string, namespace string) (*DaemonSet, error) {
	query := `
		SELECT id, name, namespace, uid, desired_number_scheduled, number_ready, number_available,
		       labels, annotations, created_at, updated_at, synced_at
		FROM daemonsets
		WHERE name = @name AND namespace = @namespace
		LIMIT 1
	`

	var ds DaemonSet
	err := Pool.QueryRow(ctx, query,
		pgx.StrictNamedArgs{
			"name":      name,
			"namespace": namespace,
		},
	).Scan(
		&ds.ID, &ds.Name, &ds.Namespace, &ds.UID,
		&ds.DesiredNumberScheduled, &ds.NumberReady, &ds.NumberAvailable,
		&ds.Labels, &ds.Annotations,
		&ds.CreatedAt, &ds.UpdatedAt, &ds.SyncedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("daemonset", fmt.Sprintf("%s/%s", namespace, name))
		}
		return nil, fmt.Errorf("failed to query daemonset by name: %w", err)
	}

	return &ds, nil
}

// PruneDaemonSets deletes DaemonSets not listed in uids.
// Empty uids deletes every DaemonSet in the namespace.
func PruneDaemonSets(ctx context.Context, namespace string, uids []string) error {
	var err error
	if len(uids) == 0 {
		_, err = Pool.Exec(ctx,
			`DELETE FROM daemonsets WHERE namespace = @namespace`,
			pgx.StrictNamedArgs{"namespace": namespace},
		)
	} else {
		_, err = Pool.Exec(ctx,
			`DELETE FROM daemonsets WHERE namespace = @namespace AND NOT (uid = ANY(@uids::text[]))`,
			pgx.StrictNamedArgs{
				"namespace": namespace,
				"uids":      uids,
			},
		)
	}
	if err != nil {
		return fmt.Errorf("failed to prune daemonsets: %w", err)
	}
	return nil
}
