package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DependencyEdgeHour struct {
	EdgeID            uuid.UUID
	HourStart         time.Time
	Connects          float64
	TxBytes           float64
	RxBytes           float64
	ActiveConnections float64
}

type DependencyEdgeIdentity struct {
	From     DependencyNodeKey
	To       DependencyNodeKey
	Protocol string
	Port     int
}

// PreviousHourFinalized is true when the closed hour was already written this UTC hour
// (any row for prevHourStart with updated_at >= currentHourStart).
func PreviousHourFinalized(ctx context.Context, prevHourStart, currentHourStart time.Time) (bool, error) {
	var exists bool
	err := Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM dependency_edge_hours
			WHERE hour_start = @prev_hour_start
			  AND updated_at >= @current_hour_start
		)
	`, pgx.StrictNamedArgs{
		"prev_hour_start":    prevHourStart,
		"current_hour_start": currentHourStart,
	}).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check previous-hour finalization: %w", err)
	}
	return exists, nil
}

func GetDependencyEdgeIDs(ctx context.Context, keys []DependencyEdgeIdentity) (map[DependencyEdgeIdentity]uuid.UUID, error) {
	ids := make(map[DependencyEdgeIdentity]uuid.UUID, len(keys))
	if len(keys) == 0 {
		return ids, nil
	}

	batch := &pgx.Batch{}
	for _, key := range keys {
		batch.Queue(`
			SELECT e.id
			FROM dependency_edges e
			JOIN dependency_nodes f ON f.id = e.from_node_id
			JOIN dependency_nodes t ON t.id = e.to_node_id
			WHERE f.kind = @from_kind AND f.namespace = @from_namespace AND f.name = @from_name
			  AND t.kind = @to_kind AND t.namespace = @to_namespace AND t.name = @to_name
			  AND e.protocol = @protocol AND e.port = @port
		`, pgx.StrictNamedArgs{
			"from_kind":      key.From.Kind,
			"from_namespace": key.From.Namespace,
			"from_name":      key.From.Name,
			"to_kind":        key.To.Kind,
			"to_namespace":   key.To.Namespace,
			"to_name":        key.To.Name,
			"protocol":       key.Protocol,
			"port":           key.Port,
		})
	}

	results := Pool.SendBatch(ctx, batch)
	defer results.Close()

	for _, key := range keys {
		var edgeID uuid.UUID
		err := results.QueryRow().Scan(&edgeID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, fmt.Errorf("failed to get dependency edge id: %w", err)
		}
		ids[key] = edgeID
	}
	return ids, nil
}

// ReplaceDependencyEdgeHours writes the hour's counters as absolute values (never ADD).
func ReplaceDependencyEdgeHours(ctx context.Context, hours []*DependencyEdgeHour) error {
	if len(hours) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	batch := &pgx.Batch{}
	for _, hour := range hours {
		queueDependencyEdgeHourReplace(batch, hour)
	}
	results := tx.SendBatch(ctx, batch)
	for range hours {
		if _, err := results.Exec(); err != nil {
			results.Close()
			return fmt.Errorf("failed to replace dependency edge hour: %w", err)
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

func queueDependencyEdgeHourReplace(batch *pgx.Batch, hour *DependencyEdgeHour) {
	batch.Queue(`
		INSERT INTO dependency_edge_hours (
			edge_id, hour_start, connects, tx_bytes, rx_bytes, active_connections
		) VALUES (
			@edge_id, @hour_start, @connects, @tx_bytes, @rx_bytes, @active_connections
		)
		ON CONFLICT (edge_id, hour_start) DO UPDATE SET
			connects = EXCLUDED.connects,
			tx_bytes = EXCLUDED.tx_bytes,
			rx_bytes = EXCLUDED.rx_bytes,
			active_connections = EXCLUDED.active_connections
	`, pgx.StrictNamedArgs{
		"edge_id":            hour.EdgeID,
		"hour_start":         hour.HourStart,
		"connects":           hour.Connects,
		"tx_bytes":           hour.TxBytes,
		"rx_bytes":           hour.RxBytes,
		"active_connections": hour.ActiveConnections,
	})
}

func pruneExpiredDependencyEdgeHours(ctx context.Context, since time.Time) error {
	_, err := Pool.Exec(ctx,
		`DELETE FROM dependency_edge_hours WHERE hour_start < @since`,
		pgx.StrictNamedArgs{"since": since.UTC().Truncate(time.Hour)},
	)
	if err != nil {
		return fmt.Errorf("failed to prune expired dependency edge hours: %w", err)
	}
	return nil
}
