package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/logger"
)

// UpsertDaemonSetsBatch inserts or updates DaemonSets by Kubernetes UID inside one transaction.
func UpsertDaemonSetsBatch(ctx context.Context, daemonsets []*DaemonSet) error {
	if len(daemonsets) == 0 {
		return nil
	}

	log := logger.WithComponent("database")
	log.Debug().Int("count", len(daemonsets)).Msg("Upserting daemonsets batch")

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO daemonsets (
			name, namespace, uid, desired_number_scheduled, number_ready, number_available,
			labels, annotations, created_at, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
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
		batch.Queue(query,
			daemonset.Name, daemonset.Namespace, daemonset.UID,
			daemonset.DesiredNumberScheduled, daemonset.NumberReady, daemonset.NumberAvailable,
			daemonset.Labels, daemonset.Annotations, daemonset.CreatedAt, daemonset.SyncedAt,
		)
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

	log.Debug().Int("count", len(daemonsets)).Msg("Daemonsets upserted successfully")
	return nil
}

// GetDaemonSetsByNamespace returns all the DaemonSets in the namespace, ordered by name.
func GetDaemonSetsByNamespace(ctx context.Context, namespace string) ([]*DaemonSet, error) {
	query := `
		SELECT id, name, namespace, uid, desired_number_scheduled, number_ready, number_available,
		       labels, annotations, created_at, updated_at, synced_at
		FROM daemonsets
		WHERE namespace = $1
		ORDER BY name ASC
	`

	rows, err := Pool.Query(ctx, query, namespace)
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

// GetDaemonSetByName returns a DaemonSet by name in the namespace.
func GetDaemonSetByName(ctx context.Context, name string, namespace string) (*DaemonSet, error) {
	query := `
		SELECT id, name, namespace, uid, desired_number_scheduled, number_ready, number_available,
		       labels, annotations, created_at, updated_at, synced_at
		FROM daemonsets
		WHERE name = $1 AND namespace = $2
		LIMIT 1
	`

	var ds DaemonSet
	err := Pool.QueryRow(ctx, query, name, namespace).Scan(
		&ds.ID, &ds.Name, &ds.Namespace, &ds.UID,
		&ds.DesiredNumberScheduled, &ds.NumberReady, &ds.NumberAvailable,
		&ds.Labels, &ds.Annotations,
		&ds.CreatedAt, &ds.UpdatedAt, &ds.SyncedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("daemonset %s not found in namespace %s", name, namespace)
		}
		return nil, fmt.Errorf("failed to query daemonset by name: %w", err)
	}

	return &ds, nil
}

// PruneDaemonSets deletes DaemonSets whose UID is not in uids in the namespace.
// Empty uids deletes all DaemonSets in that namespace.
func PruneDaemonSets(ctx context.Context, namespace string, uids []string) error {
	if len(uids) == 0 {
		_, err := Pool.Exec(ctx, `DELETE FROM daemonsets WHERE namespace = $1`, namespace)
		return err
	}
	_, err := Pool.Exec(ctx, `DELETE FROM daemonsets WHERE namespace = $1 AND NOT (uid = ANY($2::text[]))`, namespace, uids)
	return err
}
