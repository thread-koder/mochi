package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func UpsertPodAttributionsBatch(ctx context.Context, attributions []*PodAttribution) error {
	if len(attributions) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO pod_attributions (
			uid, name, namespace, workload_kind, workload_name,
			phase, node, containers,
			first_seen_at, last_seen_at, finished_at
		) VALUES (
			@uid, @name, @namespace, @workload_kind, @workload_name,
			@phase, @node, @containers,
			@first_seen_at, @last_seen_at, @finished_at
		)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
			namespace = EXCLUDED.namespace,
			workload_kind = EXCLUDED.workload_kind,
			workload_name = EXCLUDED.workload_name,
			phase = EXCLUDED.phase,
			node = EXCLUDED.node,
			containers = EXCLUDED.containers,
			last_seen_at = EXCLUDED.last_seen_at,
			finished_at = COALESCE(pod_attributions.finished_at, EXCLUDED.finished_at)
	`

	batch := &pgx.Batch{}
	for _, attr := range attributions {
		batch.Queue(query, pgx.StrictNamedArgs{
			"uid":           attr.UID,
			"name":          attr.Name,
			"namespace":     attr.Namespace,
			"workload_kind": attr.WorkloadKind,
			"workload_name": attr.WorkloadName,
			"phase":         attr.Phase,
			"node":          attr.Node,
			"containers":    attr.Containers,
			"first_seen_at": attr.FirstSeenAt,
			"last_seen_at":  attr.LastSeenAt,
			"finished_at":   attr.FinishedAt,
		})
	}

	results := tx.SendBatch(ctx, batch)
	for i := range attributions {
		if _, err := results.Exec(); err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for pod attribution %d: %w", i, err)
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

// FinishUnlistedPodAttributions stamps finished_at for attributions whose UID is not in listedUIDs.
func FinishUnlistedPodAttributions(ctx context.Context, namespace string, listedUIDs []string, finishedAt time.Time) error {
	var err error
	if len(listedUIDs) == 0 {
		_, err = Pool.Exec(ctx, `
			UPDATE pod_attributions
			SET finished_at = @finished_at
			WHERE namespace = @namespace AND finished_at IS NULL
		`, pgx.StrictNamedArgs{
			"namespace":   namespace,
			"finished_at": finishedAt,
		})
	} else {
		_, err = Pool.Exec(ctx, `
			UPDATE pod_attributions
			SET finished_at = @finished_at
			WHERE namespace = @namespace
			  AND finished_at IS NULL
			  AND NOT (uid = ANY(@uids::text[]))
		`, pgx.StrictNamedArgs{
			"namespace":   namespace,
			"finished_at": finishedAt,
			"uids":        listedUIDs,
		})
	}
	if err != nil {
		return fmt.Errorf("failed to finish unlisted pod attributions: %w", err)
	}
	return nil
}

func PruneExpiredPodAttributions(ctx context.Context, since time.Time) error {
	_, err := Pool.Exec(ctx,
		`DELETE FROM pod_attributions WHERE last_seen_at < @last_seen_at`,
		pgx.StrictNamedArgs{"last_seen_at": since},
	)
	if err != nil {
		return fmt.Errorf("failed to prune expired pod attributions: %w", err)
	}
	return nil
}

func getPodAttributionByUID(ctx context.Context, uid string) (*PodAttribution, error) {
	query := `
		SELECT uid, name, namespace, workload_kind, workload_name,
			   phase, node, containers,
		       first_seen_at, last_seen_at, finished_at
		FROM pod_attributions
		WHERE uid = @uid
		LIMIT 1
	`

	var a PodAttribution
	err := Pool.QueryRow(ctx, query, pgx.StrictNamedArgs{"uid": uid}).Scan(
		&a.UID, &a.Name, &a.Namespace, &a.WorkloadKind, &a.WorkloadName,
		&a.Phase, &a.Node, &a.Containers,
		&a.FirstSeenAt, &a.LastSeenAt, &a.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query pod attribution by UID: %w", err)
	}
	return &a, nil
}

func getAttributedPods(ctx context.Context, workloadKind, workloadName, namespace string, since time.Time) ([]*Pod, error) {
	query := `
		SELECT uid, name, namespace, node, phase,
		       workload_kind, workload_name
		FROM pod_attributions
		WHERE namespace = @namespace
		  AND workload_kind = @workload_kind
		  AND workload_name = @workload_name
		  AND last_seen_at >= @since
		ORDER BY last_seen_at DESC
	`

	rows, err := Pool.Query(ctx, query, pgx.StrictNamedArgs{
		"namespace":     namespace,
		"workload_kind": workloadKind,
		"workload_name": workloadName,
		"since":         since,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query attributed pods: %w", err)
	}
	defer rows.Close()

	pods := make([]*Pod, 0)
	for rows.Next() {
		var p Pod
		if err := rows.Scan(
			&p.UID, &p.Name, &p.Namespace, &p.Node, &p.Phase,
			&p.WorkloadKind, &p.WorkloadName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attributed pod: %w", err)
		}
		pods = append(pods, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate attributed pods: %w", err)
	}
	return pods, nil
}
