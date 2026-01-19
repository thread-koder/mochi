package config

import (
	"fmt"
	"net/url"
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
	Redis      RedisConfig      `mapstructure:"redis"`
	Workers    WorkerConfig     `mapstructure:"workers"`
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

// Holds Redis configuration
type RedisConfig struct {
	Host            string `mapstructure:"host"`               // Redis host
	Port            int    `mapstructure:"port"`               // Redis port
	Password        string `mapstructure:"password"`           // Redis password (empty = no auth)
	Database        int    `mapstructure:"database"`           // Redis database number (0-15)
	MaxRetries      int    `mapstructure:"max_retries"`        // Maximum number of retries
	PoolSize        int    `mapstructure:"pool_size"`          // Connection pool size
	MinIdleConns    int    `mapstructure:"min_idle_conns"`     // Minimum idle connections
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`  // Maximum connection lifetime in seconds
	ConnMaxIdleTime int    `mapstructure:"conn_max_idle_time"` // Maximum idle time in seconds
	CacheTTL        int    `mapstructure:"cache_ttl"`          // Default cache TTL in seconds
}

// Holds worker configuration
type WorkerConfig struct {
	ResourceSyncInterval   int      `mapstructure:"resource_sync_interval"`   // Resource sync interval in seconds
	StaleResourceThreshold int      `mapstructure:"stale_resource_threshold"` // Stale resource threshold in seconds
	ExcludeNamespaces      []string `mapstructure:"exclude_namespaces"`       // List of namespaces to exclude from syncing
	IncludeNamespaces      []string `mapstructure:"include_namespaces"`       // If non-empty, only sync these namespaces (exclude_namespaces ignored)
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

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
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

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "mochi")
	viper.SetDefault("redis.database", 0)
	viper.SetDefault("redis.max_retries", 3)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.min_idle_conns", 5)
	viper.SetDefault("redis.conn_max_lifetime", 300)
	viper.SetDefault("redis.conn_max_idle_time", 60)
	viper.SetDefault("redis.cache_ttl", 300)

	// Worker defaults
	viper.SetDefault("workers.resource_sync_interval", 180)   // 3 minutes
	viper.SetDefault("workers.stale_resource_threshold", 300) // 5 minutes
	viper.SetDefault("workers.exclude_namespaces", []string{"default", "kube-system", "kube-public", "kube-node-lease"})
	viper.SetDefault("workers.include_namespaces", []string{})
}

// Validates the entire configuration
func (c *Config) Validate() error {
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log config: %w", err)
	}
	if err := c.API.Validate(); err != nil {
		return fmt.Errorf("api config: %w", err)
	}
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database config: %w", err)
	}
	if err := c.Kubernetes.Validate(); err != nil {
		return fmt.Errorf("kubernetes config: %w", err)
	}
	if err := c.Prometheus.Validate(); err != nil {
		return fmt.Errorf("prometheus config: %w", err)
	}
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("redis config: %w", err)
	}
	if err := c.Workers.Validate(); err != nil {
		return fmt.Errorf("workers config: %w", err)
	}
	return nil
}

// Validates log configuration
func (c *LogConfig) Validate() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[strings.ToLower(c.Level)] {
		return fmt.Errorf("level must be one of: debug, info, warn, error, got: %s", c.Level)
	}

	validFormats := map[string]bool{
		"json":    true,
		"console": true,
	}
	if !validFormats[strings.ToLower(c.Format)] {
		return fmt.Errorf("format must be one of: json, console, got: %s", c.Format)
	}
	return nil
}

// Validates API configuration
func (c *APIConfig) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got: %d", c.Port)
	}
	if c.ReadTimeout <= 0 {
		return fmt.Errorf("read_timeout must be greater than 0, got: %d", c.ReadTimeout)
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("write_timeout must be greater than 0, got: %d", c.WriteTimeout)
	}

	validModes := map[string]bool{
		"debug":   true,
		"release": true,
		"test":    true,
	}
	if !validModes[strings.ToLower(c.Mode)] {
		return fmt.Errorf("mode must be one of: debug, release, test, got: %s", c.Mode)
	}
	return nil
}

// Validates database configuration
func (c *DatabaseConfig) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got: %d", c.Port)
	}
	if c.MaxConnections <= 0 {
		return fmt.Errorf("max_connections must be greater than 0, got: %d", c.MaxConnections)
	}
	if c.MaxIdleConns < 0 {
		return fmt.Errorf("max_idle_conns must be non-negative, got: %d", c.MaxIdleConns)
	}
	if c.ConnMaxLifetime <= 0 {
		return fmt.Errorf("conn_max_lifetime must be greater than 0, got: %d", c.ConnMaxLifetime)
	}
	if c.ConnMaxIdleTime <= 0 {
		return fmt.Errorf("conn_max_idle_time must be greater than 0, got: %d", c.ConnMaxIdleTime)
	}

	validSSLModes := map[string]bool{
		"disable":     true,
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
	}
	if !validSSLModes[strings.ToLower(c.SSLMode)] {
		return fmt.Errorf("ssl_mode must be one of: disable, require, verify-ca, verify-full, got: %s", c.SSLMode)
	}
	return nil
}

// Validates Kubernetes configuration
func (c *KubernetesConfig) Validate() error {
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request_timeout must be greater than 0, got: %d", c.RequestTimeout)
	}
	if c.QPS <= 0 {
		return fmt.Errorf("qps must be greater than 0, got: %d", c.QPS)
	}
	if c.Burst <= 0 {
		return fmt.Errorf("burst must be greater than 0, got: %d", c.Burst)
	}
	return nil
}

// Validates Prometheus configuration
func (c *PrometheusConfig) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url cannot be empty")
	}
	if _, err := url.Parse(c.URL); err != nil {
		return fmt.Errorf("url is invalid: %w", err)
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0, got: %d", c.Timeout)
	}
	return nil
}

// Validates Redis configuration
func (c *RedisConfig) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got: %d", c.Port)
	}
	if c.Database < 0 || c.Database > 15 {
		return fmt.Errorf("database must be between 0 and 15, got: %d", c.Database)
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative, got: %d", c.MaxRetries)
	}
	if c.PoolSize <= 0 {
		return fmt.Errorf("pool_size must be greater than 0, got: %d", c.PoolSize)
	}
	if c.MinIdleConns < 0 {
		return fmt.Errorf("min_idle_conns must be non-negative, got: %d", c.MinIdleConns)
	}
	if c.ConnMaxLifetime <= 0 {
		return fmt.Errorf("conn_max_lifetime must be greater than 0, got: %d", c.ConnMaxLifetime)
	}
	if c.ConnMaxIdleTime <= 0 {
		return fmt.Errorf("conn_max_idle_time must be greater than 0, got: %d", c.ConnMaxIdleTime)
	}
	if c.CacheTTL <= 0 {
		return fmt.Errorf("cache_ttl must be greater than 0, got: %d", c.CacheTTL)
	}
	return nil
}

// Validates worker configuration
func (c *WorkerConfig) Validate() error {
	if c.ResourceSyncInterval <= 0 {
		return fmt.Errorf("resource_sync_interval must be greater than 0, got: %d", c.ResourceSyncInterval)
	}
	if c.StaleResourceThreshold <= 0 {
		return fmt.Errorf("stale_resource_threshold must be greater than 0, got: %d", c.StaleResourceThreshold)
	}
	if c.ExcludeNamespaces == nil {
		c.ExcludeNamespaces = []string{}
	}
	if c.IncludeNamespaces == nil {
		c.IncludeNamespaces = []string{}
	}
	return nil
}
