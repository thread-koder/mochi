package kubernetes

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Holds basic information about cluster resources
type ClusterResources struct {
	Namespaces        int
	Nodes             int
	Pods              int
	Services          int
	Deployments       int
	StatefulSets      int
	DaemonSets        int
	ConfigMaps        int
	Secrets           int
	PersistentVolumes int
}

// Performs cluster discovery and returns resource counts
func DiscoverCluster(ctx context.Context) (*ClusterResources, error) {
	if Clientset == nil {
		return nil, fmt.Errorf("Kubernetes client not initialized")
	}

	resources := &ClusterResources{}

	// Discover namespaces
	namespaces, err := Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}
	resources.Namespaces = len(namespaces.Items)

	// Discover nodes
	nodes, err := Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	resources.Nodes = len(nodes.Items)

	// Discover pods (across all namespaces)
	pods, err := Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	resources.Pods = len(pods.Items)

	// Discover services (across all namespaces)
	services, err := Clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	resources.Services = len(services.Items)

	// Discover deployments (across all namespaces)
	deployments, err := Clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	resources.Deployments = len(deployments.Items)

	// Discover statefulsets (across all namespaces)
	statefulSets, err := Clientset.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list statefulsets: %w", err)
	}
	resources.StatefulSets = len(statefulSets.Items)

	// Discover daemonsets (across all namespaces)
	daemonSets, err := Clientset.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list daemonsets: %w", err)
	}
	resources.DaemonSets = len(daemonSets.Items)

	// Discover configmaps (across all namespaces)
	configMaps, err := Clientset.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list configmaps: %w", err)
	}
	resources.ConfigMaps = len(configMaps.Items)

	// Discover secrets (across all namespaces)
	secrets, err := Clientset.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	resources.Secrets = len(secrets.Items)

	// Discover persistent volumes
	pvs, err := Clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list persistent volumes: %w", err)
	}
	resources.PersistentVolumes = len(pvs.Items)

	return resources, nil
}
