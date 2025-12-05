package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/lan-dot-party/flowgauge/internal/config"
)

// SQLiteStorage implements the Storage interface using SQLite.
type SQLiteStorage struct {
	db   *sql.DB
	path string
}

// NewSQLiteStorage creates a new SQLite storage instance.
func NewSQLiteStorage(cfg config.SQLiteConfig) (*SQLiteStorage, error) {
	return &SQLiteStorage{
		path: cfg.Path,
	}, nil
}

// Init initializes the SQLite database connection and schema.
func (s *SQLiteStorage) Init(ctx context.Context) error {
	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", s.path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	s.db = db

	// Enable WAL mode for better concurrency
	if _, err := s.db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := s.db.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create schema
	if err := s.createSchema(ctx); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// createSchema creates the database tables if they don't exist.
func (s *SQLiteStorage) createSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS test_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		connection_name TEXT NOT NULL,
		server_id INTEGER,
		server_name TEXT,
		server_country TEXT,
		server_host TEXT,
		latency_ms REAL,
		jitter_ms REAL,
		download_mbps REAL,
		upload_mbps REAL,
		packet_loss_pct REAL,
		source_ip TEXT,
		dscp INTEGER,
		error TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_results_connection ON test_results(connection_name);
	CREATE INDEX IF NOT EXISTS idx_results_created ON test_results(created_at);
	CREATE INDEX IF NOT EXISTS idx_results_connection_created ON test_results(connection_name, created_at);
	`

	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// Close closes the database connection.
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SaveResult saves a speedtest result to the database.
func (s *SQLiteStorage) SaveResult(ctx context.Context, result *TestResult) error {
	query := `
	INSERT INTO test_results (
		connection_name, server_id, server_name, server_country, server_host,
		latency_ms, jitter_ms, download_mbps, upload_mbps, packet_loss_pct,
		source_ip, dscp, error, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	res, err := s.db.ExecContext(ctx, query,
		result.ConnectionName,
		result.ServerID,
		result.ServerName,
		result.ServerCountry,
		result.ServerHost,
		result.LatencyMs,
		result.JitterMs,
		result.DownloadMbps,
		result.UploadMbps,
		result.PacketLossPct,
		result.SourceIP,
		result.DSCP,
		result.Error,
		result.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert result: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}
	result.ID = id

	return nil
}

// GetResult retrieves a single result by ID.
func (s *SQLiteStorage) GetResult(ctx context.Context, id int64) (*TestResult, error) {
	query := `
	SELECT id, connection_name, server_id, server_name, server_country, server_host,
		   latency_ms, jitter_ms, download_mbps, upload_mbps, packet_loss_pct,
		   source_ip, dscp, error, created_at
	FROM test_results
	WHERE id = ?
	`

	result := &TestResult{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&result.ID,
		&result.ConnectionName,
		&result.ServerID,
		&result.ServerName,
		&result.ServerCountry,
		&result.ServerHost,
		&result.LatencyMs,
		&result.JitterMs,
		&result.DownloadMbps,
		&result.UploadMbps,
		&result.PacketLossPct,
		&result.SourceIP,
		&result.DSCP,
		&result.Error,
		&result.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("result not found: %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get result: %w", err)
	}

	return result, nil
}

// GetResults retrieves results based on filter criteria.
func (s *SQLiteStorage) GetResults(ctx context.Context, filter ResultFilter) ([]TestResult, error) {
	query := `
	SELECT id, connection_name, server_id, server_name, server_country, server_host,
		   latency_ms, jitter_ms, download_mbps, upload_mbps, packet_loss_pct,
		   source_ip, dscp, error, created_at
	FROM test_results
	WHERE 1=1
	`
	args := []interface{}{}

	if filter.ConnectionName != "" {
		query += " AND connection_name = ?"
		args = append(args, filter.ConnectionName)
	}

	if !filter.Since.IsZero() {
		query += " AND created_at >= ?"
		args = append(args, filter.Since)
	}

	if !filter.Until.IsZero() {
		query += " AND created_at <= ?"
		args = append(args, filter.Until)
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []TestResult
	for rows.Next() {
		var r TestResult
		err := rows.Scan(
			&r.ID,
			&r.ConnectionName,
			&r.ServerID,
			&r.ServerName,
			&r.ServerCountry,
			&r.ServerHost,
			&r.LatencyMs,
			&r.JitterMs,
			&r.DownloadMbps,
			&r.UploadMbps,
			&r.PacketLossPct,
			&r.SourceIP,
			&r.DSCP,
			&r.Error,
			&r.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return results, nil
}

// GetLatestResults retrieves the most recent result for each connection.
func (s *SQLiteStorage) GetLatestResults(ctx context.Context) ([]TestResult, error) {
	query := `
	SELECT t.id, t.connection_name, t.server_id, t.server_name, t.server_country, t.server_host,
		   t.latency_ms, t.jitter_ms, t.download_mbps, t.upload_mbps, t.packet_loss_pct,
		   t.source_ip, t.dscp, t.error, t.created_at
	FROM test_results t
	INNER JOIN (
		SELECT connection_name, MAX(created_at) as max_created
		FROM test_results
		GROUP BY connection_name
	) latest ON t.connection_name = latest.connection_name AND t.created_at = latest.max_created
	ORDER BY t.connection_name
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest results: %w", err)
	}
	defer rows.Close()

	var results []TestResult
	for rows.Next() {
		var r TestResult
		err := rows.Scan(
			&r.ID,
			&r.ConnectionName,
			&r.ServerID,
			&r.ServerName,
			&r.ServerCountry,
			&r.ServerHost,
			&r.LatencyMs,
			&r.JitterMs,
			&r.DownloadMbps,
			&r.UploadMbps,
			&r.PacketLossPct,
			&r.SourceIP,
			&r.DSCP,
			&r.Error,
			&r.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		results = append(results, r)
	}

	return results, nil
}

// GetStats calculates statistics for a connection over a time period.
func (s *SQLiteStorage) GetStats(ctx context.Context, connectionName string, period time.Duration) (*Stats, error) {
	since := time.Now().Add(-period)
	until := time.Now()

	query := `
	SELECT 
		COUNT(*) as test_count,
		COUNT(CASE WHEN error != '' THEN 1 END) as error_count,
		AVG(CASE WHEN error = '' THEN download_mbps END) as avg_download,
		AVG(CASE WHEN error = '' THEN upload_mbps END) as avg_upload,
		AVG(CASE WHEN error = '' THEN latency_ms END) as avg_latency,
		MIN(CASE WHEN error = '' THEN download_mbps END) as min_download,
		MAX(CASE WHEN error = '' THEN download_mbps END) as max_download,
		MIN(CASE WHEN error = '' THEN upload_mbps END) as min_upload,
		MAX(CASE WHEN error = '' THEN upload_mbps END) as max_upload,
		MIN(CASE WHEN error = '' THEN latency_ms END) as min_latency,
		MAX(CASE WHEN error = '' THEN latency_ms END) as max_latency
	FROM test_results
	WHERE connection_name = ? AND created_at >= ? AND created_at <= ?
	`

	stats := &Stats{
		ConnectionName: connectionName,
		Period:         period,
		Since:          since,
		Until:          until,
	}

	var avgDownload, avgUpload, avgLatency sql.NullFloat64
	var minDownload, maxDownload, minUpload, maxUpload, minLatency, maxLatency sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, connectionName, since, until).Scan(
		&stats.TestCount,
		&stats.ErrorCount,
		&avgDownload,
		&avgUpload,
		&avgLatency,
		&minDownload,
		&maxDownload,
		&minUpload,
		&maxUpload,
		&minLatency,
		&maxLatency,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	if avgDownload.Valid {
		stats.AvgDownload = avgDownload.Float64
	}
	if avgUpload.Valid {
		stats.AvgUpload = avgUpload.Float64
	}
	if avgLatency.Valid {
		stats.AvgLatency = avgLatency.Float64
	}
	if minDownload.Valid {
		stats.MinDownload = minDownload.Float64
	}
	if maxDownload.Valid {
		stats.MaxDownload = maxDownload.Float64
	}
	if minUpload.Valid {
		stats.MinUpload = minUpload.Float64
	}
	if maxUpload.Valid {
		stats.MaxUpload = maxUpload.Float64
	}
	if minLatency.Valid {
		stats.MinLatency = minLatency.Float64
	}
	if maxLatency.Valid {
		stats.MaxLatency = maxLatency.Float64
	}

	return stats, nil
}

// DeleteOldResults removes results older than the specified time.
func (s *SQLiteStorage) DeleteOldResults(ctx context.Context, olderThan time.Time) (int64, error) {
	query := "DELETE FROM test_results WHERE created_at < ?"

	result, err := s.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old results: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return count, nil
}

