package prometheus

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/thread_koder/mochi/internal/config"
	"github.com/thread_koder/mochi/internal/logger"
)

var (
	// Global Prometheus API client
	Client api.Client
	// Global Prometheus v1 API
	API v1.API
)

// Initializes the Prometheus client
func Init(cfg *config.PrometheusConfig) error {
	if cfg == nil {
		return fmt.Errorf("prometheus config is nil")
	}

	log := logger.WithComponent("prometheus")
	log.Info().Msg("Initializing client...")

	// Create HTTP transport
	transport := &http.Transport{}

	// Configure TLS
	if cfg.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	// Create Prometheus API client configuration
	clientConfig := api.Config{
		Address:      cfg.URL,
		RoundTripper: transport,
	}

	// Create the API client
	client, err := api.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	Client = client
	API = v1.NewAPI(client)

	// Test connection by querying Prometheus build info
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = API.Buildinfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Prometheus server: %w", err)
	}

	log.Info().
		Str("url", cfg.URL).
		Int("timeout", cfg.Timeout).
		Msg("Connection established")

	return nil
}

// Performs a health check on the Prometheus connection
func HealthCheck(ctx context.Context) error {
	if Client == nil {
		return fmt.Errorf("Prometheus client not initialized")
	}

	// Try to get build info as a health check
	_, err := API.Buildinfo(ctx)
	if err != nil {
		return fmt.Errorf("Prometheus health check failed: %w", err)
	}

	return nil
}
