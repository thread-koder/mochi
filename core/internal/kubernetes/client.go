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

	"github.com/thread_koder/mochi/core/internal/config"
	"github.com/thread_koder/mochi/core/internal/logger"
)

var (
	// Clientset is the global Kubernetes client.
	Clientset *kubernetes.Clientset
	// RestConfig is the global client configuration used by Clientset.
	RestConfig *rest.Config
)

// ClusterInfo describes the connected cluster.
type ClusterInfo struct {
	ServerVersion string `json:"server_version"`
	ClusterName   string `json:"cluster_name"`
	ContextName   string `json:"context_name"`
	APIServerURL  string `json:"api_server_url"`
}

// Init configures and verifies the Kubernetes client connection.
func Init(cfg *config.KubernetesConfig) error {
	if cfg == nil {
		return fmt.Errorf("kubernetes config is nil")
	}

	var err error
	var kubeconfig string

	log := logger.WithComponent("kubernetes")
	log.Info().Msg("Initializing client...")

	if cfg.KubeconfigPath != "" {
		kubeconfig = cfg.KubeconfigPath
	} else {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

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

	RestConfig.Timeout = time.Duration(cfg.RequestTimeout) * time.Second

	RestConfig.QPS = float32(cfg.QPS)
	RestConfig.Burst = cfg.Burst

	RestConfig.UserAgent = "mochi"

	Clientset, err = kubernetes.NewForConfig(RestConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

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

// GetClusterInfo returns basic cluster metadata.
func GetClusterInfo(ctx context.Context) (*ClusterInfo, error) {
	if Clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	info := &ClusterInfo{
		APIServerURL: RestConfig.Host,
	}

	version, err := Clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get server version: %w", err)
	}
	info.ServerVersion = version.String()

	return info, nil
}

// HealthCheck verifies the API server reachability with a discovery call.
func HealthCheck(ctx context.Context) error {
	if Clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	_, err := Clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("kubernetes health check failed: %w", err)
	}

	return nil
}
