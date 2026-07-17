package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/apperrors"
)

func UpsertDeploymentsBatch(ctx context.Context, deployments []*Deployment) error {
	if len(deployments) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO deployments (
			name, namespace, uid, replicas, ready_replicas, available_replicas,
			labels, annotations, created_at, synced_at
		) VALUES (
			@name, @namespace, @uid, @replicas, @ready_replicas, @available_replicas,
			@labels, @annotations, @created_at, @synced_at
		)
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
		batch.Queue(query, pgx.StrictNamedArgs{
			"name":               deployment.Name,
			"namespace":          deployment.Namespace,
			"uid":                deployment.UID,
			"replicas":           deployment.Replicas,
			"ready_replicas":     deployment.ReadyReplicas,
			"available_replicas": deployment.AvailableReplicas,
			"labels":             deployment.Labels,
			"annotations":        deployment.Annotations,
			"created_at":         deployment.CreatedAt,
			"synced_at":          deployment.SyncedAt,
		})
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

	return nil
}

func GetDeploymentsByNamespace(ctx context.Context, namespace string) ([]*Deployment, error) {
	query := `
		SELECT id, name, namespace, uid, replicas, ready_replicas, available_replicas,
		       labels, annotations, created_at, updated_at, synced_at
		FROM deployments
		WHERE namespace = @namespace
		ORDER BY name ASC
	`

	rows, err := Pool.Query(ctx, query,
		pgx.StrictNamedArgs{"namespace": namespace},
	)
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

func GetDeploymentByName(ctx context.Context, name string, namespace string) (*Deployment, error) {
	query := `
		SELECT id, name, namespace, uid, replicas, ready_replicas, available_replicas,
		       labels, annotations, created_at, updated_at, synced_at
		FROM deployments
		WHERE name = @name AND namespace = @namespace
		LIMIT 1
	`

	var dep Deployment
	err := Pool.QueryRow(ctx, query,
		pgx.StrictNamedArgs{
			"name":      name,
			"namespace": namespace,
		},
	).Scan(
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

// PruneDeployments deletes deployments not listed in uids.
// Empty uids deletes every deployment in the namespace.
func PruneDeployments(ctx context.Context, namespace string, uids []string) error {
	if len(uids) == 0 {
		_, err := Pool.Exec(ctx,
			`DELETE FROM deployments WHERE namespace = @namespace`,
			pgx.StrictNamedArgs{"namespace": namespace},
		)
		return err
	}
	_, err := Pool.Exec(ctx,
		`DELETE FROM deployments WHERE namespace = @namespace AND NOT (uid = ANY(@uids::text[]))`,
		pgx.StrictNamedArgs{
			"namespace": namespace,
			"uids":      uids,
		},
	)
	return err
}
