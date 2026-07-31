package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/apperrors"
)

func UpsertReplicaSetsBatch(ctx context.Context, replicasets []*ReplicaSet) error {
	if len(replicasets) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO replicasets (
			name, namespace, uid, replicas, ready_replicas,
			owner_kind, owner_name, labels, annotations, created_at, synced_at
		) VALUES (
			@name, @namespace, @uid, @replicas, @ready_replicas,
			@owner_kind, @owner_name, @labels, @annotations, @created_at, @synced_at
		)
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
		batch.Queue(query, pgx.StrictNamedArgs{
			"name":           replicaset.Name,
			"namespace":      replicaset.Namespace,
			"uid":            replicaset.UID,
			"replicas":       replicaset.Replicas,
			"ready_replicas": replicaset.ReadyReplicas,
			"owner_kind":     replicaset.OwnerKind,
			"owner_name":     replicaset.OwnerName,
			"labels":         replicaset.Labels,
			"annotations":    replicaset.Annotations,
			"created_at":     replicaset.CreatedAt,
			"synced_at":      replicaset.SyncedAt,
		})
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

	return nil
}

func GetReplicaSetsByDeployment(ctx context.Context, deploymentName, namespace string) ([]*ReplicaSet, error) {
	query := `
		SELECT id, name, namespace, uid, replicas, ready_replicas,
		       owner_kind, owner_name, labels, annotations, created_at, updated_at, synced_at
		FROM replicasets
		WHERE namespace = @namespace AND owner_kind = 'Deployment' AND owner_name = @owner_name
		ORDER BY name ASC
	`

	rows, err := Pool.Query(ctx, query,
		pgx.StrictNamedArgs{
			"namespace":  namespace,
			"owner_name": deploymentName,
		},
	)
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

func GetReplicaSetByName(ctx context.Context, name string, namespace string) (*ReplicaSet, error) {
	query := `
		SELECT id, name, namespace, uid, replicas, ready_replicas,
		       owner_kind, owner_name, labels, annotations, created_at, updated_at, synced_at
		FROM replicasets
		WHERE name = @name AND namespace = @namespace
		LIMIT 1
	`

	var rs ReplicaSet
	err := Pool.QueryRow(ctx, query,
		pgx.StrictNamedArgs{
			"name":      name,
			"namespace": namespace,
		},
	).Scan(
		&rs.ID, &rs.Name, &rs.Namespace, &rs.UID,
		&rs.Replicas, &rs.ReadyReplicas,
		&rs.OwnerKind, &rs.OwnerName,
		&rs.Labels, &rs.Annotations,
		&rs.CreatedAt, &rs.UpdatedAt, &rs.SyncedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("replicaset", fmt.Sprintf("%s/%s", namespace, name))
		}
		return nil, fmt.Errorf("failed to query replicaset by name: %w", err)
	}

	return &rs, nil
}

// PruneReplicaSets deletes ReplicaSets not listed in uids.
// Empty uids deletes every ReplicaSet in the namespace.
func PruneReplicaSets(ctx context.Context, namespace string, uids []string) error {
	var err error
	if len(uids) == 0 {
		_, err = Pool.Exec(ctx,
			`DELETE FROM replicasets WHERE namespace = @namespace`,
			pgx.StrictNamedArgs{"namespace": namespace},
		)
	} else {
		_, err = Pool.Exec(ctx,
			`DELETE FROM replicasets WHERE namespace = @namespace AND NOT (uid = ANY(@uids::text[]))`,
			pgx.StrictNamedArgs{
				"namespace": namespace,
				"uids":      uids,
			},
		)
	}
	if err != nil {
		return fmt.Errorf("failed to prune replicasets: %w", err)
	}
	return nil
}
