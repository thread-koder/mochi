package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/internal/logger"
)

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

// Removes endpoints in the namespace whose uid is not in the list.
func PruneEndpoints(ctx context.Context, namespace string, uids []string) error {
	if len(uids) == 0 {
		_, err := Pool.Exec(ctx, `DELETE FROM endpoints WHERE namespace = $1`, namespace)
		return err
	}
	_, err := Pool.Exec(ctx, `DELETE FROM endpoints WHERE namespace = $1 AND NOT (uid = ANY($2::text[]))`, namespace, uids)
	return err
}
