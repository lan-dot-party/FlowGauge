package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultConfigPaths defines the search order for configuration files.
var DefaultConfigPaths = []string{
	"/etc/flowgauge/config.yaml",
	"/etc/flowgauge/config.yml",
	"./config.yaml",
	"./config.yml",
	"./flowgauge.yaml",
	"./flowgauge.yml",
}

// Load reads and parses a configuration file from the given path.
// If path is empty, it searches DefaultConfigPaths.
// Environment variable FLOWGAUGE_CONFIG takes precedence over defaults.
func Load(path string) (*Config, error) {
	configPath, err := resolveConfigPath(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	// Apply defaults for missing values
	ApplyDefaults(cfg)

	// Validate the configuration
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// resolveConfigPath determines which config file to use.
// Priority: explicit path > FLOWGAUGE_CONFIG env > default paths
func resolveConfigPath(path string) (string, error) {
	// 1. Explicit path provided
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("config file not found: %s", path)
		}
		return path, nil
	}

	// 2. Environment variable
	if envPath := os.Getenv("FLOWGAUGE_CONFIG"); envPath != "" {
		if _, err := os.Stat(envPath); err != nil {
			return "", fmt.Errorf("config file from FLOWGAUGE_CONFIG not found: %s", envPath)
		}
		return envPath, nil
	}

	// 3. Search default paths
	for _, p := range DefaultConfigPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("no config file found (searched: %v)", DefaultConfigPaths)
}

// Validate checks the configuration for errors.
func Validate(cfg *Config) error {
	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[cfg.General.LogLevel] {
		return fmt.Errorf("invalid log_level: %q (must be debug, info, warn, or error)", cfg.General.LogLevel)
	}

	// Validate storage type
	validStorageTypes := map[string]bool{
		"sqlite":   true,
		"postgres": true,
	}
	if !validStorageTypes[cfg.Storage.Type] {
		return fmt.Errorf("invalid storage type: %q (must be sqlite or postgres)", cfg.Storage.Type)
	}

	// Validate SQLite path if using SQLite
	if cfg.Storage.Type == "sqlite" && cfg.Storage.SQLite.Path == "" {
		return fmt.Errorf("sqlite path is required when storage type is sqlite")
	}

	// Validate PostgreSQL config if using PostgreSQL
	if cfg.Storage.Type == "postgres" {
		if cfg.Storage.Postgres.Host == "" {
			return fmt.Errorf("postgres host is required when storage type is postgres")
		}
		if cfg.Storage.Postgres.Database == "" {
			return fmt.Errorf("postgres database is required when storage type is postgres")
		}
	}

	// Validate webserver listen address
	if cfg.Webserver.Enabled {
		if _, _, err := net.SplitHostPort(cfg.Webserver.Listen); err != nil {
			return fmt.Errorf("invalid webserver listen address %q: %w", cfg.Webserver.Listen, err)
		}
	}

	// Validate connections
	if len(cfg.Connections) == 0 {
		return fmt.Errorf("at least one connection must be configured")
	}

	connectionNames := make(map[string]bool)
	for i, conn := range cfg.Connections {
		if conn.Name == "" {
			return fmt.Errorf("connection[%d]: name is required", i)
		}
		if connectionNames[conn.Name] {
			return fmt.Errorf("connection[%d]: duplicate connection name %q", i, conn.Name)
		}
		connectionNames[conn.Name] = true

		// Validate DSCP value (0-63)
		if conn.DSCP < 0 || conn.DSCP > 63 {
			return fmt.Errorf("connection %q: DSCP value must be between 0 and 63, got %d", conn.Name, conn.DSCP)
		}

		// Validate source IP if provided
		if conn.SourceIP != "" {
			if ip := net.ParseIP(conn.SourceIP); ip == nil {
				return fmt.Errorf("connection %q: invalid source_ip %q", conn.Name, conn.SourceIP)
			}
		}
	}

	// Validate speedtest config
	validSizes := map[string]bool{
		"auto":   true,
		"small":  true,
		"medium": true,
		"large":  true,
		"":       true, // empty is allowed, defaults to auto
	}
	if !validSizes[cfg.Speedtest.DownloadSize] {
		return fmt.Errorf("invalid speedtest download_size: %q", cfg.Speedtest.DownloadSize)
	}
	if !validSizes[cfg.Speedtest.UploadSize] {
		return fmt.Errorf("invalid speedtest upload_size: %q", cfg.Speedtest.UploadSize)
	}

	return nil
}

// MustLoad is like Load but panics on error.
// Useful for initialization where config errors should be fatal.
func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

// WriteExample writes an example configuration to the given path.
func WriteExample(path string) error {
	cfg := NewDefault()

	// Add example connections
	cfg.Connections = []ConnectionConfig{
		{
			Name:     "WAN1-Primary",
			SourceIP: "192.168.1.100",
			DSCP:     0,
			Enabled:  true,
		},
		{
			Name:     "WAN2-Backup",
			SourceIP: "192.168.2.100",
			DSCP:     46,
			Enabled:  true,
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal example config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write example config: %w", err)
	}

	return nil
}

