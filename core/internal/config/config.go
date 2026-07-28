// Package config loads, validates, and exposes application settings for Mochi.
//
// Values are resolved in order: default values, optional YAML files listed in MOCHI_CONFIG_FILES
// (comma-separated paths, the first file is the base, each later file merged on top), then
// environment variables with the MOCHI_ prefix. Nested keys use underscores in env names
// (for example log.level becomes MOCHI_LOG_LEVEL).
//
// If MOCHI_CONFIG_FILES is unset or empty, no YAML is loaded. Only defaults and environment variables apply.
// The core/configs/config.yaml file in the repository is a reference layout until you point
// MOCHI_CONFIG_FILES at it (or another file).
package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// TLSConfig holds TLS settings for outbound clients (database, Prometheus, Redis, etc.).
type TLSConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
	RootCAPath         string `mapstructure:"root_ca_path"`
	ClientCertPath     string `mapstructure:"client_cert_path"`
	ClientKeyPath      string `mapstructure:"client_key_path"`
}

type Config struct {
	Log        LogConfig        `mapstructure:"log"`
	API        APIConfig        `mapstructure:"api"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Workers    WorkerConfig     `mapstructure:"workers"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type APIConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	Mode         string `mapstructure:"mode"`
}

func (c *APIConfig) ReadTimeoutDuration() time.Duration {
	return time.Duration(c.ReadTimeout) * time.Second
}

func (c *APIConfig) WriteTimeoutDuration() time.Duration {
	return time.Duration(c.WriteTimeout) * time.Second
}

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

func (c *DatabaseConfig) ConnMaxLifetimeDuration() time.Duration {
	return time.Duration(c.ConnMaxLifetime) * time.Second
}

func (c *DatabaseConfig) ConnMaxIdleTimeDuration() time.Duration {
	return time.Duration(c.ConnMaxIdleTime) * time.Second
}

type KubernetesConfig struct {
	KubeconfigPath string `mapstructure:"kubeconfig_path"`
	InCluster      bool   `mapstructure:"in_cluster"`
	ClusterName    string `mapstructure:"cluster_name"`
	RequestTimeout int    `mapstructure:"request_timeout"`
	QPS            int    `mapstructure:"qps"`
	Burst          int    `mapstructure:"burst"`
}

func (c *KubernetesConfig) RequestTimeoutDuration() time.Duration {
	return time.Duration(c.RequestTimeout) * time.Second
}

type PrometheusConfig struct {
	URL     string    `mapstructure:"url"`
	Timeout int       `mapstructure:"timeout"`
	TLS     TLSConfig `mapstructure:"tls"`
}

func (c *PrometheusConfig) TimeoutDuration() time.Duration {
	return time.Duration(c.Timeout) * time.Second
}

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
	TLS             TLSConfig `mapstructure:"tls"`
}

func (c *RedisConfig) ConnMaxLifetimeDuration() time.Duration {
	return time.Duration(c.ConnMaxLifetime) * time.Second
}

func (c *RedisConfig) ConnMaxIdleTimeDuration() time.Duration {
	return time.Duration(c.ConnMaxIdleTime) * time.Second
}

func (c *RedisConfig) CacheTTLDuration() time.Duration {
	return time.Duration(c.CacheTTL) * time.Second
}

type WorkerConfig struct {
	Sync      WorkerSyncConfig      `mapstructure:"sync"`
	Retention WorkerRetentionConfig `mapstructure:"retention"`
}

type WorkerSyncConfig struct {
	Interval          int      `mapstructure:"interval"`
	ExcludeNamespaces []string `mapstructure:"exclude_namespaces"`
	IncludeNamespaces []string `mapstructure:"include_namespaces"`
}

func (c *WorkerSyncConfig) IntervalDuration() time.Duration {
	return time.Duration(c.Interval) * time.Second
}

