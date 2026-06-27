package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/apperrors"
	"golang.org/x/sync/errgroup"
)

func UpsertPodsBatch(ctx context.Context, pods []*Pod) error {
	if len(pods) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO pods (
			name, namespace, uid, node, phase, restart_policy,
			labels, annotations, owner_kind, owner_name, created_at, synced_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
			namespace = EXCLUDED.namespace,
			node = EXCLUDED.node,
			phase = EXCLUDED.phase,
			restart_policy = EXCLUDED.restart_policy,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			owner_kind = EXCLUDED.owner_kind,
			owner_name = EXCLUDED.owner_name,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, pod := range pods {
		batch.Queue(query,
			pod.Name, pod.Namespace, pod.UID, pod.Node, pod.Phase, pod.RestartPolicy,
			pod.Labels, pod.Annotations, pod.OwnerKind, pod.OwnerName,
			pod.CreatedAt, pod.SyncedAt,
		)
	}

	results := tx.SendBatch(ctx, batch)

	for i := range pods {
		_, err := results.Exec()
		if err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for pod %d: %w", i, err)
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

func GetPodByName(ctx context.Context, name string, namespace string) (*Pod, error) {
	query := `
		SELECT id, name, namespace, uid, node, phase, restart_policy,
		       labels, annotations, owner_kind, owner_name,
		       created_at, updated_at, synced_at
		FROM pods
		WHERE name = $1 AND namespace = $2
		LIMIT 1
	`

	var p Pod
	err := Pool.QueryRow(ctx, query, name, namespace).Scan(
		&p.ID, &p.Name, &p.Namespace, &p.UID, &p.Node, &p.Phase, &p.RestartPolicy,
		&p.Labels, &p.Annotations, &p.OwnerKind, &p.OwnerName,
		&p.CreatedAt, &p.UpdatedAt, &p.SyncedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("pod", fmt.Sprintf("%s/%s", namespace, name))
		}
		return nil, fmt.Errorf("failed to query pod by name: %w", err)
	}

	return &p, nil
}

// GetPodsByWorkload returns the pods for a workload. Deployment is special: pods are owned by
// ReplicaSets, so we walk RS -> Pod. Other kinds query owner_kind/owner_name directly.
func GetPodsByWorkload(ctx context.Context, workloadType string, workloadName string, namespace string) ([]*Pod, error) {
	if workloadType == "Deployment" {
		return getPodsByDeployment(ctx, workloadName, namespace)
	}

	query := `
		SELECT id, name, namespace, uid, node, phase, restart_policy,
		       labels, annotations, owner_kind, owner_name,
		       created_at, updated_at, synced_at
		FROM pods
		WHERE owner_kind = $1 AND owner_name = $2 AND namespace = $3
		ORDER BY name
	`

	rows, err := Pool.Query(ctx, query, workloadType, workloadName, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to query pods by workload: %w", err)
	}
	defer rows.Close()

	pods := make([]*Pod, 0)
	for rows.Next() {
		var p Pod
		err := rows.Scan(
			&p.ID, &p.Name, &p.Namespace, &p.UID, &p.Node, &p.Phase, &p.RestartPolicy,
			&p.Labels, &p.Annotations, &p.OwnerKind, &p.OwnerName,
			&p.CreatedAt, &p.UpdatedAt, &p.SyncedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pod: %w", err)
		}
		pods = append(pods, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate pods: %w", err)
	}

	return pods, nil
}

// GetStandalonePodsByNamespace returns the pods that are not owned by a workload controller.
func GetStandalonePodsByNamespace(ctx context.Context, namespace string) ([]*Pod, error) {
	query := `
		SELECT id, name, namespace, uid, node, phase, restart_policy,
		       labels, annotations, owner_kind, owner_name,
		       created_at, updated_at, synced_at
		FROM pods
		WHERE namespace = $1 AND (owner_kind IS NULL OR owner_kind = '')
		ORDER BY name
	`

	rows, err := Pool.Query(ctx, query, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to query standalone pods by namespace: %w", err)
	}
	defer rows.Close()

	pods := make([]*Pod, 0)
	for rows.Next() {
		var p Pod
		err := rows.Scan(
			&p.ID, &p.Name, &p.Namespace, &p.UID, &p.Node, &p.Phase, &p.RestartPolicy,
			&p.Labels, &p.Annotations, &p.OwnerKind, &p.OwnerName,
			&p.CreatedAt, &p.UpdatedAt, &p.SyncedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pod: %w", err)
		}
		pods = append(pods, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate pods: %w", err)
	}

	return pods, nil
}

func GetPodsByOwnerKind(ctx context.Context, ownerKind string, namespace string) ([]*Pod, error) {
	query := `
		SELECT id, name, namespace, uid, node, phase, restart_policy,
		       labels, annotations, owner_kind, owner_name,
		       created_at, updated_at, synced_at
		FROM pods
		WHERE namespace = $1 AND owner_kind = $2
		ORDER BY name
	`

	rows, err := Pool.Query(ctx, query, namespace, ownerKind)
	if err != nil {
		return nil, fmt.Errorf("failed to query pods by owner kind: %w", err)
	}
	defer rows.Close()

	pods := make([]*Pod, 0)
	for rows.Next() {
		var p Pod
		err := rows.Scan(
			&p.ID, &p.Name, &p.Namespace, &p.UID, &p.Node, &p.Phase, &p.RestartPolicy,
			&p.Labels, &p.Annotations, &p.OwnerKind, &p.OwnerName,
			&p.CreatedAt, &p.UpdatedAt, &p.SyncedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pod: %w", err)
		}
		pods = append(pods, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate pods: %w", err)
	}

	return pods, nil
}

// PrunePods deletes pods not listed in uids.
// Empty uids deletes every pod in the namespace.
func PrunePods(ctx context.Context, namespace string, uids []string) error {
	if len(uids) == 0 {
		_, err := Pool.Exec(ctx, `DELETE FROM pods WHERE namespace = $1`, namespace)
		return err
	}
	_, err := Pool.Exec(ctx, `DELETE FROM pods WHERE namespace = $1 AND NOT (uid = ANY($2::text[]))`, namespace, uids)
	return err
}

func getPodsByDeployment(ctx context.Context, deploymentName, namespace string) ([]*Pod, error) {
	replicasets, err := GetReplicaSetsByDeployment(ctx, deploymentName, namespace)
	if err != nil {
		return nil, err
	}

	if len(replicasets) == 0 {
		return []*Pod{}, nil
	}

	podSets := make([][]*Pod, len(replicasets))
	g, gctx := errgroup.WithContext(ctx)

	for i, replicaset := range replicasets {
		g.Go(func() error {
			pods, err := GetPodsByWorkload(gctx, "ReplicaSet", replicaset.Name, namespace)
			if err != nil {
				// Best-effort: omit pods for one RS if the subquery fails, so caller still gets partial data.
				return nil
			}
			podSets[i] = pods
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	allPods := make([]*Pod, 0)
	for _, pods := range podSets {
		allPods = append(allPods, pods...)
	}

	return allPods, nil
}
