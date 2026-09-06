package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds agent runtime settings from environment variables.
type Config struct {
	MetricsHost string
	MetricsPort int
	LogLevel    string
	LogFormat   string
	EBPFEnabled bool
	MaxSeries   int
	NodeName    string
}

func Load() (Config, error) {
	metricsPort, err := envInt("METRICS_PORT", 9800)
	if err != nil {
		return Config{}, fmt.Errorf("METRICS_PORT: %w", err)
	}

	maxSeries, err := envInt("MAX_SERIES", 100000)
	if err != nil {
		return Config{}, fmt.Errorf("MAX_SERIES: %w", err)
	}
	if maxSeries <= 0 {
		return Config{}, fmt.Errorf("MAX_SERIES must be > 0, got: %d", maxSeries)
	}

	ebpfEnabled, err := envBool("EBPF_ENABLED", true)
	if err != nil {
		return Config{}, fmt.Errorf("EBPF_ENABLED: %w", err)
	}

	cfg := Config{
		MetricsHost: envOr("METRICS_HOST", "0.0.0.0"),
		MetricsPort: metricsPort,
		LogLevel:    envOr("LOG_LEVEL", "info"),
		LogFormat:   envOr("LOG_FORMAT", "console"),
		EBPFEnabled: ebpfEnabled,
		MaxSeries:   maxSeries,
		NodeName:    strings.TrimSpace(os.Getenv("NODE_NAME")),
	}

	if cfg.EBPFEnabled && cfg.NodeName == "" {
		return Config{}, fmt.Errorf("NODE_NAME is required when EBPF_ENABLED=true")
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q", raw)
	}
	return value, nil
}

func envBool(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("invalid boolean %q", raw)
	}
	return value, nil
}
