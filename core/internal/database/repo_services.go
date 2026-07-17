package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

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
		) VALUES (
			@name, @namespace, @uid, @type, @cluster_ip, @ports, @selector,
			@labels, @annotations, @created_at, @synced_at
		)
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
		batch.Queue(query, pgx.StrictNamedArgs{
			"name":        service.Name,
			"namespace":   service.Namespace,
			"uid":         service.UID,
			"type":        service.Type,
			"cluster_ip":  service.ClusterIP,
			"ports":       service.Ports,
			"selector":    service.Selector,
			"labels":      service.Labels,
			"annotations": service.Annotations,
			"created_at":  service.CreatedAt,
			"synced_at":   service.SyncedAt,
		})
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

// PruneServices deletes services not listed in uids.
// Empty uids deletes every service in the namespace.
func PruneServices(ctx context.Context, namespace string, uids []string) error {
	if len(uids) == 0 {
		_, err := Pool.Exec(ctx,
			`DELETE FROM services WHERE namespace = @namespace`,
			pgx.StrictNamedArgs{"namespace": namespace},
		)
		return err
	}
	_, err := Pool.Exec(ctx,
		`DELETE FROM services WHERE namespace = @namespace AND NOT (uid = ANY(@uids::text[]))`,
		pgx.StrictNamedArgs{
			"namespace": namespace,
			"uids":      uids,
		},
	)
	return err
}
