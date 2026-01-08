package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/internal/logger"
)

// Upserts multiple statefulsets in a batch transaction
func UpsertStatefulSetsBatch(ctx context.Context, statefulsets []*StatefulSet) error {
	if len(statefulsets) == 0 {
		return nil
	}

	log := logger.WithComponent("database")
	log.Debug().Int("count", len(statefulsets)).Msg("Upserting statefulsets batch")

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO statefulsets (
			name, namespace, uid, replicas, ready_replicas,
			labels, annotations, created_at, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
			namespace = EXCLUDED.namespace,
			replicas = EXCLUDED.replicas,
			ready_replicas = EXCLUDED.ready_replicas,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, statefulset := range statefulsets {
		batch.Queue(query,
			statefulset.Name, statefulset.Namespace, statefulset.UID,
			statefulset.Replicas, statefulset.ReadyReplicas,
			statefulset.Labels, statefulset.Annotations, statefulset.CreatedAt, statefulset.SyncedAt,
		)
	}

	results := tx.SendBatch(ctx, batch)

	for i := range statefulsets {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for statefulset %d: %w", i, err)
		}
	}

	if err := results.Close(); err != nil {
		return fmt.Errorf("failed to close batch results: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug().Int("count", len(statefulsets)).Msg("Statefulsets upserted successfully")
	return nil
}

// Deletes statefulsets that haven't been synced since the specified time
func DeleteStatefulSetsNotSyncedSince(ctx context.Context, since time.Time) error {
	log := logger.WithComponent("database")

	query := `DELETE FROM statefulsets WHERE synced_at < $1`
	result, err := Pool.Exec(ctx, query, since)
	if err != nil {
		return fmt.Errorf("failed to delete stale statefulsets: %w", err)
	}

	deleted := result.RowsAffected()
	if deleted > 0 {
		log.Debug().Int64("count", deleted).Msg("Stale statefulsets deleted")
	}

	return nil
}
