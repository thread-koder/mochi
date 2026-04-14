// Package config loads, validates, and exposes application settings for Mochi.
//
// Values are resolved in order: default values, optional YAML files listed in MOCHI_CONFIG_FILES
// (comma-separated paths, the first file is the base, each later file merged on top), then
// environment variables with the MOCHI_ prefix. Nested keys use underscores in env names
// (for example log.level becomes MOCHI_LOG_LEVEL).
//
// If MOCHI_CONFIG_FILES is unset or empty, no YAML is loaded. Only defaults and environment variables apply.
// The configs/config.yaml file in the repository is a reference layout until you point
// MOCHI_CONFIG_FILES at it (or another file). Load assigns AppConfig for callers that read
// the global config after startup.
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

// TLSConfig holds optional TLS settings for outbound clients (database, Prometheus, Redis, etc.).
type TLSConfig struct {
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
	RootCAPath         string `mapstructure:"root_ca_path"`
	ClientCertPath     string `mapstructure:"client_cert_path"`
	ClientKeyPath      string `mapstructure:"client_key_path"`
}

// Config is the top-level application configuration.
type Config struct {
	Log        LogConfig        `mapstructure:"log"`
	API        APIConfig        `mapstructure:"api"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Workers    WorkerConfig     `mapstructure:"workers"`
}

// LogConfig configures application logging (level and serialization format).
// Level must be one of debug, info, warn, error. Format must be json or console.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// APIConfig configures the HTTP API server (bind address, timeouts, Gin mode).
// Mode must be one of debug, release, test. Durations are in seconds.
type APIConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	Mode         string `mapstructure:"mode"`
}

// DatabaseConfig configures the PostgreSQL client and pool.
// SSLMode must be one of disable, require, verify-ca, verify-full. TLS applies when connecting
// with encryption (so ssl_mode is not "disable").
type DatabaseConfig struct {
	Host            string    `mapstructure:"host"`
	Port            int       `mapstructure:"port"`
	User            string    `mapstructure:"user"`
	Password        string    `mapstructure:"password"`
	Database        string    `mapstructure:"database"`
	SSLMode         string    `mapstructure:"ssl_mode"`
	MaxConnections  int       `mapstructure:"max_connections"`
	MinIdleConns    int       `mapstructure:"min_idle_conns"`
	ConnMaxLifetime int       `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime int       `mapstructure:"conn_max_idle_time"`
	TLS             TLSConfig `mapstructure:"tls"`
}

// KubernetesConfig configures the Kubernetes API client (kubeconfig vs in-cluster, rate limits).
// Empty KubeconfigPath uses the default loader rules (for example KUBECONFIG or ~/.kube/config).
type KubernetesConfig struct {
	KubeconfigPath string `mapstructure:"kubeconfig_path"`
	InCluster      bool   `mapstructure:"in_cluster"`
	RequestTimeout int    `mapstructure:"request_timeout"`
	QPS            int    `mapstructure:"qps"`
	Burst          int    `mapstructure:"burst"`
}

// PrometheusConfig configures the Prometheus client base URL and timeouts.
type PrometheusConfig struct {
	URL     string    `mapstructure:"url"`
	Timeout int       `mapstructure:"timeout"`
	TLS     TLSConfig `mapstructure:"tls"`
}

