package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/internal/logger"
)

// Upserts endpoint metadata into the database
func UpsertEndpoint(ctx context.Context, endpoint *Endpoint) error {
	log := logger.WithComponent("database")
	log.Debug().
		Str("name", endpoint.Name).
		Str("namespace", endpoint.Namespace).
		Msg("Upserting endpoint")

	query := `
		INSERT INTO endpoints (
			name, namespace, uid, addresses, ports,
			labels, annotations, created_at, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
			namespace = EXCLUDED.namespace,
			addresses = EXCLUDED.addresses,
			ports = EXCLUDED.ports,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			synced_at = EXCLUDED.synced_at
	`

	_, err := Pool.Exec(ctx, query,
		endpoint.Name, endpoint.Namespace, endpoint.UID,
		endpoint.Addresses, endpoint.Ports,
		endpoint.Labels, endpoint.Annotations, endpoint.CreatedAt, endpoint.SyncedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert endpoint: %w", err)
	}

	log.Debug().
		Str("name", endpoint.Name).
		Str("namespace", endpoint.Namespace).
		Msg("Endpoint upserted successfully")

	return nil
}

// Upserts multiple endpoints in a batch transaction
func UpsertEndpointsBatch(ctx context.Context, endpoints []*Endpoint) error {
	if len(endpoints) == 0 {
		return nil
	}

	log := logger.WithComponent("database")
	log.Debug().Int("count", len(endpoints)).Msg("Upserting endpoints batch")

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO endpoints (
			name, namespace, uid, addresses, ports,
			labels, annotations, created_at, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
			namespace = EXCLUDED.namespace,
			addresses = EXCLUDED.addresses,
			ports = EXCLUDED.ports,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, endpoint := range endpoints {
		batch.Queue(query,
			endpoint.Name, endpoint.Namespace, endpoint.UID,
			endpoint.Addresses, endpoint.Ports,
			endpoint.Labels, endpoint.Annotations, endpoint.CreatedAt, endpoint.SyncedAt,
		)
	}

	results := tx.SendBatch(ctx, batch)

	for i := range endpoints {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for endpoint %d: %w", i, err)
		}
	}

	if err := results.Close(); err != nil {
		return fmt.Errorf("failed to close batch results: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug().Int("count", len(endpoints)).Msg("Endpoints upserted successfully")
	return nil
}

// Deletes endpoints that haven't been synced since the specified time
func DeleteEndpointsNotSyncedSince(ctx context.Context, since time.Time) error {
	log := logger.WithComponent("database")

	query := `DELETE FROM endpoints WHERE synced_at < $1`
	result, err := Pool.Exec(ctx, query, since)
	if err != nil {
		return fmt.Errorf("failed to delete stale endpoints: %w", err)
	}

	deleted := result.RowsAffected()
	if deleted > 0 {
		log.Debug().Int64("count", deleted).Msg("Stale endpoints deleted")
	}

	return nil
}
