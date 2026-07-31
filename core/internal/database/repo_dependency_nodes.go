package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func UpsertDependencyNodesBatch(ctx context.Context, nodes []*DependencyNode) error {
	if len(nodes) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO dependency_nodes (
			kind, namespace, name, metadata, first_seen_at, last_seen_at
		) VALUES (
			@kind, @namespace, @name, @metadata, @first_seen_at, @last_seen_at
		)
		ON CONFLICT (kind, namespace, name) DO UPDATE SET
			metadata = EXCLUDED.metadata,
			last_seen_at = EXCLUDED.last_seen_at
		RETURNING id
	`

	batch := &pgx.Batch{}
	for _, node := range nodes {
		batch.Queue(query, pgx.StrictNamedArgs{
			"kind":          node.Kind,
			"namespace":     node.Namespace,
			"name":          node.Name,
			"metadata":      node.Metadata,
			"first_seen_at": node.FirstSeenAt,
			"last_seen_at":  node.LastSeenAt,
		})
	}

	results := tx.SendBatch(ctx, batch)

	for i, node := range nodes {
		var id uuid.UUID
		if err := results.QueryRow().Scan(&id); err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for dependency node %d: %w", i, err)
		}
		node.ID = id
	}

	if err := results.Close(); err != nil {
		return fmt.Errorf("failed to close batch results: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func PruneOrphanDependencyNodes(ctx context.Context) error {
	query := `
		DELETE FROM dependency_nodes n
		WHERE NOT EXISTS (
			SELECT 1 FROM dependency_edges e
			WHERE e.from_node_id = n.id OR e.to_node_id = n.id
		)
	`
	_, err := Pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prune orphan dependency nodes: %w", err)
	}
	return nil
}
