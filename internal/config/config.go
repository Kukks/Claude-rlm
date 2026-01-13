package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for RLM
type Config struct {
	Orchestrator OrchestratorConfig `mapstructure:"orchestrator"`
	Storage      StorageConfig      `mapstructure:"storage"`
	Updater      UpdaterConfig      `mapstructure:"updater"`
	Logging      LoggingConfig      `mapstructure:"logging"`
}

// OrchestratorConfig holds orchestrator settings
type OrchestratorConfig struct {
	MaxRecursionDepth int           `mapstructure:"max_recursion_depth"`
	MaxIterations     int           `mapstructure:"max_iterations"`
	CacheEnabled      bool          `mapstructure:"cache_enabled"`
	CacheTTLHours     int           `mapstructure:"cache_ttl_hours"`
}

// StorageConfig holds storage settings
type StorageConfig struct {
	QdrantAddress  string `mapstructure:"qdrant_address"`
	QdrantEnabled  bool   `mapstructure:"qdrant_enabled"`
	CollectionName string `mapstructure:"collection_name"`
	JSONFallback   bool   `mapstructure:"json_fallback"`
	RAGDir         string `mapstructure:"rag_dir"`
}

// UpdaterConfig holds auto-updater settings
type UpdaterConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	AutoUpdate    bool   `mapstructure:"auto_update"`
	CheckInterval string `mapstructure:"check_interval"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Orchestrator: OrchestratorConfig{
			MaxRecursionDepth: 10,
			MaxIterations:     1000,
			CacheEnabled:      true,
			CacheTTLHours:     24,
		},
		Storage: StorageConfig{
			QdrantAddress:  "localhost:6334",
			QdrantEnabled:  true,
			CollectionName: "rlm_analyses",
			JSONFallback:   true,
			RAGDir:         ".rlm",
		},
		Updater: UpdaterConfig{
			Enabled:       true,
			AutoUpdate:    false,
			CheckInterval: "24h",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// Load loads configuration from file and environment
func Load() (*Config, error) {
	// Set defaults
	config := DefaultConfig()

	// Set up viper
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Look for config in ~/.config/rlm/
	home, err := os.UserHomeDir()
	if err == nil {
		viper.AddConfigPath(filepath.Join(home, ".config", "rlm"))
	}

	// Also look in current directory
	viper.AddConfigPath(".")

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		// Config file not found; use defaults
	}

	// Unmarshal into config struct
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}

// CacheTTL returns the cache TTL as a duration
func (c *OrchestratorConfig) CacheTTL() time.Duration {
	return time.Duration(c.CacheTTLHours) * time.Hour
}

// CheckIntervalDuration returns the check interval as a duration
func (c *UpdaterConfig) CheckIntervalDuration() time.Duration {
	duration, err := time.ParseDuration(c.CheckInterval)
	if err != nil {
		return 24 * time.Hour // Default to 24 hours
	}
	return duration
}
