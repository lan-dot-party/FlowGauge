package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/lan-dot-party/flowgauge/internal/storage"
)

var (
	resultsConnection string
	resultsLimit      int
	resultsJSON       bool
	resultsSince      string
	resultsStats      bool
	resultsStatsPeriod string
)

// resultsCmd represents the results command
var resultsCmd = &cobra.Command{
	Use:   "results",
	Short: "Show speedtest results",
	Long: `Display stored speedtest results.

Examples:
  # Show recent results
  flowgauge results

  # Show results for a specific connection
  flowgauge results --connection WAN1

  # Show last 10 results as JSON
  flowgauge results --limit 10 --json
  
  # Show results from the last 24 hours
  flowgauge results --since 24h
  
  # Show statistics for a connection
  flowgauge results --stats --connection WAN1 --period 7d`,
	RunE: runResults,
}

func runResults(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Initialize storage
	store, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}
	if err := store.Init(context.Background()); err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// Show statistics if requested
	if resultsStats {
		return showStats(ctx, store)
	}

	// Build filter
	filter := storage.ResultFilter{
		ConnectionName: resultsConnection,
		Limit:          resultsLimit,
	}

	// Parse since duration
	if resultsSince != "" {
		duration, err := time.ParseDuration(resultsSince)
		if err != nil {
			return fmt.Errorf("invalid duration format for --since: %w", err)
		}
		filter.Since = time.Now().Add(-duration)
	}

	// Get results
	results, err := store.GetResults(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get results: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	// Output results
	if resultsJSON {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal results: %w", err)
		}
		fmt.Println(string(data))
	} else {
		printResultsTable(results)
	}

	return nil
}

func showStats(ctx context.Context, store storage.Storage) error {
	// Parse period
	period := 24 * time.Hour // Default 24h
	if resultsStatsPeriod != "" {
		var err error
		period, err = time.ParseDuration(resultsStatsPeriod)
		if err != nil {
			return fmt.Errorf("invalid duration format for --period: %w", err)
		}
	}

	if resultsConnection == "" {
		return fmt.Errorf("--connection is required when using --stats")
	}

	stats, err := store.GetStats(ctx, resultsConnection, period)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	if resultsJSON {
		data, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal stats: %w", err)
		}
		fmt.Println(string(data))
	} else {
		printStats(stats)
	}

	return nil
}

func printResultsTable(results []storage.TestResult) {
	fmt.Println()
	fmt.Println("Speedtest Results")
	fmt.Println("=================")
	fmt.Println()
	
	// Header
	fmt.Printf("%-5s | %-20s | %11s | %14s | %14s | %-20s | %s\n",
		"ID", "Connection", "Latency", "Download", "Upload", "Server", "Time")
	fmt.Println("------+----------------------+-------------+----------------+----------------+----------------------+---------------------")

	for _, r := range results {
		timeStr := r.CreatedAt.Local().Format("2006-01-02 15:04:05")
		
		if r.IsError() {
			fmt.Printf("%-5d | %-20s | %-11s | %-14s | %-14s | %-20s | %s\n",
				r.ID, truncate(r.ConnectionName, 20), "ERROR", "-", "-", truncate(r.Error, 20), timeStr)
		} else {
			fmt.Printf("%-5d | %-20s | %8.2f ms | %10.2f Mbps | %10.2f Mbps | %-20s | %s\n",
				r.ID, truncate(r.ConnectionName, 20), r.LatencyMs, r.DownloadMbps, r.UploadMbps,
				truncate(r.ServerName, 20), timeStr)
		}
	}
	
	fmt.Println()
	fmt.Printf("Total: %d results\n", len(results))
}

func printStats(stats *storage.Stats) {
	fmt.Println()
	fmt.Printf("Statistics for: %s\n", stats.ConnectionName)
	fmt.Printf("Period: %s (from %s to %s)\n",
		stats.Period,
		stats.Since.Local().Format("2006-01-02 15:04"),
		stats.Until.Local().Format("2006-01-02 15:04"))
	fmt.Println("==========================================")
	fmt.Println()
	
	fmt.Printf("Tests:     %d total, %d errors\n", stats.TestCount, stats.ErrorCount)
	fmt.Println()
	
	if stats.TestCount > stats.ErrorCount {
		fmt.Println("Download (Mbps):")
		fmt.Printf("  Average: %.2f | Min: %.2f | Max: %.2f\n",
			stats.AvgDownload, stats.MinDownload, stats.MaxDownload)
		fmt.Println()
		
		fmt.Println("Upload (Mbps):")
		fmt.Printf("  Average: %.2f | Min: %.2f | Max: %.2f\n",
			stats.AvgUpload, stats.MinUpload, stats.MaxUpload)
		fmt.Println()
		
		fmt.Println("Latency (ms):")
		fmt.Printf("  Average: %.2f | Min: %.2f | Max: %.2f\n",
			stats.AvgLatency, stats.MinLatency, stats.MaxLatency)
	} else {
		fmt.Println("No successful tests in this period.")
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func init() {
	rootCmd.AddCommand(resultsCmd)

	resultsCmd.Flags().StringVarP(&resultsConnection, "connection", "C", "",
		"filter results by connection name")
	resultsCmd.Flags().IntVarP(&resultsLimit, "limit", "n", 10,
		"maximum number of results to show")
	resultsCmd.Flags().BoolVar(&resultsJSON, "json", false,
		"output results as JSON")
	resultsCmd.Flags().StringVar(&resultsSince, "since", "",
		"show results since duration (e.g., 24h, 7d)")
	resultsCmd.Flags().BoolVar(&resultsStats, "stats", false,
		"show statistics instead of individual results")
	resultsCmd.Flags().StringVar(&resultsStatsPeriod, "period", "24h",
		"time period for statistics (e.g., 24h, 7d, 30d)")
}
