package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// UpsertServicesBatch inserts or updates Services by Kubernetes UID inside one transaction.
func UpsertServicesBatch(ctx context.Context, services []*Service) error {
	if len(services) == 0 {
		return nil
	}

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

	return nil
}

// PruneServices deletes services whose UID is not in uids in the namespace.
// Empty uids deletes all services in that namespace.
func PruneServices(ctx context.Context, namespace string, uids []string) error {
	if len(uids) == 0 {
		_, err := Pool.Exec(ctx, `DELETE FROM services WHERE namespace = $1`, namespace)
		return err
	}
	_, err := Pool.Exec(ctx, `DELETE FROM services WHERE namespace = $1 AND NOT (uid = ANY($2::text[]))`, namespace, uids)
	return err
}