func (c *WorkerSyncConfig) ShouldSyncNamespace(namespace string) bool {
	if len(c.IncludeNamespaces) > 0 {
		return slices.Contains(c.IncludeNamespaces, namespace)
	}

	if len(c.ExcludeNamespaces) > 0 {
		return !slices.Contains(c.ExcludeNamespaces, namespace)
	}

	return true
}

type WorkerRetentionConfig struct {
	Interval int `mapstructure:"interval"`
	MaxAge   int `mapstructure:"max_age"`
}

func (c *WorkerRetentionConfig) IntervalDuration() time.Duration {
	return time.Duration(c.Interval) * time.Second
}

func (c *WorkerRetentionConfig) MaxAgeDuration() time.Duration {
	return time.Duration(c.MaxAge) * 24 * time.Hour
}

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
	viper.SetDefault("database.tls.enabled", false)
	viper.SetDefault("database.tls.insecure_skip_verify", false)
	viper.SetDefault("database.tls.root_ca_path", "")
	viper.SetDefault("database.tls.client_cert_path", "")
	viper.SetDefault("database.tls.client_key_path", "")

	viper.SetDefault("kubernetes.kubeconfig_path", "")
	viper.SetDefault("kubernetes.in_cluster", false)
	viper.SetDefault("kubernetes.cluster_name", "Kubernetes Cluster")
	viper.SetDefault("kubernetes.request_timeout", 30)
	viper.SetDefault("kubernetes.qps", 50)
	viper.SetDefault("kubernetes.burst", 100)

	viper.SetDefault("prometheus.url", "http://localhost:9090")
	viper.SetDefault("prometheus.timeout", 30)
	viper.SetDefault("prometheus.tls.enabled", false)
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
	viper.SetDefault("redis.tls.enabled", false)
	viper.SetDefault("redis.tls.insecure_skip_verify", false)
	viper.SetDefault("redis.tls.root_ca_path", "")
	viper.SetDefault("redis.tls.client_cert_path", "")
	viper.SetDefault("redis.tls.client_key_path", "")

	viper.SetDefault("workers.sync.interval", 120)
	viper.SetDefault("workers.sync.exclude_namespaces", []string{"default", "kube-system", "kube-public", "kube-node-lease"})
	viper.SetDefault("workers.sync.include_namespaces", []string{})
	viper.SetDefault("workers.retention.interval", 86400)
	viper.SetDefault("workers.retention.max_age", 90)
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
	if c.TLS.Enabled && c.SSLMode == "disable" {
		return fmt.Errorf("tls.enabled requires ssl_mode other than disable")
	}
	if !c.TLS.Enabled && c.SSLMode != "disable" {
		return fmt.Errorf("ssl_mode %q requires tls.enabled", c.SSLMode)
	}
	return nil
}

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
	if c.ClusterName == "" {
		return fmt.Errorf("cluster_name cannot be empty")
	}
	return nil
}

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

func (c *WorkerConfig) Validate() error {
	if err := c.Sync.Validate(); err != nil {
		return err
	}
	if err := c.Retention.Validate(); err != nil {
		return err
	}
	return nil
}

func (c *WorkerSyncConfig) Validate() error {
	if c.Interval <= 0 {
		return fmt.Errorf("workers.sync.interval must be greater than 0, got: %d", c.Interval)
	}
	if c.ExcludeNamespaces == nil {
		c.ExcludeNamespaces = []string{}
	}
	if c.IncludeNamespaces == nil {
		c.IncludeNamespaces = []string{}
	}
	return nil
}

func (c *WorkerRetentionConfig) Validate() error {
	if c.Interval <= 0 {
		return fmt.Errorf("workers.retention.interval must be greater than 0, got: %d", c.Interval)
	}
	if c.MaxAge <= 0 {
		return fmt.Errorf("workers.retention.max_age must be at least 1 day, got: %d", c.MaxAge)
	}
	if c.MaxAge > 3650 {
		return fmt.Errorf("workers.retention.max_age must be at most 3650 days (about 10 years), got: %d", c.MaxAge)
	}
	return nil
}

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
