package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/internal/logger"
)

// UpsertContainersBatch inserts or updates containers by their (pod_uid, name) inside one transaction.
func UpsertContainersBatch(ctx context.Context, containers []*Container) error {
	if len(containers) == 0 {
		return nil
	}

	log := logger.WithComponent("database")
	log.Debug().Int("count", len(containers)).Msg("Upserting containers batch")

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO containers (
			name, pod_uid, pod_name, namespace, image, image_pull_policy, ports,
			cpu_request, cpu_limit, memory_request, memory_limit,
			created_at, synced_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
		ON CONFLICT (pod_uid, name) DO UPDATE SET
			pod_name = EXCLUDED.pod_name,
			namespace = EXCLUDED.namespace,
			image = EXCLUDED.image,
			image_pull_policy = EXCLUDED.image_pull_policy,
			ports = EXCLUDED.ports,
			cpu_request = EXCLUDED.cpu_request,
			cpu_limit = EXCLUDED.cpu_limit,
			memory_request = EXCLUDED.memory_request,
			memory_limit = EXCLUDED.memory_limit,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, container := range containers {
		batch.Queue(query,
			container.Name, container.PodUID, container.PodName, container.Namespace,
			container.Image, container.ImagePullPolicy, container.Ports,
			container.CPURequest, container.CPULimit, container.MemoryRequest, container.MemoryLimit,
			container.CreatedAt, container.SyncedAt,
		)
	}

	results := tx.SendBatch(ctx, batch)

	for i := range containers {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for container %d: %w", i, err)
		}
	}

	if err := results.Close(); err != nil {
		return fmt.Errorf("failed to close batch results: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug().Int("count", len(containers)).Msg("Containers upserted successfully")
	return nil
}

// GetContainersByPodUID returns all the containers for one pod, ordered by name.
func GetContainersByPodUID(ctx context.Context, podUID string) ([]*Container, error) {
	query := `
		SELECT id, name, pod_uid, pod_name, namespace, image, image_pull_policy, ports,
		       cpu_request, cpu_limit, memory_request, memory_limit,
		       created_at, updated_at, synced_at
		FROM containers
		WHERE pod_uid = $1
		ORDER BY name
	`

	rows, err := Pool.Query(ctx, query, podUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query containers by pod UID: %w", err)
	}
	defer rows.Close()

	containers := make([]*Container, 0)
	for rows.Next() {
		var c Container
		err := rows.Scan(
			&c.ID, &c.Name, &c.PodUID, &c.PodName, &c.Namespace,
			&c.Image, &c.ImagePullPolicy, &c.Ports,
			&c.CPURequest, &c.CPULimit, &c.MemoryRequest, &c.MemoryLimit,
			&c.CreatedAt, &c.UpdatedAt, &c.SyncedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan container: %w", err)
		}
		containers = append(containers, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate containers: %w", err)
	}

	return containers, nil
}

// PruneContainers deletes containers whose pod_uid is not in podUIDs in the namespace.
// Empty podUIDs deletes all containers in that namespace (eg. containers that do not belong to any pod).
func PruneContainers(ctx context.Context, namespace string, podUIDs []string) error {
	if len(podUIDs) == 0 {
		_, err := Pool.Exec(ctx, `DELETE FROM containers WHERE namespace = $1`, namespace)
		return err
	}
	_, err := Pool.Exec(ctx, `DELETE FROM containers WHERE namespace = $1 AND NOT (pod_uid = ANY($2::text[]))`, namespace, podUIDs)
	return err
}
