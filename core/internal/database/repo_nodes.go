package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func UpsertNodesBatch(ctx context.Context, nodes []*Node) error {
	if len(nodes) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO nodes (
			name, uid, internal_ip, external_ip, os_image, kernel_version,
			container_runtime_version, kubelet_version, cpu_capacity, memory_capacity,
			cpu_allocatable, memory_allocatable, labels, annotations, conditions, created_at, synced_at
		) VALUES (
			@name, @uid, @internal_ip, @external_ip, @os_image, @kernel_version,
			@container_runtime_version, @kubelet_version, @cpu_capacity, @memory_capacity,
			@cpu_allocatable, @memory_allocatable, @labels, @annotations, @conditions, @created_at, @synced_at
		)
		ON CONFLICT (name) DO UPDATE SET
			uid = EXCLUDED.uid,
			internal_ip = EXCLUDED.internal_ip,
			external_ip = EXCLUDED.external_ip,
			os_image = EXCLUDED.os_image,
			kernel_version = EXCLUDED.kernel_version,
			container_runtime_version = EXCLUDED.container_runtime_version,
			kubelet_version = EXCLUDED.kubelet_version,
			cpu_capacity = EXCLUDED.cpu_capacity,
			memory_capacity = EXCLUDED.memory_capacity,
			cpu_allocatable = EXCLUDED.cpu_allocatable,
			memory_allocatable = EXCLUDED.memory_allocatable,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			conditions = EXCLUDED.conditions,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, node := range nodes {
		batch.Queue(query, pgx.StrictNamedArgs{
			"name":                      node.Name,
			"uid":                       node.UID,
			"internal_ip":               node.InternalIP,
			"external_ip":               node.ExternalIP,
			"os_image":                  node.OSImage,
			"kernel_version":            node.KernelVersion,
			"container_runtime_version": node.ContainerRuntimeVersion,
			"kubelet_version":           node.KubeletVersion,
			"cpu_capacity":              node.CPUCapacity,
			"memory_capacity":           node.MemoryCapacity,
			"cpu_allocatable":           node.CPUAllocatable,
			"memory_allocatable":        node.MemoryAllocatable,
			"labels":                    node.Labels,
			"annotations":               node.Annotations,
			"conditions":                node.Conditions,
			"created_at":                node.CreatedAt,
			"synced_at":                 node.SyncedAt,
		})
	}

	results := tx.SendBatch(ctx, batch)

	for i := range nodes {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for node %d: %w", i, err)
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

// PruneNodes deletes nodes not listed in uids.
// Empty uids deletes all nodes.
func PruneNodes(ctx context.Context, uids []string) error {
	var err error
	if len(uids) == 0 {
		_, err = Pool.Exec(ctx, `DELETE FROM nodes`)
	} else {
		_, err = Pool.Exec(ctx,
			`DELETE FROM nodes WHERE NOT (uid = ANY(@uids::text[]))`,
			pgx.StrictNamedArgs{"uids": uids},
		)
	}
	if err != nil {
		return fmt.Errorf("failed to prune nodes: %w", err)
	}
	return nil
}
