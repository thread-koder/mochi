package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/apperrors"
)

const maxHistoricalAttributionNames = 256

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
			labels, annotations, workload_kind, workload_name, created_at, synced_at
		) VALUES (
			@name, @namespace, @uid, @node, @pod_ip, @phase, @restart_policy,
			@labels, @annotations, @workload_kind, @workload_name, @created_at, @synced_at
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
			workload_kind = EXCLUDED.workload_kind,
			workload_name = EXCLUDED.workload_name,
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
			"workload_kind":  pod.WorkloadKind,
			"workload_name":  pod.WorkloadName,
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
		       labels, annotations, workload_kind, workload_name,
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
		&p.Labels, &p.Annotations, &p.WorkloadKind, &p.WorkloadName,
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
		       labels, annotations, workload_kind, workload_name,
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
		&p.Labels, &p.Annotations, &p.WorkloadKind, &p.WorkloadName,
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

// GetPodIdentityByUID returns the live pod, or a synthesized pod from attribution after prune.
func GetPodIdentityByUID(ctx context.Context, uid string) (*Pod, error) {
	pod, err := GetPodByUID(ctx, uid)
	if err == nil {
		return pod, nil
	}
	if !errors.Is(err, &apperrors.NotFoundError{}) {
		return nil, err
	}

	attr, err := getPodAttributionByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if attr == nil {
		return nil, apperrors.NewNotFound("pod", uid)
	}
	return &Pod{
		UID:          attr.UID,
		Name:         attr.Name,
		Namespace:    attr.Namespace,
		Node:         attr.Node,
		Phase:        attr.Phase,
		WorkloadKind: attr.WorkloadKind,
		WorkloadName: attr.WorkloadName,
	}, nil
}

func GetPodsByIP(ctx context.Context, ip string) ([]*Pod, error) {
	query := `
		SELECT id, name, namespace, uid, node, pod_ip, phase, restart_policy,
		       labels, annotations, workload_kind, workload_name,
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
			&p.Labels, &p.Annotations, &p.WorkloadKind, &p.WorkloadName,
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

func GetPodsByWorkload(ctx context.Context, workloadKind string, workloadName string, namespace string) ([]*Pod, error) {
	query := `
		SELECT id, name, namespace, uid, node, pod_ip, phase, restart_policy,
		       labels, annotations, workload_kind, workload_name,
		       created_at, updated_at, synced_at
		FROM pods
		WHERE workload_kind = @workload_kind AND workload_name = @workload_name AND namespace = @namespace
		ORDER BY name
	`

	rows, err := Pool.Query(ctx, query,
		pgx.StrictNamedArgs{
			"workload_kind": workloadKind,
			"workload_name": workloadName,
			"namespace":     namespace,
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
			&p.Labels, &p.Annotations, &p.WorkloadKind, &p.WorkloadName,
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

// GetStandalonePodsByNamespace returns pods with no workload identity.
func GetStandalonePodsByNamespace(ctx context.Context, namespace string) ([]*Pod, error) {
	query := `
		SELECT id, name, namespace, uid, node, pod_ip, phase, restart_policy,
		       labels, annotations, workload_kind, workload_name,
		       created_at, updated_at, synced_at
		FROM pods
		WHERE namespace = @namespace AND workload_kind IS NULL
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
			&p.Labels, &p.Annotations, &p.WorkloadKind, &p.WorkloadName,
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

// GetNodePodsByNamespace returns Node-owned pods (system/mirror).
func GetNodePodsByNamespace(ctx context.Context, namespace string) ([]*Pod, error) {
	query := `
		SELECT id, name, namespace, uid, node, pod_ip, phase, restart_policy,
		       labels, annotations, workload_kind, workload_name,
		       created_at, updated_at, synced_at
		FROM pods
		WHERE namespace = @namespace AND workload_kind = 'Node'
		ORDER BY name
	`

	rows, err := Pool.Query(ctx, query,
		pgx.StrictNamedArgs{"namespace": namespace},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query node pods by namespace: %w", err)
	}
	defer rows.Close()

	pods := make([]*Pod, 0)
	for rows.Next() {
		var p Pod
		err := rows.Scan(
			&p.ID, &p.Name, &p.Namespace, &p.UID, &p.Node, &p.PodIP, &p.Phase, &p.RestartPolicy,
			&p.Labels, &p.Annotations, &p.WorkloadKind, &p.WorkloadName,
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

// GetPodsForAnalysis merges live pods with attributions seen in the analysis window.
func GetPodsForAnalysis(ctx context.Context, workloadKind, workloadName, namespace string, since time.Time) (PodsForAnalysis, error) {
	live, err := GetPodsByWorkload(ctx, workloadKind, workloadName, namespace)
	if err != nil {
		return PodsForAnalysis{}, err
	}

	attributed, err := getAttributedPods(ctx, workloadKind, workloadName, namespace, since)
	if err != nil {
		return PodsForAnalysis{}, err
	}

	return mergePodsForAnalysis(live, attributed), nil
}

func mergePodsForAnalysis(live, attributed []*Pod) PodsForAnalysis {
	liveUIDs := make(map[string]struct{}, len(live))
	liveNames := make(map[string]struct{}, len(live))
	for _, pod := range live {
		liveUIDs[pod.UID] = struct{}{}
		liveNames[pod.Name] = struct{}{}
	}

	all := make([]*Pod, 0, len(live)+len(attributed))
	all = append(all, live...)

	historicalNames := make(map[string]struct{})
	for _, pod := range attributed {
		if _, ok := liveUIDs[pod.UID]; ok {
			continue
		}
		_, nameKnown := liveNames[pod.Name]
		if !nameKnown {
			if _, ok := historicalNames[pod.Name]; !ok {
				if len(historicalNames) >= maxHistoricalAttributionNames {
					continue
				}
				historicalNames[pod.Name] = struct{}{}
			}
		}
		all = append(all, pod)
	}

	return PodsForAnalysis{Live: live, All: all}
}

// UniquePodNames returns deduplicated pod names for Prom selectors, preserving order.
func UniquePodNames(pods []*Pod) []string {
	if len(pods) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(pods))
	names := make([]string, 0, len(pods))
	for _, pod := range pods {
		if _, ok := seen[pod.Name]; ok {
			continue
		}
		seen[pod.Name] = struct{}{}
		names = append(names, pod.Name)
	}
	return names
}

// PrunePods deletes pods not listed in uids.
// Empty uids deletes every pod in the namespace.
func PrunePods(ctx context.Context, namespace string, uids []string) error {
	var err error
	if len(uids) == 0 {
		_, err = Pool.Exec(ctx,
			`DELETE FROM pods WHERE namespace = @namespace`,
			pgx.StrictNamedArgs{"namespace": namespace},
		)
	} else {
		_, err = Pool.Exec(ctx,
			`DELETE FROM pods WHERE namespace = @namespace AND NOT (uid = ANY(@uids::text[]))`,
			pgx.StrictNamedArgs{
				"namespace": namespace,
				"uids":      uids,
			},
		)
	}
	if err != nil {
		return fmt.Errorf("failed to prune pods: %w", err)
	}
	return nil
}
