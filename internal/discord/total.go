// Package discord updates a Discord channel name with the measured bandwidth.
package discord

import (
	"github.com/lan-dot-party/flowgauge/internal/config"
	"github.com/lan-dot-party/flowgauge/internal/speedtest"
)

// Total is the aggregated result of all connections that count towards it.
type Total struct {
	// DownloadMbps and UploadMbps are the sums over all counting connections,
	// because in a multi-WAN setup the lines add up.
	DownloadMbps float64
	UploadMbps   float64
	// LatencyMs is the worst (highest) latency of all counting connections.
	// Latency does not add up - the slowest line is what users notice.
	LatencyMs float64
	// Connections is the number of successful results included.
	Connections int
	// Skipped is the number of results left out (failed or not counting).
	Skipped int
}

// CountingNames builds the lookup used by Aggregate from the configured
// connections, using config.Config.GetTotalConnections.
func CountingNames(connections []config.ConnectionConfig) map[string]bool {
	names := make(map[string]bool, len(connections))
	for _, conn := range connections {
		names[conn.Name] = true
	}
	return names
}

// Aggregate combines all successful results whose connection counts towards the
// total: bandwidth is summed, latency is the worst measured value. Failed tests
// are skipped so a single broken WAN does not silently drag the total down.
func Aggregate(results []speedtest.Result, counting map[string]bool) Total {
	var total Total

	for i := range results {
		result := &results[i]
		if result.IsError() || !counting[result.ConnectionName] {
			total.Skipped++
			continue
		}

		total.DownloadMbps += result.DownloadMbps
		total.UploadMbps += result.UploadMbps
		if result.LatencyMs > total.LatencyMs {
			total.LatencyMs = result.LatencyMs
		}
		total.Connections++
	}

	return total
}
