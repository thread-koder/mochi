package database

import (
	"context"
	"fmt"

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
			via_service_namespace, via_service_name, source, confidence,
			connects, tx_bytes, rx_bytes, active_connections,
			first_seen_at, last_seen_at, evidence, attrs
		) VALUES (
			@from_node_id, @to_node_id, @protocol, @port,
			@via_service_namespace, @via_service_name, @source, @confidence,
			@connects, @tx_bytes, @rx_bytes, @active_connections,
			@first_seen_at, @last_seen_at, @evidence, @attrs
		)
		ON CONFLICT (from_node_id, to_node_id, protocol, port) DO UPDATE SET
			via_service_namespace = EXCLUDED.via_service_namespace,
			via_service_name = EXCLUDED.via_service_name,
			source = EXCLUDED.source,
			confidence = EXCLUDED.confidence,
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
			"confidence":            edge.Confidence,
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
