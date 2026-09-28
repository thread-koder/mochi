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
	FirstSeenAt         time.Time
	LastSeenAt          time.Time
	Evidence            json.RawMessage
}

// DependencyEdgeWindow is an identity edge plus hour-window volume for analyze.
type DependencyEdgeWindow struct {
	ID                  uuid.UUID
	FromNodeID          uuid.UUID
	ToNodeID            uuid.UUID
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
}

func UpsertDependencyEdgesWithHours(ctx context.Context, edges []*DependencyEdgeUpsert, hours []*DependencyEdgeHour) error {
	if len(edges) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	edgeIDs, err := upsertDependencyEdgesTx(ctx, tx, edges)
	if err != nil {
		return err
	}

	hourBatch := &pgx.Batch{}
	for i, hour := range hours {
		hour.EdgeID = edgeIDs[i]
		queueDependencyEdgeHourReplace(hourBatch, hour)
	}
	hourResults := tx.SendBatch(ctx, hourBatch)
	for range hours {
		if _, err := hourResults.Exec(); err != nil {
			hourResults.Close()
			return fmt.Errorf("failed to replace dependency edge hour: %w", err)
		}
	}
	if err := hourResults.Close(); err != nil {
		return fmt.Errorf("failed to close batch results: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func upsertDependencyEdgesTx(ctx context.Context, tx pgx.Tx, edges []*DependencyEdgeUpsert) ([]uuid.UUID, error) {
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

	nodeBatch := &pgx.Batch{}
	for _, node := range nodes {
		queueDependencyNodeUpsert(nodeBatch, node)
	}
	nodeResults := tx.SendBatch(ctx, nodeBatch)
	for range nodes {
		if _, err := nodeResults.Exec(); err != nil {
			nodeResults.Close()
			return nil, fmt.Errorf("failed to execute batch upsert for dependency graph: %w", err)
		}
	}
	if err := nodeResults.Close(); err != nil {
		return nil, fmt.Errorf("failed to close batch results: %w", err)
	}

	edgeBatch := &pgx.Batch{}
	for _, edge := range edges {
		queueDependencyEdgeUpsert(edgeBatch, edge)
	}
	edgeResults := tx.SendBatch(ctx, edgeBatch)
	edgeIDs := make([]uuid.UUID, len(edges))
	for i, edge := range edges {
		if err := edgeResults.QueryRow().Scan(&edgeIDs[i]); err != nil {
			edgeResults.Close()
			return nil, fmt.Errorf(
				"failed to upsert dependency edge: missing node %s/%s/%s or %s/%s/%s: %w",
				edge.From.Kind, edge.From.Namespace, edge.From.Name,
				edge.To.Kind, edge.To.Namespace, edge.To.Name,
				err,
			)
		}
	}
	if err := edgeResults.Close(); err != nil {
		return nil, fmt.Errorf("failed to close batch results: %w", err)
	}
	return edgeIDs, nil
}

func queueDependencyEdgeUpsert(batch *pgx.Batch, edge *DependencyEdgeUpsert) {
	batch.Queue(`
		INSERT INTO dependency_edges (
			from_node_id, to_node_id, protocol, port,
			via_service_namespace, via_service_name, via_service_port, source,
			first_seen_at, last_seen_at, evidence
		)
		SELECT f.id, t.id, @protocol, @port,
			@via_service_namespace, @via_service_name, @via_service_port, @source,
			@first_seen_at, @last_seen_at, @evidence
		FROM dependency_nodes f
		JOIN dependency_nodes t
			ON t.kind = @to_kind AND t.namespace = @to_namespace AND t.name = @to_name
		WHERE f.kind = @from_kind AND f.namespace = @from_namespace AND f.name = @from_name
		ON CONFLICT (from_node_id, to_node_id, protocol, port) DO UPDATE SET
			via_service_namespace = EXCLUDED.via_service_namespace,
			via_service_name = EXCLUDED.via_service_name,
			via_service_port = EXCLUDED.via_service_port,
			source = EXCLUDED.source,
			last_seen_at = EXCLUDED.last_seen_at,
			evidence = EXCLUDED.evidence
		RETURNING id
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
		"first_seen_at":         edge.FirstSeenAt,
		"last_seen_at":          edge.LastSeenAt,
		"evidence":              edge.Evidence,
	})
}

const dependencyEdgeSelectColumns = `
	e.id, e.from_node_id, e.to_node_id, e.protocol, e.port,
	e.via_service_namespace, e.via_service_name, e.via_service_port, e.source,
	COALESCE(hour_totals.connects, 0), COALESCE(hour_totals.tx_bytes, 0), COALESCE(hour_totals.rx_bytes, 0),
	COALESCE(latest_active.active_connections, 0),
	e.first_seen_at, e.last_seen_at
`

const dependencyEdgeHourJoins = `
	LEFT JOIN (
		SELECT edge_id,
			SUM(connects) AS connects,
			SUM(tx_bytes) AS tx_bytes,
			SUM(rx_bytes) AS rx_bytes
		FROM dependency_edge_hours
		WHERE hour_start >= @hour_since
		GROUP BY edge_id
	) hour_totals ON hour_totals.edge_id = e.id
	LEFT JOIN LATERAL (
		SELECT active_connections
		FROM dependency_edge_hours
		WHERE edge_id = e.id
		  AND hour_start >= @hour_since
		ORDER BY hour_start DESC
		LIMIT 1
	) latest_active ON true
`

func GetDependencyEdgesForNamespace(ctx context.Context, namespace string, since time.Time) ([]*DependencyEdgeWindow, error) {
	hourSince := since.UTC().Truncate(time.Hour)
	query := `
		SELECT ` + dependencyEdgeSelectColumns + `
		FROM dependency_edges e
		JOIN dependency_nodes f ON f.id = e.from_node_id
		JOIN dependency_nodes t ON t.id = e.to_node_id
		` + dependencyEdgeHourJoins + `
		WHERE e.last_seen_at >= @since
		  AND (f.namespace = @namespace OR t.namespace = @namespace)
	`

	rows, err := Pool.Query(ctx, query, pgx.StrictNamedArgs{
		"namespace":  namespace,
		"since":      since,
		"hour_since": hourSince,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency edges for namespace: %w", err)
	}
	defer rows.Close()

	return collectDependencyEdgeWindows(rows)
}

func GetDependencyEdgesByFromNode(ctx context.Context, nodeID uuid.UUID, since time.Time) ([]*DependencyEdgeWindow, error) {
	hourSince := since.UTC().Truncate(time.Hour)
	query := `
		SELECT ` + dependencyEdgeSelectColumns + `
		FROM dependency_edges e
		` + dependencyEdgeHourJoins + `
		WHERE e.from_node_id = @node_id
		  AND e.last_seen_at >= @since
	`

	rows, err := Pool.Query(ctx, query, pgx.StrictNamedArgs{
		"node_id":    nodeID,
		"since":      since,
		"hour_since": hourSince,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency edges by from node: %w", err)
	}
	defer rows.Close()

	return collectDependencyEdgeWindows(rows)
}

func GetDependencyEdgesByToNode(ctx context.Context, nodeID uuid.UUID, since time.Time) ([]*DependencyEdgeWindow, error) {
	hourSince := since.UTC().Truncate(time.Hour)
	query := `
		SELECT ` + dependencyEdgeSelectColumns + `
		FROM dependency_edges e
		` + dependencyEdgeHourJoins + `
		WHERE e.to_node_id = @node_id
		  AND e.last_seen_at >= @since
	`

	rows, err := Pool.Query(ctx, query, pgx.StrictNamedArgs{
		"node_id":    nodeID,
		"since":      since,
		"hour_since": hourSince,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency edges by to node: %w", err)
	}
	defer rows.Close()

	return collectDependencyEdgeWindows(rows)
}

func collectDependencyEdgeWindows(rows pgx.Rows) ([]*DependencyEdgeWindow, error) {
	edges := make([]*DependencyEdgeWindow, 0)
	for rows.Next() {
		var edge DependencyEdgeWindow
		if err := rows.Scan(
			&edge.ID, &edge.FromNodeID, &edge.ToNodeID, &edge.Protocol, &edge.Port,
			&edge.ViaServiceNamespace, &edge.ViaServiceName, &edge.ViaServicePort, &edge.Source,
			&edge.Connects, &edge.TxBytes, &edge.RxBytes, &edge.ActiveConnections,
			&edge.FirstSeenAt, &edge.LastSeenAt,
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

func PruneExpiredDependencyGraph(ctx context.Context, since time.Time) error {
	var errs []error
	if err := pruneExpiredDependencyEdgeHours(ctx, since); err != nil {
		errs = append(errs, err)
	}
	if err := pruneExpiredDependencyEdges(ctx, since); err != nil {
		errs = append(errs, err)
	}
	if err := pruneOrphanDependencyNodes(ctx); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
