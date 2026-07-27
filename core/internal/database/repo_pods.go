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
			name, namespace, uid, node, pod_ip, phase, restart_policy,
			labels, annotations, owner_kind, owner_name, created_at, synced_at
		) VALUES (
			@name, @namespace, @uid, @node, @pod_ip, @phase, @restart_policy,
			@labels, @annotations, @owner_kind, @owner_name, @created_at, @synced_at
		)
		ON CONFLICT (uid) DO UPDATE SET
			name = EXCLUDED.name,
			namespace = EXCLUDED.namespace,
			node = EXCLUDED.node,
			pod_ip = EXCLUDED.pod_ip,
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
		batch.Queue(query, pgx.StrictNamedArgs{
			"name":           pod.Name,
			"namespace":      pod.Namespace,
			"uid":            pod.UID,
			"node":           pod.Node,
			"pod_ip":         pod.PodIP,
			"phase":          pod.Phase,
			"restart_policy": pod.RestartPolicy,
			"labels":         pod.Labels,
			"annotations":    pod.Annotations,
			"owner_kind":     pod.OwnerKind,
			"owner_name":     pod.OwnerName,
			"created_at":     pod.CreatedAt,
			"synced_at":      pod.SyncedAt,
		})
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
		SELECT id, name, namespace, uid, node, pod_ip, phase, restart_policy,
		       labels, annotations, owner_kind, owner_name,
		       created_at, updated_at, synced_at
		FROM pods
		WHERE name = @name AND namespace = @namespace
		LIMIT 1
	`

	var p Pod
	err := Pool.QueryRow(ctx, query,
		pgx.StrictNamedArgs{
			"name":      name,
			"namespace": namespace,
		},
	).Scan(
		&p.ID, &p.Name, &p.Namespace, &p.UID, &p.Node, &p.PodIP, &p.Phase, &p.RestartPolicy,
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

func GetPodByUID(ctx context.Context, uid string) (*Pod, error) {
	query := `
		SELECT id, name, namespace, uid, node, pod_ip, phase, restart_policy,
		       labels, annotations, owner_kind, owner_name,
		       created_at, updated_at, synced_at
		FROM pods
		WHERE uid = @uid
		LIMIT 1
	`

	var p Pod
	err := Pool.QueryRow(ctx, query,
		pgx.StrictNamedArgs{"uid": uid},
	).Scan(
		&p.ID, &p.Name, &p.Namespace, &p.UID, &p.Node, &p.PodIP, &p.Phase, &p.RestartPolicy,
		&p.Labels, &p.Annotations, &p.OwnerKind, &p.OwnerName,
		&p.CreatedAt, &p.UpdatedAt, &p.SyncedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("pod", uid)
		}
		return nil, fmt.Errorf("failed to query pod by UID: %w", err)
	}

	return &p, nil
}

func GetPodsByIP(ctx context.Context, ip string) ([]*Pod, error) {
	query := `
		SELECT id, name, namespace, uid, node, pod_ip, phase, restart_policy,
		       labels, annotations, owner_kind, owner_name,
		       created_at, updated_at, synced_at
		FROM pods
		WHERE pod_ip = @pod_ip
		ORDER BY namespace, name
	`

	rows, err := Pool.Query(ctx, query,
		pgx.StrictNamedArgs{"pod_ip": ip},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query pods by IP: %w", err)
	}
	defer rows.Close()

	pods := make([]*Pod, 0)
	for rows.Next() {
		var p Pod
		err := rows.Scan(
			&p.ID, &p.Name, &p.Namespace, &p.UID, &p.Node, &p.PodIP, &p.Phase, &p.RestartPolicy,
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

// GetPodsByWorkload returns the pods for a workload. Deployment is special: pods are owned by
// ReplicaSets, so we walk RS -> Pod. Other kinds query owner_kind/owner_name directly.
func GetPodsByWorkload(ctx context.Context, workloadType string, workloadName string, namespace string) ([]*Pod, error) {
	if workloadType == "Deployment" {
		return getPodsByDeployment(ctx, workloadName, namespace)
	}

	query := `
		SELECT id, name, namespace, uid, node, pod_ip, phase, restart_policy,
		       labels, annotations, owner_kind, owner_name,
		       created_at, updated_at, synced_at
		FROM pods
		WHERE owner_kind = @owner_kind AND owner_name = @owner_name AND namespace = @namespace
		ORDER BY name
	`

	rows, err := Pool.Query(ctx, query,
		pgx.StrictNamedArgs{
			"owner_kind": workloadType,
			"owner_name": workloadName,
			"namespace":  namespace,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query pods by workload: %w", err)
	}
	defer rows.Close()

	pods := make([]*Pod, 0)
	for rows.Next() {
		var p Pod
		err := rows.Scan(
			&p.ID, &p.Name, &p.Namespace, &p.UID, &p.Node, &p.PodIP, &p.Phase, &p.RestartPolicy,
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
		SELECT id, name, namespace, uid, node, pod_ip, phase, restart_policy,
		       labels, annotations, owner_kind, owner_name,
		       created_at, updated_at, synced_at
		FROM pods
		WHERE namespace = @namespace AND owner_kind IS NULL
		ORDER BY name
	`

	rows, err := Pool.Query(ctx, query,
		pgx.StrictNamedArgs{"namespace": namespace},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query standalone pods by namespace: %w", err)
	}
	defer rows.Close()

	pods := make([]*Pod, 0)
	for rows.Next() {
		var p Pod
		err := rows.Scan(
			&p.ID, &p.Name, &p.Namespace, &p.UID, &p.Node, &p.PodIP, &p.Phase, &p.RestartPolicy,
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
		SELECT id, name, namespace, uid, node, pod_ip, phase, restart_policy,
		       labels, annotations, owner_kind, owner_name,
		       created_at, updated_at, synced_at
		FROM pods
		WHERE namespace = @namespace AND owner_kind = @owner_kind
		ORDER BY name
	`

	rows, err := Pool.Query(ctx, query,
		pgx.StrictNamedArgs{
			"namespace":  namespace,
			"owner_kind": ownerKind,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query pods by owner kind: %w", err)
	}
	defer rows.Close()

	pods := make([]*Pod, 0)
	for rows.Next() {
		var p Pod
		err := rows.Scan(
			&p.ID, &p.Name, &p.Namespace, &p.UID, &p.Node, &p.PodIP, &p.Phase, &p.RestartPolicy,
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
		_, err := Pool.Exec(ctx,
			`DELETE FROM pods WHERE namespace = @namespace`,
			pgx.StrictNamedArgs{"namespace": namespace},
		)
		return err
	}
	_, err := Pool.Exec(ctx,
		`DELETE FROM pods WHERE namespace = @namespace AND NOT (uid = ANY(@uids::text[]))`,
		pgx.StrictNamedArgs{
			"namespace": namespace,
			"uids":      uids,
		},
	)
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
