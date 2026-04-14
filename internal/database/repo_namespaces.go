package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/internal/logger"
)

// UpsertNamespacesBatch inserts or updates namespaces by Kubernetes UID inside one transaction.
func UpsertNamespacesBatch(ctx context.Context, namespaces []*Namespace) error {
	if len(namespaces) == 0 {
		return nil
	}

	log := logger.WithComponent("database")
	log.Debug().Int("count", len(namespaces)).Msg("Upserting namespaces batch")

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO namespaces (name, uid, phase, labels, annotations, created_at, synced_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
			phase = EXCLUDED.phase,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, namespace := range namespaces {
		batch.Queue(query,
			namespace.Name, namespace.UID, namespace.Phase,
			namespace.Labels, namespace.Annotations, namespace.CreatedAt, namespace.SyncedAt,
		)
	}

	results := tx.SendBatch(ctx, batch)

	for i := range namespaces {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for namespace %d: %w", i, err)
		}
	}

	if err := results.Close(); err != nil {
		return fmt.Errorf("failed to close batch results: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug().Int("count", len(namespaces)).Msg("Namespaces upserted successfully")
	return nil
}

// GetNamespaces returns all the namespaces, ordered by name.
func GetNamespaces(ctx context.Context) ([]*Namespace, error) {
	query := `SELECT id, name, uid, phase, labels, annotations, created_at, updated_at, synced_at FROM namespaces ORDER BY name ASC`

	rows, err := Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query namespaces: %w", err)
	}
	defer rows.Close()

	namespaces := make([]*Namespace, 0)
	for rows.Next() {
		var ns Namespace
		if err := rows.Scan(
			&ns.ID, &ns.Name, &ns.UID, &ns.Phase,
			&ns.Labels, &ns.Annotations,
			&ns.CreatedAt, &ns.UpdatedAt, &ns.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan namespace: %w", err)
		}
		namespaces = append(namespaces, &ns)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate namespaces: %w", err)
	}

	return namespaces, nil
}

// GetNamespaceByName returns a namespace by its name.
func GetNamespaceByName(ctx context.Context, name string) (*Namespace, error) {
	query := `SELECT id, name, uid, phase, labels, annotations, created_at, updated_at, synced_at FROM namespaces WHERE name = $1 LIMIT 1`

	var ns Namespace
	err := Pool.QueryRow(ctx, query, name).Scan(
		&ns.ID, &ns.Name, &ns.UID, &ns.Phase,
		&ns.Labels, &ns.Annotations,
		&ns.CreatedAt, &ns.UpdatedAt, &ns.SyncedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("namespace %s not found", name)
		}
		return nil, fmt.Errorf("failed to query namespace by name: %w", err)
	}

	return &ns, nil
}

// PruneNamespaces deletes namespaces whose UID is not in uids. An empty uids slice clears the whole table,
// which matches sync when the API returns zero namespaces.
func PruneNamespaces(ctx context.Context, uids []string) error {
	if len(uids) == 0 {
		_, err := Pool.Exec(ctx, `DELETE FROM namespaces`)
		return err
	}
	_, err := Pool.Exec(ctx, `DELETE FROM namespaces WHERE NOT (uid = ANY($1::text[]))`, uids)
	return err
}
