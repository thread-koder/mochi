package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Holds all application configuration
type Config struct {
	Log LogConfig `mapstructure:"log"`
}

// Holds logger configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // json, console
}

var AppConfig *Config

// Reads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Add config paths
	if configPath != "" {
		viper.AddConfigPath(configPath)
	}
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/mochi")
	viper.AddConfigPath(".")

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
			return nil, fmt.Errorf("config file at path %s not found", configPath)
		}
		// Config file not found is OK, use defaults and env vars
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	AppConfig = &config
	return &config, nil
}

func setDefaults() {
	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "console")
}
