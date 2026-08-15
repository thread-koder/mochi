package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/apperrors"
)

func UpsertNamespacesBatch(ctx context.Context, namespaces []*Namespace) error {
	if len(namespaces) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO namespaces (
			name, uid, phase, labels, annotations, created_at, synced_at
		) VALUES (
			@name, @uid, @phase, @labels, @annotations, @created_at, @synced_at
		)
		ON CONFLICT (name) DO UPDATE SET
			uid = EXCLUDED.uid,
			phase = EXCLUDED.phase,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, namespace := range namespaces {
		batch.Queue(query, pgx.StrictNamedArgs{
			"name":        namespace.Name,
			"uid":         namespace.UID,
			"phase":       namespace.Phase,
			"labels":      namespace.Labels,
			"annotations": namespace.Annotations,
			"created_at":  namespace.CreatedAt,
			"synced_at":   namespace.SyncedAt,
		})
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

	return nil
}

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

func GetNamespaceByName(ctx context.Context, name string) (*Namespace, error) {
	query := `SELECT id, name, uid, phase, labels, annotations, created_at, updated_at, synced_at FROM namespaces WHERE name = @name LIMIT 1`

	var ns Namespace
	err := Pool.QueryRow(ctx, query,
		pgx.StrictNamedArgs{"name": name},
	).Scan(
		&ns.ID, &ns.Name, &ns.UID, &ns.Phase,
		&ns.Labels, &ns.Annotations,
		&ns.CreatedAt, &ns.UpdatedAt, &ns.SyncedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("namespace", name)
		}
		return nil, fmt.Errorf("failed to query namespace by name: %w", err)
	}

	return &ns, nil
}

// PruneNamespaces deletes namespaces not listed in uids.
// Empty uids deletes all namespaces.
func PruneNamespaces(ctx context.Context, uids []string) error {
	var err error
	if len(uids) == 0 {
		_, err = Pool.Exec(ctx, `DELETE FROM namespaces`)
	} else {
		_, err = Pool.Exec(ctx,
			`DELETE FROM namespaces WHERE NOT (uid = ANY(@uids::text[]))`,
			pgx.StrictNamedArgs{"uids": uids},
		)
	}
	if err != nil {
		return fmt.Errorf("failed to prune namespaces: %w", err)
	}
	return nil
}
