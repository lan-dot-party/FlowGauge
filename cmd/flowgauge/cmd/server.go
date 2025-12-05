package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/lan-dot-party/flowgauge/internal/api"
	"github.com/lan-dot-party/flowgauge/internal/logger"
	"github.com/lan-dot-party/flowgauge/internal/scheduler"
	"github.com/lan-dot-party/flowgauge/internal/speedtest"
	"github.com/lan-dot-party/flowgauge/internal/storage"
)

var (
	noScheduler bool
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the web server (Dashboard + API)",
	Long: `Start the FlowGauge web server with optional scheduler.

The server provides:
  • Web Dashboard for visualizing results
  • REST API for querying results
  • Prometheus metrics endpoint (/api/v1/metrics)
  • Optional scheduled speedtests

Examples:
  # Start server with scheduler (if enabled in config)
  flowgauge server

  # Start server without scheduler
  flowgauge server --no-scheduler`,
	RunE: runServer,
}

func runServer(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	if !cfg.Webserver.Enabled {
		return fmt.Errorf("webserver is disabled in configuration (set webserver.enabled: true)")
	}

	// Initialize storage
	store, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}
	if err := store.Init(context.Background()); err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer store.Close()

	// Create speedtest runner
	var runner *speedtest.MultiWANRunner
	connections := cfg.GetEnabledConnections()
	if len(connections) > 0 {
		runner, err = speedtest.NewMultiWANRunner(connections, &cfg.Speedtest, logger.Log)
		if err != nil {
			logger.Warn("Failed to create speedtest runner", zap.Error(err))
		}
	}

	// Create web server
	server, err := api.NewServer(cfg, store, runner, logger.Log)
	if err != nil {
		return fmt.Errorf("failed to create web server: %w", err)
	}

	// Initialize Prometheus metrics from stored results
	initPrometheusMetrics(context.Background(), store)

	// Create scheduler if enabled
	var sched *scheduler.Scheduler
	schedulerEnabled := cfg.Scheduler.Enabled && !noScheduler && runner != nil
	if schedulerEnabled {
		sched, err = scheduler.NewScheduler(&cfg.Scheduler, runner, store, logger.Log)
		if err != nil {
			logger.Warn("Failed to create scheduler", zap.Error(err))
			schedulerEnabled = false
		}
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("Received signal, shutting down", zap.String("signal", sig.String()))
		cancel()

		// Stop scheduler first
		if sched != nil {
			sched.Stop()
		}

		// Give server time to shutdown gracefully
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("Server shutdown error", zap.Error(err))
		}
	}()

	// Print startup info
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════╗")
	fmt.Println("║       FlowGauge Web Server                ║")
	fmt.Println("╚═══════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  Listen:      http://%s\n", cfg.Webserver.Listen)
	fmt.Printf("  Storage:     %s\n", cfg.Storage.Type)
	fmt.Printf("  Connections: %d configured\n", len(cfg.Connections))
	if cfg.Webserver.Auth != nil && cfg.Webserver.Auth.Username != "" {
		fmt.Printf("  Auth:        Basic Auth enabled\n")
	} else {
		fmt.Printf("  Auth:        None\n")
	}

	// Start scheduler if enabled
	if schedulerEnabled && sched != nil {
		if err := sched.Start(); err != nil {
			logger.Error("Failed to start scheduler", zap.Error(err))
		} else {
			fmt.Printf("  Scheduler:   ✅ enabled (%s)\n", cfg.Scheduler.Schedule)
			fmt.Printf("  Next run:    %s\n", sched.NextRun())
		}
	} else {
		fmt.Printf("  Scheduler:   disabled\n")
	}

	fmt.Println()
	fmt.Println("  Dashboard:")
	fmt.Println("    GET  /                    - Web Dashboard")
	fmt.Println()
	fmt.Println("  API Endpoints (Read-Only):")
	fmt.Println("    GET  /api/                - API Documentation")
	fmt.Println("    GET  /health              - Health check")
	fmt.Println("    GET  /api/v1/results      - List results")
	fmt.Println("    GET  /api/v1/results/latest - Latest results")
	fmt.Println("    GET  /api/v1/connections  - List connections")
	fmt.Println("    GET  /api/v1/connections/{name}/stats - Connection stats")
	fmt.Println("    GET  /api/v1/metrics      - Prometheus metrics")
	fmt.Println()
	fmt.Println("  Press Ctrl+C to stop")
	fmt.Println()

	// Start server (blocks until shutdown)
	if err := server.Start(); err != nil {
		// Check if we're shutting down
		select {
		case <-ctx.Done():
			return nil
		default:
			return fmt.Errorf("server error: %w", err)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().BoolVar(&noScheduler, "no-scheduler", false,
		"disable scheduler even if enabled in config")
}

// initPrometheusMetrics loads latest results from storage and initializes Prometheus metrics.
func initPrometheusMetrics(ctx context.Context, store storage.Storage) {
	// Load latest results for each connection
	results, err := store.GetLatestResults(ctx)
	if err != nil {
		logger.Warn("Failed to load results for Prometheus metrics initialization", zap.Error(err))
		return
	}

	if len(results) == 0 {
		logger.Debug("No stored results to initialize Prometheus metrics")
		return
	}

	// Convert storage.TestResult to speedtest.Result and update metrics
	for _, dbResult := range results {
		result := dbResult.ToSpeedtestResult()
		api.UpdateMetricsForResult(result)
	}

	logger.Info("Prometheus metrics initialized from stored results",
		zap.Int("connections", len(results)),
	)
}
