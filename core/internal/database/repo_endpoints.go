package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// UpsertEndpointSlicesBatch inserts or updates EndpointSlices by Kubernetes UID inside one transaction.
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
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
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
		batch.Queue(query,
			es.Name, es.Namespace, es.UID, es.AddressType,
			es.OwnerKind, es.OwnerName, es.Endpoints, es.Ports,
			es.Labels, es.Annotations, es.CreatedAt, es.SyncedAt,
		)
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

// PruneEndpointSlices deletes EndpointSlices whose UID is not in uids in the namespace.
// Empty uids deletes all EndpointSlices in that namespace.
func PruneEndpointSlices(ctx context.Context, namespace string, uids []string) error {
	if len(uids) == 0 {
		_, err := Pool.Exec(ctx, `DELETE FROM endpoint_slices WHERE namespace = $1`, namespace)
		return err
	}
	_, err := Pool.Exec(ctx, `DELETE FROM endpoint_slices WHERE namespace = $1 AND NOT (uid = ANY($2::text[]))`, namespace, uids)
	return err
}
