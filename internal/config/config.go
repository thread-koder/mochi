package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Holds TLS settings for clients
type TLSConfig struct {
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"` // Skip server certificate verification
	RootCAPath         string `mapstructure:"root_ca_path"`         // Path to CA cert for server verification (optional)
	ClientCertPath     string `mapstructure:"client_cert_path"`     // Path to client cert for mutual TLS (optional)
	ClientKeyPath      string `mapstructure:"client_key_path"`      // Path to client key for mutual TLS (optional)
}

// Holds all application configuration
type Config struct {
	Log        LogConfig        `mapstructure:"log"`
	API        APIConfig        `mapstructure:"api"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Workers    WorkerConfig     `mapstructure:"workers"`
	Compute    ComputeConfig    `mapstructure:"compute"`
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
	Host            string    `mapstructure:"host"`               // PostgreSQL host
	Port            int       `mapstructure:"port"`               // PostgreSQL port
	User            string    `mapstructure:"user"`               // PostgreSQL user
	Password        string    `mapstructure:"password"`           // PostgreSQL password
	Database        string    `mapstructure:"database"`           // PostgreSQL database name
	SSLMode         string    `mapstructure:"ssl_mode"`           // SSL mode (disable, require, verify-ca, verify-full)
	MaxConnections  int       `mapstructure:"max_connections"`    // Maximum number of connections in pool
	MinIdleConns    int       `mapstructure:"min_idle_conns"`     // Minimum number of idle connections in pool
	ConnMaxLifetime int       `mapstructure:"conn_max_lifetime"`  // Maximum connection lifetime in seconds
	ConnMaxIdleTime int       `mapstructure:"conn_max_idle_time"` // Maximum idle time in seconds
	TLS             TLSConfig `mapstructure:"tls"`                // TLS config (used when ssl_mode != disable)
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
	URL     string    `mapstructure:"url"`     // Prometheus server URL
	Timeout int       `mapstructure:"timeout"` // Request timeout in seconds
	TLS     TLSConfig `mapstructure:"tls"`     // TLS config for HTTPS
}

// Holds Redis configuration
type RedisConfig struct {
	Host            string    `mapstructure:"host"`               // Redis host
	Port            int       `mapstructure:"port"`               // Redis port
	Username        string    `mapstructure:"username"`           // Redis username (ACL, empty = default)
	Password        string    `mapstructure:"password"`           // Redis password (empty = no auth)
	Database        int       `mapstructure:"database"`           // Redis database number (0-15)
	MaxRetries      int       `mapstructure:"max_retries"`        // Maximum number of retries
	PoolSize        int       `mapstructure:"pool_size"`          // Connection pool size
	MinIdleConns    int       `mapstructure:"min_idle_conns"`     // Minimum idle connections
	ConnMaxLifetime int       `mapstructure:"conn_max_lifetime"`  // Maximum connection lifetime in seconds
	ConnMaxIdleTime int       `mapstructure:"conn_max_idle_time"` // Maximum idle time in seconds
	CacheTTL        int       `mapstructure:"cache_ttl"`          // Default cache TTL in seconds
	UseTLS          bool      `mapstructure:"use_tls"`            // Enable TLS for connection
	TLS             TLSConfig `mapstructure:"tls"`                // TLS config (used when use_tls is true)
}

// Holds worker configuration
type WorkerConfig struct {
	ResourceSyncInterval int      `mapstructure:"resource_sync_interval"` // Resource sync interval in seconds
	ExcludeNamespaces    []string `mapstructure:"exclude_namespaces"`     // List of namespaces to exclude from syncing
	IncludeNamespaces    []string `mapstructure:"include_namespaces"`     // If non-empty, only sync these namespaces (exclude_namespaces ignored)
	Retention            int      `mapstructure:"retention"`              // Keep data for this many days
}

// Holds compute recommendation configuration
type ComputeConfig struct {
	MinConfidenceThreshold float64 `mapstructure:"min_confidence_threshold"` // Minimum confidence (0-1) to generate recommendations
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
	viper.SetDefault("database.min_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", 3600)
	viper.SetDefault("database.conn_max_idle_time", 1800)
	viper.SetDefault("database.tls.insecure_skip_verify", false)
	viper.SetDefault("database.tls.root_ca_path", "")
	viper.SetDefault("database.tls.client_cert_path", "")
	viper.SetDefault("database.tls.client_key_path", "")

	// Kubernetes defaults
	viper.SetDefault("kubernetes.kubeconfig_path", "")
	viper.SetDefault("kubernetes.in_cluster", false)
	viper.SetDefault("kubernetes.request_timeout", 30)
	viper.SetDefault("kubernetes.qps", 50)
	viper.SetDefault("kubernetes.burst", 100)

	// Prometheus defaults
	viper.SetDefault("prometheus.url", "http://localhost:9090")
	viper.SetDefault("prometheus.timeout", 30)
	viper.SetDefault("prometheus.tls.insecure_skip_verify", false)
	viper.SetDefault("prometheus.tls.root_ca_path", "")
	viper.SetDefault("prometheus.tls.client_cert_path", "")
	viper.SetDefault("prometheus.tls.client_key_path", "")

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.username", "")
	viper.SetDefault("redis.password", "mochi")
	viper.SetDefault("redis.database", 0)
	viper.SetDefault("redis.max_retries", 3)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.min_idle_conns", 5)
	viper.SetDefault("redis.conn_max_lifetime", 3600)
	viper.SetDefault("redis.conn_max_idle_time", 1800)
	viper.SetDefault("redis.cache_ttl", 300)
	viper.SetDefault("redis.use_tls", false)
	viper.SetDefault("redis.tls.insecure_skip_verify", false)
	viper.SetDefault("redis.tls.root_ca_path", "")
	viper.SetDefault("redis.tls.client_cert_path", "")
	viper.SetDefault("redis.tls.client_key_path", "")

	// Worker defaults
	viper.SetDefault("workers.resource_sync_interval", 120) // 2 minutes
	viper.SetDefault("workers.exclude_namespaces", []string{"default", "kube-system", "kube-public", "kube-node-lease"})
	viper.SetDefault("workers.include_namespaces", []string{})
	viper.SetDefault("workers.retention", 90) // Days

	// Compute defaults
	viper.SetDefault("compute.min_confidence_threshold", 0.8)
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
	if err := c.Compute.Validate(); err != nil {
		return fmt.Errorf("compute config: %w", err)
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
	if !validLevels[c.Level] {
		return fmt.Errorf("level must be one of: debug, info, warn, error, got: %s", c.Level)
	}

	validFormats := map[string]bool{
		"json":    true,
		"console": true,
	}
	if !validFormats[c.Format] {
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
	if !validModes[c.Mode] {
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
	if c.MinIdleConns < 0 {
		return fmt.Errorf("min_idle_conns must be non-negative, got: %d", c.MinIdleConns)
	}
	if c.MinIdleConns > c.MaxConnections {
		return fmt.Errorf("min_idle_conns must be less than or equal to max_connections, got: %d", c.MinIdleConns)
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
	if !validSSLModes[c.SSLMode] {
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
	if c.Retention <= 0 {
		return fmt.Errorf("retention must be at least 1 day, got: %d", c.Retention)
	}
	if c.Retention > 3650 {
		return fmt.Errorf("retention must be at most 3650 days (about 10 years), got: %d", c.Retention)
	}
	if c.ExcludeNamespaces == nil {
		c.ExcludeNamespaces = []string{}
	}
	if c.IncludeNamespaces == nil {
		c.IncludeNamespaces = []string{}
	}
	return nil
}

// Validates compute configuration
func (c *ComputeConfig) Validate() error {
	if c.MinConfidenceThreshold <= 0.2 {
		return fmt.Errorf("min_confidence_threshold must be greater than 0.2 (values 0.2 or lower produce low-confidence recommendations); use a value between 0.5 and 0.95, got: %g", c.MinConfidenceThreshold)
	}
	if c.MinConfidenceThreshold >= 1.0 {
		return fmt.Errorf("min_confidence_threshold must be less than 1.0 (1.0 would exclude all recommendations); use a value between 0.5 and 0.95, got: %g", c.MinConfidenceThreshold)
	}
	return nil
}

// builds a TLS configuration for TLS clients
func BuildTLSConfig(cfg TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	if cfg.RootCAPath != "" {
		pem, err := os.ReadFile(cfg.RootCAPath)
		if err != nil {
			return nil, fmt.Errorf("read root CA: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("failed to parse root CA from %s", cfg.RootCAPath)
		}
		tlsConfig.RootCAs = pool
	}
	if cfg.ClientCertPath != "" && cfg.ClientKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(cfg.ClientCertPath, cfg.ClientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("load client cert/key: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	return tlsConfig, nil
}