// RedisConfig configures the Redis client, pool, default cache TTL, and optional TLS.
// Database is the logical DB index (0–15). When UseTLS is true, TLS fields apply.
type RedisConfig struct {
	Host            string    `mapstructure:"host"`
	Port            int       `mapstructure:"port"`
	Username        string    `mapstructure:"username"`
	Password        string    `mapstructure:"password"`
	Database        int       `mapstructure:"database"`
	MaxRetries      int       `mapstructure:"max_retries"`
	PoolSize        int       `mapstructure:"pool_size"`
	MinIdleConns    int       `mapstructure:"min_idle_conns"`
	ConnMaxLifetime int       `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime int       `mapstructure:"conn_max_idle_time"`
	CacheTTL        int       `mapstructure:"cache_ttl"`
	UseTLS          bool      `mapstructure:"use_tls"`
	TLS             TLSConfig `mapstructure:"tls"`
}

// WorkerConfig configures background sync intervals, namespace filters, and retention for stored data.
// If IncludeNamespaces is non-empty, only those namespaces are synced and ExcludeNamespaces is ignored.
type WorkerConfig struct {
	ResourceSyncInterval int      `mapstructure:"resource_sync_interval"`
	ExcludeNamespaces    []string `mapstructure:"exclude_namespaces"`
	IncludeNamespaces    []string `mapstructure:"include_namespaces"`
	Retention            int      `mapstructure:"retention"`
}

// AppConfig holds the configuration from the last successful Load.
// Call Load before reading it. Prefer threading *Config from Load where possible.
var AppConfig *Config

// Load reads configuration from defaults, optional YAML files (MOCHI_CONFIG_FILES), and MOCHI_* env vars,
// unmarshals into a Config, validates it, assigns AppConfig, and returns a pointer to the same value.
func Load() (*Config, error) {
	setDefaults()

	viper.SetConfigType("yaml")
	configFiles := parseConfigFilesEnv(os.Getenv("MOCHI_CONFIG_FILES"))
	if len(configFiles) > 0 {
		if err := readConfigFiles(configFiles); err != nil {
			return nil, err
		}
	}

	viper.SetEnvPrefix("MOCHI")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("config is invalid: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	AppConfig = &config
	return &config, nil
}

func setDefaults() {
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "console")

	viper.SetDefault("api.host", "0.0.0.0")
	viper.SetDefault("api.port", 8080)
	viper.SetDefault("api.read_timeout", 30)
	viper.SetDefault("api.write_timeout", 30)
	viper.SetDefault("api.mode", "release")

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

	viper.SetDefault("kubernetes.kubeconfig_path", "")
	viper.SetDefault("kubernetes.in_cluster", false)
	viper.SetDefault("kubernetes.request_timeout", 30)
	viper.SetDefault("kubernetes.qps", 50)
	viper.SetDefault("kubernetes.burst", 100)

	viper.SetDefault("prometheus.url", "http://localhost:9090")
	viper.SetDefault("prometheus.timeout", 30)
	viper.SetDefault("prometheus.tls.insecure_skip_verify", false)
	viper.SetDefault("prometheus.tls.root_ca_path", "")
	viper.SetDefault("prometheus.tls.client_cert_path", "")
	viper.SetDefault("prometheus.tls.client_key_path", "")

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

	viper.SetDefault("workers.resource_sync_interval", 120)
	viper.SetDefault("workers.exclude_namespaces", []string{"default", "kube-system", "kube-public", "kube-node-lease"})
	viper.SetDefault("workers.include_namespaces", []string{})
	viper.SetDefault("workers.retention", 90)
}

func parseConfigFilesEnv(pathsList string) []string {
	if pathsList == "" {
		return nil
	}
	var paths []string
	for segment := range strings.SplitSeq(pathsList, ",") {
		if path := strings.TrimSpace(segment); path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

// readConfigFiles reads the first path as the primary config, then merges each additional path in order
// so later files override overlapping keys.
func readConfigFiles(configFiles []string) error {
	viper.SetConfigFile(configFiles[0])
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("config file %q: %w", configFiles[0], err)
	}
	for _, file := range configFiles[1:] {
		viper.SetConfigFile(file)
		if err := viper.MergeInConfig(); err != nil {
			return fmt.Errorf("config file %q: %w", file, err)
		}
	}
	return nil
}

// Validate checks every section of configuration and returns the first error, wrapped with the section name.
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

// Validate checks LogConfig allowed level and format values.
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

// Validate checks APIConfig port, timeouts, and mode.
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

// Validate checks DatabaseConfig connectivity-related fields and ssl_mode.
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

// Validate checks KubernetesConfig timeouts and client rate limiter settings.
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

// Validate checks PrometheusConfig URL and timeout.
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

// Validate checks RedisConfig pool sizing, DB index, and TTL fields.
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

// Validate checks WorkerConfig intervals and retention bounds, and normalizes nil namespace slices to empty.
// YAML or JSON null for a list would otherwise leave nil slices. Empty slices make “no filter” logic predictable.
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

// BuildTLSConfig builds a TLS configuration for clients from TLSConfig.
// It optionally loads a custom root CA from RootCAPath and, when both ClientCertPath and ClientKeyPath are set,
// loads a mutual TLS client certificate. Setting only one of the client paths is ignored here, so configure both or neither.
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
