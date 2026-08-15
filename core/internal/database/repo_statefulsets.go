package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/apperrors"
)

func UpsertStatefulSetsBatch(ctx context.Context, statefulsets []*StatefulSet) error {
	if len(statefulsets) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO statefulsets (
			name, namespace, uid, replicas, ready_replicas,
			labels, annotations, created_at, synced_at
		) VALUES (
			@name, @namespace, @uid, @replicas, @ready_replicas,
			@labels, @annotations, @created_at, @synced_at
		)
		ON CONFLICT (namespace, name) DO UPDATE SET
			uid = EXCLUDED.uid,
			replicas = EXCLUDED.replicas,
			ready_replicas = EXCLUDED.ready_replicas,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, statefulset := range statefulsets {
		batch.Queue(query, pgx.StrictNamedArgs{
			"name":           statefulset.Name,
			"namespace":      statefulset.Namespace,
			"uid":            statefulset.UID,
			"replicas":       statefulset.Replicas,
			"ready_replicas": statefulset.ReadyReplicas,
			"labels":         statefulset.Labels,
			"annotations":    statefulset.Annotations,
			"created_at":     statefulset.CreatedAt,
			"synced_at":      statefulset.SyncedAt,
		})
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

	return nil
}

func GetStatefulSetsByNamespace(ctx context.Context, namespace string) ([]*StatefulSet, error) {
	query := `
		SELECT id, name, namespace, uid, replicas, ready_replicas,
		       labels, annotations, created_at, updated_at, synced_at
		FROM statefulsets
		WHERE namespace = @namespace
		ORDER BY name ASC
	`

	rows, err := Pool.Query(ctx, query,
		pgx.StrictNamedArgs{"namespace": namespace},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query statefulsets by namespace: %w", err)
	}
	defer rows.Close()

	statefulsets := make([]*StatefulSet, 0)
	for rows.Next() {
		var sts StatefulSet
		if err := rows.Scan(
			&sts.ID, &sts.Name, &sts.Namespace, &sts.UID,
			&sts.Replicas, &sts.ReadyReplicas,
			&sts.Labels, &sts.Annotations,
			&sts.CreatedAt, &sts.UpdatedAt, &sts.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan statefulset: %w", err)
		}
		statefulsets = append(statefulsets, &sts)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate statefulsets: %w", err)
	}

	return statefulsets, nil
}

func GetStatefulSetByName(ctx context.Context, name string, namespace string) (*StatefulSet, error) {
	query := `
		SELECT id, name, namespace, uid, replicas, ready_replicas,
		       labels, annotations, created_at, updated_at, synced_at
		FROM statefulsets
		WHERE name = @name AND namespace = @namespace
		LIMIT 1
	`

	var sts StatefulSet
	err := Pool.QueryRow(ctx, query,
		pgx.StrictNamedArgs{
			"name":      name,
			"namespace": namespace,
		},
	).Scan(
		&sts.ID, &sts.Name, &sts.Namespace, &sts.UID,
		&sts.Replicas, &sts.ReadyReplicas,
		&sts.Labels, &sts.Annotations,
		&sts.CreatedAt, &sts.UpdatedAt, &sts.SyncedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("statefulset", fmt.Sprintf("%s/%s", namespace, name))
		}
		return nil, fmt.Errorf("failed to query statefulset by name: %w", err)
	}

	return &sts, nil
}

// PruneStatefulSets deletes StatefulSets not listed in uids.
// Empty uids deletes every StatefulSet in the namespace.
func PruneStatefulSets(ctx context.Context, namespace string, uids []string) error {
	var err error
	if len(uids) == 0 {
		_, err = Pool.Exec(ctx,
			`DELETE FROM statefulsets WHERE namespace = @namespace`,
			pgx.StrictNamedArgs{"namespace": namespace},
		)
	} else {
		_, err = Pool.Exec(ctx,
			`DELETE FROM statefulsets WHERE namespace = @namespace AND NOT (uid = ANY(@uids::text[]))`,
			pgx.StrictNamedArgs{
				"namespace": namespace,
				"uids":      uids,
			},
		)
	}
	if err != nil {
		return fmt.Errorf("failed to prune statefulsets: %w", err)
	}
	return nil
}
