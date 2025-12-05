package scheduler

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/lan-dot-party/flowgauge/internal/api"
	"github.com/lan-dot-party/flowgauge/internal/speedtest"
	"github.com/lan-dot-party/flowgauge/internal/storage"
)

// SpeedtestJob runs speedtests on a schedule.
type SpeedtestJob struct {
	runner  *speedtest.MultiWANRunner
	storage storage.Storage
	logger  *zap.Logger
}

// NewSpeedtestJob creates a new speedtest job.
func NewSpeedtestJob(runner *speedtest.MultiWANRunner, store storage.Storage, logger *zap.Logger) *SpeedtestJob {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &SpeedtestJob{
		runner:  runner,
		storage: store,
		logger:  logger,
	}
}

// Run executes the speedtest job (implements cron.Job interface).
func (j *SpeedtestJob) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := j.RunWithContext(ctx); err != nil {
		j.logger.Error("Scheduled speedtest failed", zap.Error(err))
	}
}

// RunWithContext executes the speedtest job with a context.
func (j *SpeedtestJob) RunWithContext(ctx context.Context) error {
	startTime := time.Now()
	j.logger.Info("Starting scheduled speedtest")

	// Get all connections from runner
	connections := j.runner.GetConnections()
	j.logger.Info("Running speedtest for connections",
		zap.Int("count", len(connections)),
	)

	// Run speedtests
	results, err := j.runner.RunAll(ctx)
	if err != nil {
		return err
	}

	// Save results to storage and update Prometheus metrics
	var savedCount, errorCount int
	for _, result := range results {
		// Update Prometheus metrics
		api.UpdateMetricsForResult(&result)
		
		// Save to database
		dbResult := storage.FromSpeedtestResult(&result)
		
		if err := j.storage.SaveResult(ctx, dbResult); err != nil {
			j.logger.Error("Failed to save speedtest result",
				zap.String("connection", result.ConnectionName),
				zap.Error(err),
			)
			errorCount++
			continue
		}

		savedCount++

		if result.IsError() {
			j.logger.Warn("Speedtest completed with error",
				zap.String("connection", result.ConnectionName),
				zap.String("error", result.Error),
			)
		} else {
			j.logger.Info("Speedtest result saved",
				zap.String("connection", result.ConnectionName),
				zap.Float64("download_mbps", result.DownloadMbps),
				zap.Float64("upload_mbps", result.UploadMbps),
				zap.Float64("latency_ms", result.LatencyMs),
			)
		}
	}

	duration := time.Since(startTime)
	j.logger.Info("Scheduled speedtest completed",
		zap.Int("total", len(results)),
		zap.Int("saved", savedCount),
		zap.Int("errors", errorCount),
		zap.Duration("duration", duration),
	)

	return nil
}

