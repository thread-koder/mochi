package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/logger"
)

// UpsertStatefulSetsBatch inserts or updates StatefulSets by Kubernetes UID inside one transaction.
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

// GetStatefulSetsByNamespace returns all the StatefulSets in the namespace, ordered by name.
func GetStatefulSetsByNamespace(ctx context.Context, namespace string) ([]*StatefulSet, error) {
	query := `
		SELECT id, name, namespace, uid, replicas, ready_replicas,
		       labels, annotations, created_at, updated_at, synced_at
		FROM statefulsets
		WHERE namespace = $1
		ORDER BY name ASC
	`

	rows, err := Pool.Query(ctx, query, namespace)
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

// GetStatefulSetByName returns a StatefulSet by name in the namespace.
func GetStatefulSetByName(ctx context.Context, name string, namespace string) (*StatefulSet, error) {
	query := `
		SELECT id, name, namespace, uid, replicas, ready_replicas,
		       labels, annotations, created_at, updated_at, synced_at
		FROM statefulsets
		WHERE name = $1 AND namespace = $2
		LIMIT 1
	`

	var sts StatefulSet
	err := Pool.QueryRow(ctx, query, name, namespace).Scan(
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

// PruneStatefulSets deletes StatefulSets whose UID is not in uids in the namespace.
// Empty uids deletes all StatefulSets in that namespace.
func PruneStatefulSets(ctx context.Context, namespace string, uids []string) error {
	if len(uids) == 0 {
		_, err := Pool.Exec(ctx, `DELETE FROM statefulsets WHERE namespace = $1`, namespace)
		return err
	}
	_, err := Pool.Exec(ctx, `DELETE FROM statefulsets WHERE namespace = $1 AND NOT (uid = ANY($2::text[]))`, namespace, uids)
	return err
}
