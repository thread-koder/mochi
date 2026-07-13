package kubernetes

import (
	"context"
	"fmt"
	"path/filepath"

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

	RestConfig.Timeout = cfg.RequestTimeoutDuration()
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

func HealthCheck(ctx context.Context) error {
	_, err := Clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("kubernetes health check failed: %w", err)
	}

	return nil
}
