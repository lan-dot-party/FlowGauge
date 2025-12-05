package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/lan-dot-party/flowgauge/internal/logger"
	"github.com/lan-dot-party/flowgauge/internal/speedtest"
	"github.com/lan-dot-party/flowgauge/internal/storage"
)

var (
	testConnection string
	testOnce       bool
	testJSON       bool
	testNoSave     bool
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run a speedtest",
	Long: `Run a speedtest for one or all configured connections.

Examples:
  # Test all enabled connections
  flowgauge test

  # Test a specific connection
  flowgauge test --connection WAN1

  # Output results as JSON
  flowgauge test --json
  
  # Run test without saving to database
  flowgauge test --no-save`,
	RunE: runTest,
}

func runTest(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Get connections to test
	connections := cfg.GetEnabledConnections()
	if len(connections) == 0 {
		return fmt.Errorf("no enabled connections found in configuration")
	}

	// Filter to specific connection if requested
	if testConnection != "" {
		conn := cfg.GetConnectionByName(testConnection)
		if conn == nil {
			return fmt.Errorf("connection %q not found", testConnection)
		}
		if !conn.Enabled {
			return fmt.Errorf("connection %q is disabled", testConnection)
		}
		connections = connections[:0]
		connections = append(connections, *conn)
	}

	// Create Multi-WAN runner
	runner, err := speedtest.NewMultiWANRunner(connections, &cfg.Speedtest, logger.Log)
	if err != nil {
		return fmt.Errorf("failed to create speedtest runner: %w", err)
	}

	// Initialize storage if saving results
	var store storage.Storage
	if !testNoSave {
		store, err = storage.NewStorage(cfg.Storage)
		if err != nil {
			return fmt.Errorf("failed to create storage: %w", err)
		}
		if err := store.Init(context.Background()); err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}
		defer func() { _ = store.Close() }()
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Received interrupt, cancelling tests...")
		cancel()
	}()

	// Print header
	if !testJSON {
		fmt.Println()
		fmt.Println("FlowGauge Speedtest")
		fmt.Println("===================")
		fmt.Printf("Testing %d connection(s)...\n\n", len(connections))
	}

	// Run tests
	logger.Info("Starting speedtests", zap.Int("connections", len(connections)))
	results, err := runner.RunAll(ctx)
	if err != nil {
		return fmt.Errorf("speedtest failed: %w", err)
	}

	// Save results to storage
	if store != nil {
		for _, result := range results {
			dbResult := storage.FromSpeedtestResult(&result)
			if err := store.SaveResult(ctx, dbResult); err != nil {
				logger.Warn("Failed to save result", 
					zap.String("connection", result.ConnectionName),
					zap.Error(err),
				)
			} else {
				logger.Debug("Result saved", 
					zap.String("connection", result.ConnectionName),
					zap.Int64("id", dbResult.ID),
				)
			}
		}
	}

	// Output results
	if testJSON {
		fmt.Println(speedtest.Results(results).ToJSON())
	} else {
		fmt.Println(speedtest.Results(results).PrintTable())
		fmt.Println()

		// Summary
		rs := speedtest.Results(results)
		fmt.Printf("Summary: %d/%d tests successful\n", rs.SuccessCount(), len(results))
		if rs.SuccessCount() > 0 {
			fmt.Printf("Average: ↓ %.2f Mbps | ↑ %.2f Mbps | %.2f ms\n",
				rs.AverageDownload(),
				rs.AverageUpload(),
				rs.AverageLatency(),
			)
		}
		
		if store != nil {
			fmt.Printf("\n✅ Results saved to database\n")
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(testCmd)

	testCmd.Flags().StringVarP(&testConnection, "connection", "C", "",
		"test only a specific connection by name")
	testCmd.Flags().BoolVar(&testOnce, "once", false,
		"run test once and exit (default behavior)")
	testCmd.Flags().BoolVar(&testJSON, "json", false,
		"output results as JSON")
	testCmd.Flags().BoolVar(&testNoSave, "no-save", false,
		"don't save results to database")
}
