package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/apperrors"
)

func queueDependencyNodeUpsert(batch *pgx.Batch, node *DependencyNode) {
	batch.Queue(`
		INSERT INTO dependency_nodes (
			kind, namespace, name, metadata, first_seen_at, last_seen_at
		) VALUES (
			@kind, @namespace, @name, @metadata, @first_seen_at, @last_seen_at
		)
		ON CONFLICT (kind, namespace, name) DO UPDATE SET
			metadata = EXCLUDED.metadata,
			last_seen_at = EXCLUDED.last_seen_at
	`, pgx.StrictNamedArgs{
		"kind":          node.Kind,
		"namespace":     node.Namespace,
		"name":          node.Name,
		"metadata":      node.Metadata,
		"first_seen_at": node.FirstSeenAt,
		"last_seen_at":  node.LastSeenAt,
	})
}

func GetDependencyNodeByKey(ctx context.Context, kind, namespace, name string) (*DependencyNode, error) {
	query := `
		SELECT id, kind, namespace, name, metadata, 
			   first_seen_at, last_seen_at, created_at, updated_at
		FROM dependency_nodes
		WHERE kind = @kind AND namespace = @namespace AND name = @name
		LIMIT 1
	`

	var node DependencyNode
	err := Pool.QueryRow(ctx, query, pgx.StrictNamedArgs{
		"kind":      kind,
		"namespace": namespace,
		"name":      name,
	}).Scan(
		&node.ID, &node.Kind, &node.Namespace, &node.Name, &node.Metadata,
		&node.FirstSeenAt, &node.LastSeenAt, &node.CreatedAt, &node.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("dependency_node", fmt.Sprintf("%s/%s/%s", kind, namespace, name))
		}
		return nil, fmt.Errorf("failed to query dependency node by key: %w", err)
	}

	return &node, nil
}

func GetDependencyNodesByIDs(ctx context.Context, ids []uuid.UUID) ([]*DependencyNode, error) {
	if len(ids) == 0 {
		return []*DependencyNode{}, nil
	}

	query := `
		SELECT id, kind, namespace, name, metadata, 
			   first_seen_at, last_seen_at, created_at, updated_at
		FROM dependency_nodes
		WHERE id = ANY(@ids::uuid[])
	`

	rows, err := Pool.Query(ctx, query,
		pgx.StrictNamedArgs{"ids": ids},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query dependency nodes by ids: %w", err)
	}
	defer rows.Close()

	nodes := make([]*DependencyNode, 0, len(ids))
	for rows.Next() {
		var node DependencyNode
		if err := rows.Scan(
			&node.ID, &node.Kind, &node.Namespace, &node.Name, &node.Metadata,
			&node.FirstSeenAt, &node.LastSeenAt, &node.CreatedAt, &node.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan dependency node: %w", err)
		}
		nodes = append(nodes, &node)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate dependency nodes: %w", err)
	}

	return nodes, nil
}

func pruneOrphanDependencyNodes(ctx context.Context) error {
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
