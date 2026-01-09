package database

import (
	"context"
	"fmt"
)

// Gets the count of namespaces
func GetNamespaceCount(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM namespaces`
	err := Pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count namespaces: %w", err)
	}
	return count, nil
}

// Gets the count of workloads (deployments + statefulsets + daemonsets)
func GetWorkloadCount(ctx context.Context) (int, error) {
	var count int
	query := `
		SELECT 
			(SELECT COUNT(*) FROM deployments) +
			(SELECT COUNT(*) FROM statefulsets) +
			(SELECT COUNT(*) FROM daemonsets) as total
	`
	err := Pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count workloads: %w", err)
	}
	return count, nil
}

// Gets the count of pods
func GetPodCount(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM pods`
	err := Pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count pods: %w", err)
	}
	return count, nil
}

// Gets the count of pods in a specific namespace
func GetPodCountByNamespace(ctx context.Context, namespace string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM pods WHERE namespace = $1`
	err := Pool.QueryRow(ctx, query, namespace).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count pods by namespace: %w", err)
	}
	return count, nil
}

// Gets the count of containers in a specific namespace
func GetContainerCountByNamespace(ctx context.Context, namespace string) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) 
		FROM containers c
		INNER JOIN pods p ON c.pod_uid = p.uid
		WHERE p.namespace = $1
	`
	err := Pool.QueryRow(ctx, query, namespace).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count containers by namespace: %w", err)
	}
	return count, nil
}
