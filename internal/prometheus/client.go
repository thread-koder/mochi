package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/thread_koder/mochi/internal/config"
	"github.com/thread_koder/mochi/internal/logger"
)

var (
	// Client is the global Prometheus HTTP client.
	Client api.Client
	// API is the global Prometheus v1 API handle.
	API v1.API
)

// Init configures and verifies the Prometheus client connection.
func Init(cfg *config.PrometheusConfig) error {
	if cfg == nil {
		return fmt.Errorf("prometheus config is nil")
	}

	log := logger.WithComponent("prometheus")
	log.Info().Msg("Initializing client...")

	transport := &http.Transport{}
	tlsConfig, err := config.BuildTLSConfig(cfg.TLS)
	if err != nil {
		return fmt.Errorf("build TLS config: %w", err)
	}
	transport.TLSClientConfig = tlsConfig

	clientConfig := api.Config{
		Address: cfg.URL,
		Client: &http.Client{
			Transport: transport,
			Timeout:   time.Duration(cfg.Timeout) * time.Second,
		},
	}

	client, err := api.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create prometheus client: %w", err)
	}

	Client = client
	API = v1.NewAPI(client)

	// Buildinfo is a cheap endpoint that proves the API is reachable.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = API.Buildinfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to prometheus server: %w", err)
	}

	log.Info().
		Str("url", cfg.URL).
		Str("timeout", fmt.Sprintf("%ds", cfg.Timeout)).
		Msg("Connection established")

	return nil
}

// HealthCheck verifies the API reachability with a Buildinfo call.
func HealthCheck(ctx context.Context) error {
	if Client == nil {
		return fmt.Errorf("prometheus client not initialized")
	}

	_, err := API.Buildinfo(ctx)
	if err != nil {
		return fmt.Errorf("prometheus health check failed: %w", err)
	}

	return nil
}
