package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/internal/logger"
)

// Upserts node metadata into the database
func UpsertNode(ctx context.Context, node *Node) error {
	log := logger.WithComponent("database")
	log.Debug().
		Str("name", node.Name).
		Msg("Upserting node")

	query := `
		INSERT INTO nodes (
			name, uid, internal_ip, external_ip, os_image, kernel_version,
			container_runtime_version, kubelet_version, cpu_capacity, memory_capacity,
			cpu_allocatable, memory_allocatable, labels, annotations, conditions, created_at, synced_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
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

	_, err := Pool.Exec(ctx, query,
		node.Name, node.UID, node.InternalIP, node.ExternalIP, node.OSImage,
		node.KernelVersion, node.ContainerRuntimeVersion, node.KubeletVersion,
		node.CPUCapacity, node.MemoryCapacity, node.CPUAllocatable, node.MemoryAllocatable,
		node.Labels, node.Annotations, node.Conditions, node.CreatedAt, node.SyncedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert node: %w", err)
	}

	log.Debug().
		Str("name", node.Name).
		Msg("Node upserted successfully")

	return nil
}

// Upserts multiple nodes in a batch transaction
func UpsertNodesBatch(ctx context.Context, nodes []*Node) error {
	if len(nodes) == 0 {
		return nil
	}

	log := logger.WithComponent("database")
	log.Debug().Int("count", len(nodes)).Msg("Upserting nodes batch")

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
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
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
		batch.Queue(query,
			node.Name, node.UID, node.InternalIP, node.ExternalIP, node.OSImage,
			node.KernelVersion, node.ContainerRuntimeVersion, node.KubeletVersion,
			node.CPUCapacity, node.MemoryCapacity, node.CPUAllocatable, node.MemoryAllocatable,
			node.Labels, node.Annotations, node.Conditions, node.CreatedAt, node.SyncedAt,
		)
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

	log.Debug().Int("count", len(nodes)).Msg("Nodes upserted successfully")
	return nil
}

// Deletes nodes that haven't been synced since the specified time
func DeleteNodesNotSyncedSince(ctx context.Context, since time.Time) error {
	log := logger.WithComponent("database")

	query := `DELETE FROM nodes WHERE synced_at < $1`
	result, err := Pool.Exec(ctx, query, since)
	if err != nil {
		return fmt.Errorf("failed to delete stale nodes: %w", err)
	}

	deleted := result.RowsAffected()
	if deleted > 0 {
		log.Debug().Int64("count", deleted).Msg("Stale nodes deleted")
	}

	return nil
}
