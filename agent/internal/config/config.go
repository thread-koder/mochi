package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds agent runtime settings from environment variables.
type Config struct {
	MetricsHost    string
	MetricsPort    int
	LogLevel       string
	LogFormat      string
	EBPFEnabled    bool
	MaxSeries      int
	NodeName       string
	CoreURL        string
	SpanSampleRate float64
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

	spanSampleRate, err := envFloat("SPAN_SAMPLE_RATE", 0.01)
	if err != nil {
		return Config{}, fmt.Errorf("SPAN_SAMPLE_RATE: %w", err)
	}
	if spanSampleRate < 0 || spanSampleRate > 1 {
		return Config{}, fmt.Errorf("SPAN_SAMPLE_RATE must be in [0, 1], got: %g", spanSampleRate)
	}

	cfg := Config{
		MetricsHost:    envOr("METRICS_HOST", "0.0.0.0"),
		MetricsPort:    metricsPort,
		LogLevel:       envOr("LOG_LEVEL", "info"),
		LogFormat:      envOr("LOG_FORMAT", "console"),
		EBPFEnabled:    ebpfEnabled,
		MaxSeries:      maxSeries,
		NodeName:       strings.TrimSpace(os.Getenv("NODE_NAME")),
		CoreURL:        strings.TrimSpace(os.Getenv("CORE_URL")),
		SpanSampleRate: spanSampleRate,
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

func envFloat(key string, fallback float64) (float64, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float %q", raw)
	}
	return value, nil
}
