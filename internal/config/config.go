// Package config provides configuration structures and loading for FlowGauge.
package config

import "time"

// Config is the main configuration structure for FlowGauge.
type Config struct {
	General     GeneralConfig      `yaml:"general"`
	Storage     StorageConfig      `yaml:"storage"`
	Webserver   WebserverConfig    `yaml:"webserver"`
	Connections []ConnectionConfig `yaml:"connections"`
	Scheduler   SchedulerConfig    `yaml:"scheduler"`
	Speedtest   SpeedtestConfig    `yaml:"speedtest"`
}

// GeneralConfig contains general application settings.
type GeneralConfig struct {
	// LogLevel sets the logging verbosity: debug, info, warn, error
	LogLevel string `yaml:"log_level"`
	// DataDir is the directory for storing application data
	DataDir string `yaml:"data_dir"`
}

// StorageConfig defines the storage backend settings.
type StorageConfig struct {
	// Type is the storage backend: sqlite or postgres
	Type     string         `yaml:"type"`
	SQLite   SQLiteConfig   `yaml:"sqlite"`
	Postgres PostgresConfig `yaml:"postgres"`
}

// SQLiteConfig contains SQLite-specific settings.
type SQLiteConfig struct {
	// Path is the file path for the SQLite database
	Path string `yaml:"path"`
}

// PostgresConfig contains PostgreSQL-specific settings.
type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"ssl_mode"`
}

// WebserverConfig defines the web server settings (Dashboard + API).
type WebserverConfig struct {
	// Enabled controls whether the web server is started
	Enabled bool `yaml:"enabled"`
	// Listen is the address and port to bind to (e.g., "0.0.0.0:8080")
	Listen string `yaml:"listen"`
	// Auth contains optional authentication settings
	Auth *AuthConfig `yaml:"auth,omitempty"`
}

// AuthConfig contains optional Basic Auth settings for the API.
type AuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// ConnectionConfig defines a network connection to test.
type ConnectionConfig struct {
	// Name is the display name for this connection
	Name string `yaml:"name"`
	// SourceIP is the local IP address to bind to for this test
	SourceIP string `yaml:"source_ip"`
	// DSCP is the Differentiated Services Code Point value (0-63)
	DSCP int `yaml:"dscp"`
	// Enabled controls whether this connection is tested
	Enabled bool `yaml:"enabled"`
}

// SchedulerConfig defines the automatic test scheduling.
type SchedulerConfig struct {
	// Enabled controls whether scheduled tests run automatically
	Enabled bool `yaml:"enabled"`
	// Schedule is a cron expression (e.g., "*/30 * * * *" for every 30 minutes)
	Schedule string `yaml:"schedule"`
}

// SpeedtestConfig contains speedtest-specific settings.
type SpeedtestConfig struct {
	// ServerIDs is a list of specific speedtest server IDs to use (empty = auto-select)
	ServerIDs []int `yaml:"server_ids"`
	// Timeout is the maximum duration for a single test
	Timeout time.Duration `yaml:"timeout"`
	// DownloadSize controls the download test size: auto, small, medium, large
	DownloadSize string `yaml:"download_size"`
	// UploadSize controls the upload test size: auto, small, medium, large
	UploadSize string `yaml:"upload_size"`
}

// DSCPValue represents common DSCP values for QoS marking.
const (
	DSCPBestEffort = 0  // BE - Default/Best Effort
	DSCPEF         = 46 // EF - Expedited Forwarding (voice)
	DSCPAF11       = 10 // AF11 - Assured Forwarding Class 1, Low Drop
	DSCPAF12       = 12 // AF12 - Assured Forwarding Class 1, Medium Drop
	DSCPAF13       = 14 // AF13 - Assured Forwarding Class 1, High Drop
	DSCPAF21       = 18 // AF21 - Assured Forwarding Class 2, Low Drop
	DSCPAF22       = 20 // AF22 - Assured Forwarding Class 2, Medium Drop
	DSCPAF23       = 22 // AF23 - Assured Forwarding Class 2, High Drop
	DSCPAF31       = 26 // AF31 - Assured Forwarding Class 3, Low Drop
	DSCPAF32       = 28 // AF32 - Assured Forwarding Class 3, Medium Drop
	DSCPAF33       = 30 // AF33 - Assured Forwarding Class 3, High Drop
	DSCPAF41       = 34 // AF41 - Assured Forwarding Class 4, Low Drop
	DSCPAF42       = 36 // AF42 - Assured Forwarding Class 4, Medium Drop
	DSCPAF43       = 38 // AF43 - Assured Forwarding Class 4, High Drop
	DSCPCS1        = 8  // CS1 - Class Selector 1 (scavenger)
	DSCPCS2        = 16 // CS2 - Class Selector 2
	DSCPCS3        = 24 // CS3 - Class Selector 3
	DSCPCS4        = 32 // CS4 - Class Selector 4
	DSCPCS5        = 40 // CS5 - Class Selector 5 (signaling)
	DSCPCS6        = 48 // CS6 - Class Selector 6 (network control)
	DSCPCS7        = 56 // CS7 - Class Selector 7
)

// DSCPNames maps DSCP values to their common names.
var DSCPNames = map[int]string{
	DSCPBestEffort: "BE (Best Effort)",
	DSCPEF:         "EF (Expedited Forwarding)",
	DSCPAF11:       "AF11",
	DSCPAF12:       "AF12",
	DSCPAF13:       "AF13",
	DSCPAF21:       "AF21",
	DSCPAF22:       "AF22",
	DSCPAF23:       "AF23",
	DSCPAF31:       "AF31",
	DSCPAF32:       "AF32",
	DSCPAF33:       "AF33",
	DSCPAF41:       "AF41",
	DSCPAF42:       "AF42",
	DSCPAF43:       "AF43",
	DSCPCS1:        "CS1 (Scavenger)",
	DSCPCS2:        "CS2",
	DSCPCS3:        "CS3",
	DSCPCS4:        "CS4",
	DSCPCS5:        "CS5 (Signaling)",
	DSCPCS6:        "CS6 (Network Control)",
	DSCPCS7:        "CS7",
}

