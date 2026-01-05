package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/internal/logger"
)

// Upserts pod metadata into the database
func UpsertPod(ctx context.Context, pod *Pod) error {
	log := logger.WithComponent("database")
	log.Debug().
		Str("name", pod.Name).
		Str("namespace", pod.Namespace).
		Msg("Upserting pod")

	query := `
		INSERT INTO pods (
			name, namespace, uid, node_name, phase, restart_policy,
			labels, annotations, owner_kind, owner_name, created_at, synced_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
			namespace = EXCLUDED.namespace,
			node_name = EXCLUDED.node_name,
			phase = EXCLUDED.phase,
			restart_policy = EXCLUDED.restart_policy,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			owner_kind = EXCLUDED.owner_kind,
			owner_name = EXCLUDED.owner_name,
			synced_at = EXCLUDED.synced_at
	`

	_, err := Pool.Exec(ctx, query,
		pod.Name, pod.Namespace, pod.UID, pod.NodeName, pod.Phase, pod.RestartPolicy,
		pod.Labels, pod.Annotations, pod.OwnerKind, pod.OwnerName,
		pod.CreatedAt, pod.SyncedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert pod: %w", err)
	}

	log.Debug().
		Str("name", pod.Name).
		Str("namespace", pod.Namespace).
		Msg("Pod upserted successfully")

	return nil
}

// Upserts multiple pods in a batch transaction
func UpsertPodsBatch(ctx context.Context, pods []*Pod) error {
	if len(pods) == 0 {
		return nil
	}

	log := logger.WithComponent("database")
	log.Debug().Int("count", len(pods)).Msg("Upserting pods batch")

	// Start transaction
	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO pods (
			name, namespace, uid, node_name, phase, restart_policy,
			labels, annotations, owner_kind, owner_name, created_at, synced_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
			namespace = EXCLUDED.namespace,
			node_name = EXCLUDED.node_name,
			phase = EXCLUDED.phase,
			restart_policy = EXCLUDED.restart_policy,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			owner_kind = EXCLUDED.owner_kind,
			owner_name = EXCLUDED.owner_name,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, pod := range pods {
		batch.Queue(query,
			pod.Name, pod.Namespace, pod.UID, pod.NodeName, pod.Phase, pod.RestartPolicy,
			pod.Labels, pod.Annotations, pod.OwnerKind, pod.OwnerName,
			pod.CreatedAt, pod.SyncedAt,
		)
	}

	results := tx.SendBatch(ctx, batch)

	for i := range pods {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for pod %d: %w", i, err)
		}
	}

	if err := results.Close(); err != nil {
		return fmt.Errorf("failed to close batch results: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug().Int("count", len(pods)).Msg("Pods upserted successfully")
	return nil
}

// Deletes pods that haven't been synced since the specified time
func DeletePodsNotSyncedSince(ctx context.Context, since time.Time) error {
	log := logger.WithComponent("database")

	query := `DELETE FROM pods WHERE synced_at < $1`
	result, err := Pool.Exec(ctx, query, since)
	if err != nil {
		return fmt.Errorf("failed to delete stale pods: %w", err)
	}

	deleted := result.RowsAffected()
	if deleted > 0 {
		log.Debug().Int64("count", deleted).Msg("Stale pods deleted")
	}

	return nil
}
