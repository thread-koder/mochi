package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func UpsertEndpointSlicesBatch(ctx context.Context, endpointSlices []*EndpointSlice) error {
	if len(endpointSlices) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO endpoint_slices (
			name, namespace, uid, address_type,
			owner_kind, owner_name, endpoints, ports, labels, annotations, created_at, synced_at
		) VALUES (
			@name, @namespace, @uid, @address_type,
			@owner_kind, @owner_name, @endpoints, @ports, @labels, @annotations, @created_at, @synced_at
		)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
			namespace = EXCLUDED.namespace,
			address_type = EXCLUDED.address_type,
			owner_kind = EXCLUDED.owner_kind,
			owner_name = EXCLUDED.owner_name,
			endpoints = EXCLUDED.endpoints,
			ports = EXCLUDED.ports,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, es := range endpointSlices {
		batch.Queue(query, pgx.StrictNamedArgs{
			"name":         es.Name,
			"namespace":    es.Namespace,
			"uid":          es.UID,
			"address_type": es.AddressType,
			"owner_kind":   es.OwnerKind,
			"owner_name":   es.OwnerName,
			"endpoints":    es.Endpoints,
			"ports":        es.Ports,
			"labels":       es.Labels,
			"annotations":  es.Annotations,
			"created_at":   es.CreatedAt,
			"synced_at":    es.SyncedAt,
		})
	}

	results := tx.SendBatch(ctx, batch)

	for i := range endpointSlices {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for endpoint slice %d: %w", i, err)
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

// PruneEndpointSlices deletes EndpointSlices not listed in uids.
// Empty uids deletes every EndpointSlice in the namespace.
func PruneEndpointSlices(ctx context.Context, namespace string, uids []string) error {
	if len(uids) == 0 {
		_, err := Pool.Exec(ctx,
			`DELETE FROM endpoint_slices WHERE namespace = @namespace`,
			pgx.StrictNamedArgs{"namespace": namespace},
		)
		return err
	}
	_, err := Pool.Exec(ctx,
		`DELETE FROM endpoint_slices WHERE namespace = @namespace AND NOT (uid = ANY(@uids::text[]))`,
		pgx.StrictNamedArgs{
			"namespace": namespace,
			"uids":      uids,
		},
	)
	return err
}
