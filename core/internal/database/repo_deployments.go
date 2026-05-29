package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/logger"
)

// UpsertDeploymentsBatch inserts or updates deployments by Kubernetes UID inside one transaction.
func UpsertDeploymentsBatch(ctx context.Context, deployments []*Deployment) error {
	if len(deployments) == 0 {
		return nil
	}

	log := logger.WithComponent("database")
	log.Debug().Int("count", len(deployments)).Msg("Upserting deployments batch")

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO deployments (
			name, namespace, uid, replicas, ready_replicas, available_replicas,
			labels, annotations, created_at, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
			namespace = EXCLUDED.namespace,
			replicas = EXCLUDED.replicas,
			ready_replicas = EXCLUDED.ready_replicas,
			available_replicas = EXCLUDED.available_replicas,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, deployment := range deployments {
		batch.Queue(query,
			deployment.Name, deployment.Namespace, deployment.UID,
			deployment.Replicas, deployment.ReadyReplicas, deployment.AvailableReplicas,
			deployment.Labels, deployment.Annotations, deployment.CreatedAt, deployment.SyncedAt,
		)
	}

	results := tx.SendBatch(ctx, batch)

	for i := range deployments {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for deployment %d: %w", i, err)
		}
	}

	if err := results.Close(); err != nil {
		return fmt.Errorf("failed to close batch results: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug().Int("count", len(deployments)).Msg("Deployments upserted successfully")
	return nil
}

// GetDeploymentsByNamespace returns all the deployments in the namespace, ordered by name.
func GetDeploymentsByNamespace(ctx context.Context, namespace string) ([]*Deployment, error) {
	query := `
		SELECT id, name, namespace, uid, replicas, ready_replicas, available_replicas,
		       labels, annotations, created_at, updated_at, synced_at
		FROM deployments
		WHERE namespace = $1
		ORDER BY name ASC
	`

	rows, err := Pool.Query(ctx, query, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to query deployments by namespace: %w", err)
	}
	defer rows.Close()

	deployments := make([]*Deployment, 0)
	for rows.Next() {
		var dep Deployment
		if err := rows.Scan(
			&dep.ID, &dep.Name, &dep.Namespace, &dep.UID,
			&dep.Replicas, &dep.ReadyReplicas, &dep.AvailableReplicas,
			&dep.Labels, &dep.Annotations,
			&dep.CreatedAt, &dep.UpdatedAt, &dep.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan deployment: %w", err)
		}
		deployments = append(deployments, &dep)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate deployments: %w", err)
	}

	return deployments, nil
}

// GetDeploymentByName returns a deployment by name in the namespace.
func GetDeploymentByName(ctx context.Context, name string, namespace string) (*Deployment, error) {
	query := `
		SELECT id, name, namespace, uid, replicas, ready_replicas, available_replicas,
		       labels, annotations, created_at, updated_at, synced_at
		FROM deployments
		WHERE name = $1 AND namespace = $2
		LIMIT 1
	`

	var dep Deployment
	err := Pool.QueryRow(ctx, query, name, namespace).Scan(
		&dep.ID, &dep.Name, &dep.Namespace, &dep.UID,
		&dep.Replicas, &dep.ReadyReplicas, &dep.AvailableReplicas,
		&dep.Labels, &dep.Annotations,
		&dep.CreatedAt, &dep.UpdatedAt, &dep.SyncedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("deployment", fmt.Sprintf("%s/%s", namespace, name))
		}
		return nil, fmt.Errorf("failed to query deployment by name: %w", err)
	}

	return &dep, nil
}

// PruneDeployments deletes deployments whose UID is not in uids in the namespace.
// Empty uids deletes all deployments in that namespace.
func PruneDeployments(ctx context.Context, namespace string, uids []string) error {
	if len(uids) == 0 {
		_, err := Pool.Exec(ctx, `DELETE FROM deployments WHERE namespace = $1`, namespace)
		return err
	}
	_, err := Pool.Exec(ctx, `DELETE FROM deployments WHERE namespace = $1 AND NOT (uid = ANY($2::text[]))`, namespace, uids)
	return err
}
