package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func UpsertDependencyEdgesBatch(ctx context.Context, edges []*DependencyEdge) error {
	if len(edges) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO dependency_edges (
			from_node_id, to_node_id, protocol, port,
			via_service_namespace, via_service_name, source,
			connects, tx_bytes, rx_bytes, active_connections,
			first_seen_at, last_seen_at, evidence, attrs
		) VALUES (
			@from_node_id, @to_node_id, @protocol, @port,
			@via_service_namespace, @via_service_name, @source,
			@connects, @tx_bytes, @rx_bytes, @active_connections,
			@first_seen_at, @last_seen_at, @evidence, @attrs
		)
		ON CONFLICT (from_node_id, to_node_id, protocol, port) DO UPDATE SET
			via_service_namespace = EXCLUDED.via_service_namespace,
			via_service_name = EXCLUDED.via_service_name,
			source = EXCLUDED.source,
			connects = EXCLUDED.connects,
			tx_bytes = EXCLUDED.tx_bytes,
			rx_bytes = EXCLUDED.rx_bytes,
			active_connections = EXCLUDED.active_connections,
			last_seen_at = EXCLUDED.last_seen_at,
			evidence = EXCLUDED.evidence,
			attrs = EXCLUDED.attrs
	`

	batch := &pgx.Batch{}
	for _, edge := range edges {
		batch.Queue(query, pgx.StrictNamedArgs{
			"from_node_id":          edge.FromNodeID,
			"to_node_id":            edge.ToNodeID,
			"protocol":              edge.Protocol,
			"port":                  edge.Port,
			"via_service_namespace": edge.ViaServiceNamespace,
			"via_service_name":      edge.ViaServiceName,
			"source":                edge.Source,
			"connects":              edge.Connects,
			"tx_bytes":              edge.TxBytes,
			"rx_bytes":              edge.RxBytes,
			"active_connections":    edge.ActiveConnections,
			"first_seen_at":         edge.FirstSeenAt,
			"last_seen_at":          edge.LastSeenAt,
			"evidence":              edge.Evidence,
			"attrs":                 edge.Attrs,
		})
	}

	results := tx.SendBatch(ctx, batch)

	for i := range edges {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for dependency edge %d: %w", i, err)
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

const dependencyEdgeSelectColumns = `
	e.id, e.from_node_id, e.to_node_id, e.protocol, e.port,
	e.via_service_namespace, e.via_service_name, e.source,
	e.connects, e.tx_bytes, e.rx_bytes, e.active_connections,
	e.first_seen_at, e.last_seen_at, e.evidence, e.attrs, e.created_at, e.updated_at
`

func GetDependencyEdgesForNamespace(ctx context.Context, namespace string, since time.Time) ([]*DependencyEdge, error) {
	query := `
		SELECT ` + dependencyEdgeSelectColumns + `
		FROM dependency_edges e
		JOIN dependency_nodes f ON f.id = e.from_node_id
		JOIN dependency_nodes t ON t.id = e.to_node_id
		WHERE f.namespace = @namespace
		  AND e.last_seen_at >= @since
		UNION
		SELECT ` + dependencyEdgeSelectColumns + `
		FROM dependency_edges e
		JOIN dependency_nodes f ON f.id = e.from_node_id
		JOIN dependency_nodes t ON t.id = e.to_node_id
		WHERE t.namespace = @namespace
		  AND e.last_seen_at >= @since
	`

	rows, err := Pool.Query(ctx, query, pgx.StrictNamedArgs{
		"namespace": namespace,
		"since":     since,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency edges for namespace: %w", err)
	}
	defer rows.Close()

	return collectDependencyEdges(rows)
}

func GetDependencyEdgesByFromNode(ctx context.Context, nodeID uuid.UUID, since time.Time) ([]*DependencyEdge, error) {
	query := `
		SELECT ` + dependencyEdgeSelectColumns + `
		FROM dependency_edges e
		WHERE e.from_node_id = @node_id
		  AND e.last_seen_at >= @since
	`

	rows, err := Pool.Query(ctx, query, pgx.StrictNamedArgs{
		"node_id": nodeID,
		"since":   since,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency edges by from node: %w", err)
	}
	defer rows.Close()

	return collectDependencyEdges(rows)
}

func GetDependencyEdgesByToNode(ctx context.Context, nodeID uuid.UUID, since time.Time) ([]*DependencyEdge, error) {
	query := `
		SELECT ` + dependencyEdgeSelectColumns + `
		FROM dependency_edges e
		WHERE e.to_node_id = @node_id
		  AND e.last_seen_at >= @since
	`

	rows, err := Pool.Query(ctx, query, pgx.StrictNamedArgs{
		"node_id": nodeID,
		"since":   since,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency edges by to node: %w", err)
	}
	defer rows.Close()

	return collectDependencyEdges(rows)
}

func collectDependencyEdges(rows pgx.Rows) ([]*DependencyEdge, error) {
	edges := make([]*DependencyEdge, 0)
	for rows.Next() {
		var edge DependencyEdge
		if err := rows.Scan(
			&edge.ID, &edge.FromNodeID, &edge.ToNodeID, &edge.Protocol, &edge.Port,
			&edge.ViaServiceNamespace, &edge.ViaServiceName, &edge.Source,
			&edge.Connects, &edge.TxBytes, &edge.RxBytes, &edge.ActiveConnections,
			&edge.FirstSeenAt, &edge.LastSeenAt, &edge.Evidence, &edge.Attrs,
			&edge.CreatedAt, &edge.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan dependency edge: %w", err)
		}
		edges = append(edges, &edge)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate dependency edges: %w", err)
	}
	return edges, nil
}

func PruneStaleDependencyEdges(ctx context.Context, before time.Time) error {
	query := `DELETE FROM dependency_edges WHERE last_seen_at < @before`
	_, err := Pool.Exec(ctx, query,
		pgx.StrictNamedArgs{"before": before},
	)
	if err != nil {
		return fmt.Errorf("failed to prune stale dependency edges: %w", err)
	}
	return nil
}
