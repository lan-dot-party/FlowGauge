// Package storage provides database storage for speedtest results.
package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/lan-dot-party/flowgauge/internal/config"
)

// Storage defines the interface for storing and retrieving speedtest results.
type Storage interface {
	// Lifecycle
	Init(ctx context.Context) error
	Close() error

	// Results
	SaveResult(ctx context.Context, result *TestResult) error
	GetResult(ctx context.Context, id int64) (*TestResult, error)
	GetResults(ctx context.Context, filter ResultFilter) ([]TestResult, error)
	GetLatestResults(ctx context.Context) ([]TestResult, error)

	// Stats
	GetStats(ctx context.Context, connectionName string, period time.Duration) (*Stats, error)

	// Cleanup
	DeleteOldResults(ctx context.Context, olderThan time.Time) (int64, error)
}

// ResultFilter defines criteria for filtering results.
type ResultFilter struct {
	ConnectionName string
	Since          time.Time
	Until          time.Time
	Limit          int
	Offset         int
}

// Stats contains aggregated statistics for a connection.
type Stats struct {
	ConnectionName string        `json:"connection_name"`
	AvgDownload    float64       `json:"avg_download_mbps"`
	AvgUpload      float64       `json:"avg_upload_mbps"`
	AvgLatency     float64       `json:"avg_latency_ms"`
	MinDownload    float64       `json:"min_download_mbps"`
	MaxDownload    float64       `json:"max_download_mbps"`
	MinUpload      float64       `json:"min_upload_mbps"`
	MaxUpload      float64       `json:"max_upload_mbps"`
	MinLatency     float64       `json:"min_latency_ms"`
	MaxLatency     float64       `json:"max_latency_ms"`
	TestCount      int           `json:"test_count"`
	ErrorCount     int           `json:"error_count"`
	Period         time.Duration `json:"period"`
	Since          time.Time     `json:"since"`
	Until          time.Time     `json:"until"`
}

// NewStorage creates a new Storage instance based on the configuration.
func NewStorage(cfg config.StorageConfig) (Storage, error) {
	switch cfg.Type {
	case "sqlite":
		return NewSQLiteStorage(cfg.SQLite)
	case "postgres":
		return NewPostgresStorage(cfg.Postgres)
	default:
		return nil, fmt.Errorf("unknown storage type: %s", cfg.Type)
	}
}

