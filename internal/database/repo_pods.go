package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/internal/logger"
)

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

// Gets a pod by name and namespace
func GetPodByName(ctx context.Context, name string, namespace string) (*Pod, error) {
	query := `
		SELECT id, name, namespace, uid, node_name, phase, restart_policy,
		       labels, annotations, owner_kind, owner_name,
		       created_at, updated_at, synced_at
		FROM pods
		WHERE name = $1 AND namespace = $2
		LIMIT 1
	`

	var p Pod
	err := Pool.QueryRow(ctx, query, name, namespace).Scan(
		&p.ID, &p.Name, &p.Namespace, &p.UID, &p.NodeName, &p.Phase, &p.RestartPolicy,
		&p.Labels, &p.Annotations, &p.OwnerKind, &p.OwnerName,
		&p.CreatedAt, &p.UpdatedAt, &p.SyncedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("pod %s/%s not found", namespace, name)
		}
		return nil, fmt.Errorf("failed to query pod by name: %w", err)
	}

	return &p, nil
}

// Gets all pods for a specific workload (by owner kind and name)
func GetPodsByWorkload(ctx context.Context, workloadType string, workloadName string, namespace string) ([]*Pod, error) {
	query := `
		SELECT id, name, namespace, uid, node_name, phase, restart_policy,
		       labels, annotations, owner_kind, owner_name,
		       created_at, updated_at, synced_at
		FROM pods
		WHERE owner_kind = $1 AND owner_name = $2 AND namespace = $3
		ORDER BY name
	`

	rows, err := Pool.Query(ctx, query, workloadType, workloadName, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to query pods by workload: %w", err)
	}
	defer rows.Close()

	pods := make([]*Pod, 0)
	for rows.Next() {
		var p Pod
		err := rows.Scan(
			&p.ID, &p.Name, &p.Namespace, &p.UID, &p.NodeName, &p.Phase, &p.RestartPolicy,
			&p.Labels, &p.Annotations, &p.OwnerKind, &p.OwnerName,
			&p.CreatedAt, &p.UpdatedAt, &p.SyncedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pod: %w", err)
		}
		pods = append(pods, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pods: %w", err)
	}

	return pods, nil
}

// Gets standalone pods (pods without owner) in a namespace
func GetStandalonePodsByNamespace(ctx context.Context, namespace string) ([]*Pod, error) {
	query := `
		SELECT id, name, namespace, uid, node_name, phase, restart_policy,
		       labels, annotations, owner_kind, owner_name,
		       created_at, updated_at, synced_at
		FROM pods
		WHERE namespace = $1 AND (owner_kind IS NULL OR owner_kind = '')
		ORDER BY name
	`

	rows, err := Pool.Query(ctx, query, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to query standalone pods by namespace: %w", err)
	}
	defer rows.Close()

	pods := make([]*Pod, 0)
	for rows.Next() {
		var p Pod
		err := rows.Scan(
			&p.ID, &p.Name, &p.Namespace, &p.UID, &p.NodeName, &p.Phase, &p.RestartPolicy,
			&p.Labels, &p.Annotations, &p.OwnerKind, &p.OwnerName,
			&p.CreatedAt, &p.UpdatedAt, &p.SyncedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pod: %w", err)
		}
		pods = append(pods, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pods: %w", err)
	}

	return pods, nil
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
