package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Holds all application configuration
type Config struct {
	Log        LogConfig        `mapstructure:"log"`
	API        APIConfig        `mapstructure:"api"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
}

// Holds logger configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // json, console
}

// Holds API server configuration
type APIConfig struct {
	Host         string `mapstructure:"host"`          // API server host
	Port         int    `mapstructure:"port"`          // API server port
	ReadTimeout  int    `mapstructure:"read_timeout"`  // Read timeout in seconds
	WriteTimeout int    `mapstructure:"write_timeout"` // Write timeout in seconds
	Mode         string `mapstructure:"mode"`          // debug, release, test
}

// Holds database configuration
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`               // PostgreSQL host
	Port            int    `mapstructure:"port"`               // PostgreSQL port
	User            string `mapstructure:"user"`               // PostgreSQL user
	Password        string `mapstructure:"password"`           // PostgreSQL password
	Database        string `mapstructure:"database"`           // PostgreSQL database name
	SSLMode         string `mapstructure:"ssl_mode"`           // SSL mode (disable, require, verify-ca, verify-full)
	MaxConnections  int    `mapstructure:"max_connections"`    // Maximum number of connections in pool
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`     // Maximum number of idle connections
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`  // Maximum connection lifetime in seconds
	ConnMaxIdleTime int    `mapstructure:"conn_max_idle_time"` // Maximum idle time in seconds
}

// Holds Kubernetes configuration
type KubernetesConfig struct {
	KubeconfigPath string `mapstructure:"kubeconfig_path"` // Path to kubeconfig file (empty = use default)
	InCluster      bool   `mapstructure:"in_cluster"`      // Use in-cluster config
	RequestTimeout int    `mapstructure:"request_timeout"` // Request timeout in seconds
	QPS            int    `mapstructure:"qps"`             // Queries per second (rate limiting)
	Burst          int    `mapstructure:"burst"`           // Burst limit for rate limiting
}

// Holds Prometheus configuration
type PrometheusConfig struct {
	URL                string `mapstructure:"url"`                  // Prometheus server URL
	Timeout            int    `mapstructure:"timeout"`              // Request timeout in seconds
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"` // Skip TLS certificate verification
}

var AppConfig *Config

// Reads configuration from file and environment variables
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	configPath := os.Getenv("MOCHI_CONFIG_PATH")

	// Add config paths
	if configPath != "" {
		viper.AddConfigPath(configPath)
	}

	// Setup environment variables
	viper.SetEnvPrefix("MOCHI")
	// Replace dots with underscores for nested keys (e.g., log.level -> LOG_LEVEL)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	// Read config file (optional - env vars can override)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("unable to read config file: %w", err)
		}
		// Config file not found is OK, use defaults and env vars
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("config is invalid: %w", err)
	}

	AppConfig = &config
	return &config, nil
}

func setDefaults() {
	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "console")

	// API defaults
	viper.SetDefault("api.host", "0.0.0.0")
	viper.SetDefault("api.port", 8080)
	viper.SetDefault("api.read_timeout", 30)
	viper.SetDefault("api.write_timeout", 30)
	viper.SetDefault("api.mode", "release")

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "mochi")
	viper.SetDefault("database.password", "mochi")
	viper.SetDefault("database.database", "mochi")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_connections", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", 300)
	viper.SetDefault("database.conn_max_idle_time", 60)

	// Kubernetes defaults
	viper.SetDefault("kubernetes.kubeconfig_path", "")
	viper.SetDefault("kubernetes.in_cluster", false)
	viper.SetDefault("kubernetes.request_timeout", 30)
	viper.SetDefault("kubernetes.qps", 50)
	viper.SetDefault("kubernetes.burst", 100)

	// Prometheus defaults
	viper.SetDefault("prometheus.url", "http://localhost:9090")
	viper.SetDefault("prometheus.timeout", 30)
	viper.SetDefault("prometheus.insecure_skip_verify", false)
}
