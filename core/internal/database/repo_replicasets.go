package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/logger"
)

// UpsertReplicaSetsBatch inserts or updates ReplicaSets by Kubernetes UID inside one transaction.
func UpsertReplicaSetsBatch(ctx context.Context, replicasets []*ReplicaSet) error {
	if len(replicasets) == 0 {
		return nil
	}

	log := logger.WithComponent("database")
	log.Debug().Int("count", len(replicasets)).Msg("Upserting replicasets batch")

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO replicasets (
			name, namespace, uid, replicas, ready_replicas,
			owner_kind, owner_name, labels, annotations, created_at, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
			namespace = EXCLUDED.namespace,
			replicas = EXCLUDED.replicas,
			ready_replicas = EXCLUDED.ready_replicas,
			owner_kind = EXCLUDED.owner_kind,
			owner_name = EXCLUDED.owner_name,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, replicaset := range replicasets {
		batch.Queue(query,
			replicaset.Name, replicaset.Namespace, replicaset.UID,
			replicaset.Replicas, replicaset.ReadyReplicas,
			replicaset.OwnerKind, replicaset.OwnerName,
			replicaset.Labels, replicaset.Annotations, replicaset.CreatedAt, replicaset.SyncedAt,
		)
	}

	results := tx.SendBatch(ctx, batch)

	for i := range replicasets {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for replicaset %d: %w", i, err)
		}
	}

	if err := results.Close(); err != nil {
		return fmt.Errorf("failed to close batch results: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug().Int("count", len(replicasets)).Msg("Replicasets upserted successfully")
	return nil
}

// GetReplicaSetsByDeployment returns the ReplicaSets owned by the given Deployment (owner_kind/name match).
func GetReplicaSetsByDeployment(ctx context.Context, deploymentName, namespace string) ([]*ReplicaSet, error) {
	query := `
		SELECT id, name, namespace, uid, replicas, ready_replicas,
		       owner_kind, owner_name, labels, annotations, created_at, updated_at, synced_at
		FROM replicasets
		WHERE namespace = $1 AND owner_kind = 'Deployment' AND owner_name = $2
		ORDER BY name ASC
	`

	rows, err := Pool.Query(ctx, query, namespace, deploymentName)
	if err != nil {
		return nil, fmt.Errorf("failed to query replicasets by deployment: %w", err)
	}
	defer rows.Close()

	replicasets := make([]*ReplicaSet, 0)
	for rows.Next() {
		var rs ReplicaSet
		if err := rows.Scan(
			&rs.ID, &rs.Name, &rs.Namespace, &rs.UID,
			&rs.Replicas, &rs.ReadyReplicas,
			&rs.OwnerKind, &rs.OwnerName,
			&rs.Labels, &rs.Annotations,
			&rs.CreatedAt, &rs.UpdatedAt, &rs.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan replicaset: %w", err)
		}
		replicasets = append(replicasets, &rs)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate replicasets: %w", err)
	}

	return replicasets, nil
}

// PruneReplicaSets deletes ReplicaSets whose UID is not in uids in the namespace.
// Empty uids deletes all ReplicaSets in that namespace.
func PruneReplicaSets(ctx context.Context, namespace string, uids []string) error {
	if len(uids) == 0 {
		_, err := Pool.Exec(ctx, `DELETE FROM replicasets WHERE namespace = $1`, namespace)
		return err
	}
	_, err := Pool.Exec(ctx, `DELETE FROM replicasets WHERE namespace = $1 AND NOT (uid = ANY($2::text[]))`, namespace, uids)
	return err
}
