package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/internal/logger"
)

// Upserts daemonset metadata into the database
func UpsertDaemonSet(ctx context.Context, daemonset *DaemonSet) error {
	log := logger.WithComponent("database")
	log.Debug().
		Str("name", daemonset.Name).
		Str("namespace", daemonset.Namespace).
		Msg("Upserting daemonset")

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

	_, err := Pool.Exec(ctx, query,
		daemonset.Name, daemonset.Namespace, daemonset.UID,
		daemonset.DesiredNumberScheduled, daemonset.NumberReady, daemonset.NumberAvailable,
		daemonset.Labels, daemonset.Annotations, daemonset.CreatedAt, daemonset.SyncedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert daemonset: %w", err)
	}

	log.Debug().
		Str("name", daemonset.Name).
		Str("namespace", daemonset.Namespace).
		Msg("Daemonset upserted successfully")

	return nil
}

// Upserts multiple daemonsets in a batch transaction
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

// Deletes daemonsets that haven't been synced since the specified time
func DeleteDaemonSetsNotSyncedSince(ctx context.Context, since time.Time) error {
	log := logger.WithComponent("database")

	query := `DELETE FROM daemonsets WHERE synced_at < $1`
	result, err := Pool.Exec(ctx, query, since)
	if err != nil {
		return fmt.Errorf("failed to delete stale daemonsets: %w", err)
	}

	deleted := result.RowsAffected()
	if deleted > 0 {
		log.Debug().Int64("count", deleted).Msg("Stale daemonsets deleted")
	}

	return nil
}
