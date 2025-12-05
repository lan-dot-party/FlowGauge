package config

import "time"

// Default values for configuration
const (
	DefaultLogLevel         = "info"
	DefaultDataDir          = "/var/lib/flowgauge"
	DefaultStorageType      = "sqlite"
	DefaultSQLitePath       = "/var/lib/flowgauge/results.db"
	DefaultWebserverListen  = "127.0.0.1:8080"
	DefaultSchedule         = "0 * * * *" // Every hour
	DefaultTestTimeout      = 60 * time.Second
	DefaultDownloadSize     = "auto"
	DefaultUploadSize       = "auto"
	DefaultPostgresPort     = 5432
	DefaultPostgresSSL      = "disable"
)

// NewDefault creates a new Config with all default values applied.
func NewDefault() *Config {
	return &Config{
		General: GeneralConfig{
			LogLevel: DefaultLogLevel,
			DataDir:  DefaultDataDir,
		},
		Storage: StorageConfig{
			Type: DefaultStorageType,
			SQLite: SQLiteConfig{
				Path: DefaultSQLitePath,
			},
			Postgres: PostgresConfig{
				Port:    DefaultPostgresPort,
				SSLMode: DefaultPostgresSSL,
			},
		},
		Webserver: WebserverConfig{
			Enabled: true,
			Listen:  DefaultWebserverListen,
		},
		Connections: []ConnectionConfig{},
		Scheduler: SchedulerConfig{
			Enabled:  false,
			Schedule: DefaultSchedule,
		},
		Speedtest: SpeedtestConfig{
			ServerIDs:    []int{},
			Timeout:      DefaultTestTimeout,
			DownloadSize: DefaultDownloadSize,
			UploadSize:   DefaultUploadSize,
		},
	}
}

// ApplyDefaults fills in default values for any unset configuration options.
func ApplyDefaults(cfg *Config) {
	// General defaults
	if cfg.General.LogLevel == "" {
		cfg.General.LogLevel = DefaultLogLevel
	}
	if cfg.General.DataDir == "" {
		cfg.General.DataDir = DefaultDataDir
	}

	// Storage defaults
	if cfg.Storage.Type == "" {
		cfg.Storage.Type = DefaultStorageType
	}
	if cfg.Storage.Type == "sqlite" && cfg.Storage.SQLite.Path == "" {
		cfg.Storage.SQLite.Path = DefaultSQLitePath
	}
	if cfg.Storage.Postgres.Port == 0 {
		cfg.Storage.Postgres.Port = DefaultPostgresPort
	}
	if cfg.Storage.Postgres.SSLMode == "" {
		cfg.Storage.Postgres.SSLMode = DefaultPostgresSSL
	}

	// Webserver defaults
	if cfg.Webserver.Listen == "" {
		cfg.Webserver.Listen = DefaultWebserverListen
	}

	// Scheduler defaults
	if cfg.Scheduler.Schedule == "" {
		cfg.Scheduler.Schedule = DefaultSchedule
	}

	// Speedtest defaults
	if cfg.Speedtest.Timeout == 0 {
		cfg.Speedtest.Timeout = DefaultTestTimeout
	}
	if cfg.Speedtest.DownloadSize == "" {
		cfg.Speedtest.DownloadSize = DefaultDownloadSize
	}
	if cfg.Speedtest.UploadSize == "" {
		cfg.Speedtest.UploadSize = DefaultUploadSize
	}
	if cfg.Speedtest.ServerIDs == nil {
		cfg.Speedtest.ServerIDs = []int{}
	}

	// Note: YAML unmarshal sets bool to false by default for connections,
	// so we can't distinguish between "enabled: false" and unset.
	// Users must explicitly set "enabled: true" for active connections.
}

// GetEnabledConnections returns only the connections that are enabled.
func (c *Config) GetEnabledConnections() []ConnectionConfig {
	var enabled []ConnectionConfig
	for _, conn := range c.Connections {
		if conn.Enabled {
			enabled = append(enabled, conn)
		}
	}
	return enabled
}

// GetConnectionByName returns a connection by its name, or nil if not found.
func (c *Config) GetConnectionByName(name string) *ConnectionConfig {
	for i := range c.Connections {
		if c.Connections[i].Name == name {
			return &c.Connections[i]
		}
	}
	return nil
}

