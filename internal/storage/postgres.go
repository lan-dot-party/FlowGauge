package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/lan-dot-party/flowgauge/internal/config"
)

// PostgresStorage implements the Storage interface using PostgreSQL.
type PostgresStorage struct {
	db  *sql.DB
	cfg config.PostgresConfig
}

// NewPostgresStorage creates a new PostgreSQL storage instance.
func NewPostgresStorage(cfg config.PostgresConfig) (*PostgresStorage, error) {
	return &PostgresStorage{
		cfg: cfg,
	}, nil
}

// buildDSN creates the PostgreSQL connection string.
func (s *PostgresStorage) buildDSN() string {
	// Build connection string for pgx
	dsn := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s sslmode=%s",
		s.cfg.Host,
		s.cfg.Port,
		s.cfg.Database,
		s.cfg.User,
		s.cfg.SSLMode,
	)

	if s.cfg.Password != "" {
		dsn += fmt.Sprintf(" password=%s", s.cfg.Password)
	}

	return dsn
}

// Init initializes the PostgreSQL database connection and schema.
func (s *PostgresStorage) Init(ctx context.Context) error {
	dsn := s.buildDSN()

	// Open database connection using pgx stdlib driver
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	s.db = db

	// Configure connection pool
	s.db.SetMaxOpenConns(25)
	s.db.SetMaxIdleConns(5)
	s.db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Create schema
	if err := s.createSchema(ctx); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// createSchema creates the database tables if they don't exist.
func (s *PostgresStorage) createSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS test_results (
		id BIGSERIAL PRIMARY KEY,
		connection_name TEXT NOT NULL,
		server_id INTEGER,
		server_name TEXT,
		server_country TEXT,
		server_host TEXT,
		latency_ms DOUBLE PRECISION,
		jitter_ms DOUBLE PRECISION,
		download_mbps DOUBLE PRECISION,
		upload_mbps DOUBLE PRECISION,
		packet_loss_pct DOUBLE PRECISION,
		source_ip TEXT,
		dscp INTEGER,
		error TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_results_connection ON test_results(connection_name);
	CREATE INDEX IF NOT EXISTS idx_results_created ON test_results(created_at);
	CREATE INDEX IF NOT EXISTS idx_results_connection_created ON test_results(connection_name, created_at);
	`

	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// Close closes the database connection.
func (s *PostgresStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SaveResult saves a speedtest result to the database.
func (s *PostgresStorage) SaveResult(ctx context.Context, result *TestResult) error {
	query := `
	INSERT INTO test_results (
		connection_name, server_id, server_name, server_country, server_host,
		latency_ms, jitter_ms, download_mbps, upload_mbps, packet_loss_pct,
		source_ip, dscp, error, created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	RETURNING id
	`

	err := s.db.QueryRowContext(ctx, query,
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
	).Scan(&result.ID)

	if err != nil {
		return fmt.Errorf("failed to insert result: %w", err)
	}

	return nil
}

// GetResult retrieves a single result by ID.
func (s *PostgresStorage) GetResult(ctx context.Context, id int64) (*TestResult, error) {
	query := `
	SELECT id, connection_name, server_id, server_name, server_country, server_host,
		   latency_ms, jitter_ms, download_mbps, upload_mbps, packet_loss_pct,
		   source_ip, dscp, error, created_at
	FROM test_results
	WHERE id = $1
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
func (s *PostgresStorage) GetResults(ctx context.Context, filter ResultFilter) ([]TestResult, error) {
	query := `
	SELECT id, connection_name, server_id, server_name, server_country, server_host,
		   latency_ms, jitter_ms, download_mbps, upload_mbps, packet_loss_pct,
		   source_ip, dscp, error, created_at
	FROM test_results
	WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if filter.ConnectionName != "" {
		query += fmt.Sprintf(" AND connection_name = $%d", argNum)
		args = append(args, filter.ConnectionName)
		argNum++
	}

	if !filter.Since.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d", argNum)
		args = append(args, filter.Since)
		argNum++
	}

	if !filter.Until.IsZero() {
		query += fmt.Sprintf(" AND created_at <= $%d", argNum)
		args = append(args, filter.Until)
		argNum++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, filter.Limit)
		argNum++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query results: %w", err)
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return results, nil
}

// GetLatestResults retrieves the most recent result for each connection.
func (s *PostgresStorage) GetLatestResults(ctx context.Context) ([]TestResult, error) {
	// PostgreSQL DISTINCT ON is more efficient than self-join
	query := `
	SELECT DISTINCT ON (connection_name)
		id, connection_name, server_id, server_name, server_country, server_host,
		latency_ms, jitter_ms, download_mbps, upload_mbps, packet_loss_pct,
		source_ip, dscp, error, created_at
	FROM test_results
	ORDER BY connection_name, created_at DESC
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
func (s *PostgresStorage) GetStats(ctx context.Context, connectionName string, period time.Duration) (*Stats, error) {
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
	WHERE connection_name = $1 AND created_at >= $2 AND created_at <= $3
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
func (s *PostgresStorage) DeleteOldResults(ctx context.Context, olderThan time.Time) (int64, error) {
	query := "DELETE FROM test_results WHERE created_at < $1"

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

