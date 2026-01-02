package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/internal/logger"
)

// Upserts service metadata into the database
func UpsertService(ctx context.Context, service *Service) error {
	log := logger.WithComponent("database")
	log.Debug().
		Str("name", service.Name).
		Str("namespace", service.Namespace).
		Msg("Upserting service")

	query := `
		INSERT INTO services (
			name, namespace, uid, type, cluster_ip, ports, selector,
			labels, annotations, created_at, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
			namespace = EXCLUDED.namespace,
			type = EXCLUDED.type,
			cluster_ip = EXCLUDED.cluster_ip,
			ports = EXCLUDED.ports,
			selector = EXCLUDED.selector,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			synced_at = EXCLUDED.synced_at
	`

	_, err := Pool.Exec(ctx, query,
		service.Name, service.Namespace, service.UID, service.Type, service.ClusterIP,
		service.Ports, service.Selector, service.Labels, service.Annotations,
		service.CreatedAt, service.SyncedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert service: %w", err)
	}

	log.Debug().
		Str("name", service.Name).
		Str("namespace", service.Namespace).
		Msg("Service upserted successfully")

	return nil
}

// Upserts multiple services in a batch transaction
func UpsertServicesBatch(ctx context.Context, services []*Service) error {
	if len(services) == 0 {
		return nil
	}

	log := logger.WithComponent("database")
	log.Debug().Int("count", len(services)).Msg("Upserting services batch")

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO services (
			name, namespace, uid, type, cluster_ip, ports, selector,
			labels, annotations, created_at, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
			namespace = EXCLUDED.namespace,
			type = EXCLUDED.type,
			cluster_ip = EXCLUDED.cluster_ip,
			ports = EXCLUDED.ports,
			selector = EXCLUDED.selector,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, service := range services {
		batch.Queue(query,
			service.Name, service.Namespace, service.UID, service.Type, service.ClusterIP,
			service.Ports, service.Selector, service.Labels, service.Annotations,
			service.CreatedAt, service.SyncedAt,
		)
	}

	results := tx.SendBatch(ctx, batch)

	for i := range services {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for service %d: %w", i, err)
		}
	}

	if err := results.Close(); err != nil {
		return fmt.Errorf("failed to close batch results: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug().Int("count", len(services)).Msg("Services upserted successfully")
	return nil
}

// Deletes services that haven't been synced since the specified time
func DeleteServicesNotSyncedSince(ctx context.Context, since time.Time) error {
	log := logger.WithComponent("database")

	query := `DELETE FROM services WHERE synced_at < $1`
	result, err := Pool.Exec(ctx, query, since)
	if err != nil {
		return fmt.Errorf("failed to delete stale services: %w", err)
	}

	deleted := result.RowsAffected()
	if deleted > 0 {
		log.Debug().Int64("count", deleted).Msg("Stale services deleted")
	}

	return nil
}
