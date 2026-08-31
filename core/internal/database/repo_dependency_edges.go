package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// DependencyNodeKey is a natural-key node identity for graph writes.
type DependencyNodeKey struct {
	Kind      string
	Namespace string
	Name      string
}

// DependencyEdgeUpsert is a discovery edge keyed by node identity.
type DependencyEdgeUpsert struct {
	From                DependencyNodeKey
	To                  DependencyNodeKey
	Protocol            string
	Port                int
	ViaServiceNamespace *string
	ViaServiceName      *string
	ViaServicePort      *int
	Source              string
	Connects            float64
	TxBytes             float64
	RxBytes             float64
	ActiveConnections   float64
	FirstSeenAt         time.Time
	LastSeenAt          time.Time
	Evidence            json.RawMessage
	Attrs               json.RawMessage
}

func UpsertDependencyGraph(ctx context.Context, edges []*DependencyEdgeUpsert) error {
	if len(edges) == 0 {
		return nil
	}

	nodesByKey := make(map[DependencyNodeKey]*DependencyNode)
	for _, edge := range edges {
		for _, key := range []DependencyNodeKey{edge.From, edge.To} {
			if _, ok := nodesByKey[key]; ok {
				continue
			}
			nodesByKey[key] = &DependencyNode{
				Kind:        key.Kind,
				Namespace:   key.Namespace,
				Name:        key.Name,
				FirstSeenAt: edge.FirstSeenAt,
				LastSeenAt:  edge.LastSeenAt,
			}
		}
	}

	nodes := make([]*DependencyNode, 0, len(nodesByKey))
	for _, node := range nodesByKey {
		nodes = append(nodes, node)
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	batch := &pgx.Batch{}
	for _, node := range nodes {
		queueDependencyNodeUpsert(batch, node)
	}
	for _, edge := range edges {
		queueDependencyEdgeUpsert(batch, edge)
	}

	results := tx.SendBatch(ctx, batch)
	for range nodes {
		if _, err := results.Exec(); err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for dependency graph: %w", err)
		}
	}
	for _, edge := range edges {
		tag, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for dependency graph: %w", err)
		}
		if tag.RowsAffected() != 1 {
			results.Close()
			return fmt.Errorf(
				"failed to upsert dependency edge: missing node %s/%s/%s or %s/%s/%s",
				edge.From.Kind, edge.From.Namespace, edge.From.Name,
				edge.To.Kind, edge.To.Namespace, edge.To.Name,
			)
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

func queueDependencyEdgeUpsert(batch *pgx.Batch, edge *DependencyEdgeUpsert) {
	batch.Queue(`
		INSERT INTO dependency_edges (
			from_node_id, to_node_id, protocol, port,
			via_service_namespace, via_service_name, via_service_port, source,
			connects, tx_bytes, rx_bytes, active_connections,
			first_seen_at, last_seen_at, evidence, attrs
		)
		SELECT f.id, t.id, @protocol, @port,
			@via_service_namespace, @via_service_name, @via_service_port, @source,
			@connects, @tx_bytes, @rx_bytes, @active_connections,
			@first_seen_at, @last_seen_at, @evidence, @attrs
		FROM dependency_nodes f
		JOIN dependency_nodes t
			ON t.kind = @to_kind AND t.namespace = @to_namespace AND t.name = @to_name
		WHERE f.kind = @from_kind AND f.namespace = @from_namespace AND f.name = @from_name
		ON CONFLICT (from_node_id, to_node_id, protocol, port) DO UPDATE SET
			via_service_namespace = EXCLUDED.via_service_namespace,
			via_service_name = EXCLUDED.via_service_name,
			via_service_port = EXCLUDED.via_service_port,
			source = EXCLUDED.source,
			connects = EXCLUDED.connects,
			tx_bytes = EXCLUDED.tx_bytes,
			rx_bytes = EXCLUDED.rx_bytes,
			active_connections = EXCLUDED.active_connections,
			last_seen_at = EXCLUDED.last_seen_at,
			evidence = EXCLUDED.evidence,
			attrs = EXCLUDED.attrs
	`, pgx.StrictNamedArgs{
		"from_kind":             edge.From.Kind,
		"from_namespace":        edge.From.Namespace,
		"from_name":             edge.From.Name,
		"to_kind":               edge.To.Kind,
		"to_namespace":          edge.To.Namespace,
		"to_name":               edge.To.Name,
		"protocol":              edge.Protocol,
		"port":                  edge.Port,
		"via_service_namespace": edge.ViaServiceNamespace,
		"via_service_name":      edge.ViaServiceName,
		"via_service_port":      edge.ViaServicePort,
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

const dependencyEdgeSelectColumns = `
	e.id, e.from_node_id, e.to_node_id, e.protocol, e.port,
	e.via_service_namespace, e.via_service_name, e.via_service_port, e.source,
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
			&edge.ViaServiceNamespace, &edge.ViaServiceName, &edge.ViaServicePort, &edge.Source,
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

func pruneExpiredDependencyEdges(ctx context.Context, since time.Time) error {
	_, err := Pool.Exec(ctx,
		`DELETE FROM dependency_edges WHERE last_seen_at < @last_seen_at`,
		pgx.StrictNamedArgs{"last_seen_at": since},
	)
	if err != nil {
		return fmt.Errorf("failed to prune expired dependency edges: %w", err)
	}
	return nil
}

// PruneExpiredDependencyGraph deletes aged edges then removes nodes with no edges.
func PruneExpiredDependencyGraph(ctx context.Context, since time.Time) error {
	var errs []error
	if err := pruneExpiredDependencyEdges(ctx, since); err != nil {
		errs = append(errs, err)
	}
	if err := pruneOrphanDependencyNodes(ctx); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
