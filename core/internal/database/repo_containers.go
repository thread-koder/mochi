package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func UpsertContainersBatch(ctx context.Context, containers []*Container) error {
	if len(containers) == 0 {
		return nil
	}

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
			@name, @pod_uid, @pod_name, @namespace, @image, @image_pull_policy, @ports,
			@cpu_request, @cpu_limit, @memory_request, @memory_limit,
			@created_at, @synced_at
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
		batch.Queue(query, pgx.StrictNamedArgs{
			"name":              container.Name,
			"pod_uid":           container.PodUID,
			"pod_name":          container.PodName,
			"namespace":         container.Namespace,
			"image":             container.Image,
			"image_pull_policy": container.ImagePullPolicy,
			"ports":             container.Ports,
			"cpu_request":       container.CPURequest,
			"cpu_limit":         container.CPULimit,
			"memory_request":    container.MemoryRequest,
			"memory_limit":      container.MemoryLimit,
			"created_at":        container.CreatedAt,
			"synced_at":         container.SyncedAt,
		})
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

	return nil
}

func GetContainersByPodUID(ctx context.Context, podUID string) ([]*Container, error) {
	query := `
		SELECT id, name, pod_uid, pod_name, namespace, image, image_pull_policy, ports,
		       cpu_request, cpu_limit, memory_request, memory_limit,
		       created_at, updated_at, synced_at
		FROM containers
		WHERE pod_uid = @pod_uid
		ORDER BY name
	`

	rows, err := Pool.Query(ctx, query,
		pgx.StrictNamedArgs{"pod_uid": podUID},
	)
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

// PruneContainers deletes containers not listed in podUIDs.
// Empty podUIDs deletes every container in the namespace.
func PruneContainers(ctx context.Context, namespace string, podUIDs []string) error {
	var err error
	if len(podUIDs) == 0 {
		_, err = Pool.Exec(ctx,
			`DELETE FROM containers WHERE namespace = @namespace`,
			pgx.StrictNamedArgs{"namespace": namespace},
		)
	} else {
		_, err = Pool.Exec(ctx,
			`DELETE FROM containers WHERE namespace = @namespace AND NOT (pod_uid = ANY(@pod_uids::text[]))`,
			pgx.StrictNamedArgs{
				"namespace": namespace,
				"pod_uids":  podUIDs,
			},
		)
	}
	if err != nil {
		return fmt.Errorf("failed to prune containers: %w", err)
	}
	return nil
}
