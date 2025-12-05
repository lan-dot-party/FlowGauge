package speedtest

import (
	"encoding/json"
	"fmt"
	"time"
)

// Result represents the outcome of a single speedtest.
type Result struct {
	// Connection info
	ConnectionName string `json:"connection_name"`
	SourceIP       string `json:"source_ip,omitempty"`
	DSCP           int    `json:"dscp"`

	// Server info
	ServerID      int    `json:"server_id,omitempty"`
	ServerName    string `json:"server_name,omitempty"`
	ServerCountry string `json:"server_country,omitempty"`
	ServerHost    string `json:"server_host,omitempty"`

	// Test results
	LatencyMs     float64 `json:"latency_ms"`
	JitterMs      float64 `json:"jitter_ms,omitempty"`
	DownloadMbps  float64 `json:"download_mbps"`
	UploadMbps    float64 `json:"upload_mbps"`
	PacketLossPct float64 `json:"packet_loss_pct,omitempty"`

	// Metadata
	Timestamp time.Time `json:"timestamp"`
	Duration  float64   `json:"duration_seconds,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// IsError returns true if the result represents a failed test.
func (r *Result) IsError() bool {
	return r.Error != ""
}

// JSON returns the result as a JSON string.
func (r *Result) JSON() string {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal result: %v"}`, err)
	}
	return string(data)
}

// String returns a human-readable representation of the result.
func (r *Result) String() string {
	if r.IsError() {
		return fmt.Sprintf("%s: ERROR - %s", r.ConnectionName, r.Error)
	}

	return fmt.Sprintf(`%s:
  Server:    %s (%s)
  Latency:   %.2f ms
  Download:  %.2f Mbps
  Upload:    %.2f Mbps`,
		r.ConnectionName,
		r.ServerName,
		r.ServerCountry,
		r.LatencyMs,
		r.DownloadMbps,
		r.UploadMbps,
	)
}

// FormatTable returns a formatted table row for CLI output.
func (r *Result) FormatTable() string {
	if r.IsError() {
		return fmt.Sprintf("%-20s | %-10s | %s", r.ConnectionName, "ERROR", r.Error)
	}

	return fmt.Sprintf("%-20s | %8.2f ms | %10.2f Mbps | %10.2f Mbps | %s",
		r.ConnectionName,
		r.LatencyMs,
		r.DownloadMbps,
		r.UploadMbps,
		r.ServerName,
	)
}

// TableHeader returns the header for table-formatted output.
func TableHeader() string {
	return fmt.Sprintf("%-20s | %11s | %14s | %14s | %s",
		"Connection", "Latency", "Download", "Upload", "Server")
}

// TableSeparator returns a separator line for table-formatted output.
func TableSeparator() string {
	return "---------------------+-------------+----------------+----------------+------------------"
}

// Results is a collection of Result objects with helper methods.
type Results []Result

// ToJSON converts all results to JSON.
func (rs Results) ToJSON() string {
	data, err := json.MarshalIndent(rs, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal results: %v"}`, err)
	}
	return string(data)
}

// PrintTable prints results as a formatted table.
func (rs Results) PrintTable() string {
	if len(rs) == 0 {
		return "No results"
	}

	output := TableHeader() + "\n" + TableSeparator() + "\n"
	for _, r := range rs {
		output += r.FormatTable() + "\n"
	}
	return output
}

// SuccessCount returns the number of successful tests.
func (rs Results) SuccessCount() int {
	count := 0
	for _, r := range rs {
		if !r.IsError() {
			count++
		}
	}
	return count
}

// ErrorCount returns the number of failed tests.
func (rs Results) ErrorCount() int {
	return len(rs) - rs.SuccessCount()
}

// AverageDownload calculates the average download speed of successful tests.
func (rs Results) AverageDownload() float64 {
	var sum float64
	var count int
	for _, r := range rs {
		if !r.IsError() {
			sum += r.DownloadMbps
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// AverageUpload calculates the average upload speed of successful tests.
func (rs Results) AverageUpload() float64 {
	var sum float64
	var count int
	for _, r := range rs {
		if !r.IsError() {
			sum += r.UploadMbps
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// AverageLatency calculates the average latency of successful tests.
func (rs Results) AverageLatency() float64 {
	var sum float64
	var count int
	for _, r := range rs {
		if !r.IsError() {
			sum += r.LatencyMs
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

