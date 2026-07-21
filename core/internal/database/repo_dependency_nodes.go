package database

import (
	"context"
	"fmt"

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

	for i := range nodes {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for dependency node %d: %w", i, err)
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
