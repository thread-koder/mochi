package kubernetes

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/thread_koder/mochi/internal/config"
	"github.com/thread_koder/mochi/internal/logger"
)

var (
	// Global Kubernetes clientset
	Clientset *kubernetes.Clientset
	// Global Kubernetes REST config
	RestConfig *rest.Config
)

// Holds information about the connected Kubernetes cluster
type ClusterInfo struct {
	ServerVersion string
	ClusterName   string
	ContextName   string
	APIServerURL  string
}

// Initializes the Kubernetes client with the provided configuration
func Init(cfg *config.KubernetesConfig) error {
	var err error
	var kubeconfig string

	log := logger.WithComponent("kubernetes")
	log.Info().Msg("Initializing client...")

	// Determine kubeconfig path
	if cfg.KubeconfigPath != "" {
		kubeconfig = cfg.KubeconfigPath
	} else {
		// Try default locations
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	// Build config from kubeconfig file or in-cluster config
	if cfg.InCluster {
		log.Info().Msg("Using in-cluster config")
		RestConfig, err = rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	} else if kubeconfig != "" {
		log.Info().Str("kubeconfig", kubeconfig).Msg("Loading config from file")
		RestConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	} else {
		return fmt.Errorf("no kubeconfig path provided and not running in-cluster")
	}

	// Configure timeouts if specified
	if cfg.RequestTimeout > 0 {
		RestConfig.Timeout = time.Duration(cfg.RequestTimeout) * time.Second
	}

	// Set QPS and Burst for rate limiting
	if cfg.QPS > 0 {
		RestConfig.QPS = float32(cfg.QPS)
	}
	if cfg.Burst > 0 {
		RestConfig.Burst = cfg.Burst
	}

	// Create the clientset
	Clientset, err = kubernetes.NewForConfig(RestConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	// Test connection by getting server version
	version, err := Clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to Kubernetes cluster: %w", err)
	}

	log.Info().
		Str("version", version.String()).
		Str("api_server", RestConfig.Host).
		Msg("Connection established")

	return nil
}

// Returns information about the connected Kubernetes cluster
func GetClusterInfo(ctx context.Context) (*ClusterInfo, error) {
	if Clientset == nil {
		return nil, fmt.Errorf("Kubernetes client not initialized")
	}

	info := &ClusterInfo{
		APIServerURL: RestConfig.Host,
	}

	// Get server version
	version, err := Clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get server version: %w", err)
	}
	info.ServerVersion = version.String()

	// TODO: Extract context name from kubeconfig if available
	info.ContextName = "default"
	info.ClusterName = "default"

	return info, nil
}

// Performs a health check on the Kubernetes connection
func HealthCheck(ctx context.Context) error {
	if Clientset == nil {
		return fmt.Errorf("Kubernetes client not initialized")
	}

	// Try to get server version as a health check
	_, err := Clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("Kubernetes health check failed: %w", err)
	}

	return nil
}
