package storage

import (
	"time"

	"github.com/lan-dot-party/flowgauge/internal/speedtest"
)

// TestResult represents a speedtest result stored in the database.
type TestResult struct {
	ID             int64     `json:"id"`
	ConnectionName string    `json:"connection_name"`
	ServerID       int       `json:"server_id,omitempty"`
	ServerName     string    `json:"server_name,omitempty"`
	ServerCountry  string    `json:"server_country,omitempty"`
	ServerHost     string    `json:"server_host,omitempty"`
	LatencyMs      float64   `json:"latency_ms"`
	JitterMs       float64   `json:"jitter_ms,omitempty"`
	DownloadMbps   float64   `json:"download_mbps"`
	UploadMbps     float64   `json:"upload_mbps"`
	PacketLossPct  float64   `json:"packet_loss_pct,omitempty"`
	SourceIP       string    `json:"source_ip,omitempty"`
	DSCP           int       `json:"dscp"`
	Error          string    `json:"error,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// FromSpeedtestResult converts a speedtest.Result to a storage TestResult.
func FromSpeedtestResult(r *speedtest.Result) *TestResult {
	return &TestResult{
		ConnectionName: r.ConnectionName,
		ServerID:       r.ServerID,
		ServerName:     r.ServerName,
		ServerCountry:  r.ServerCountry,
		ServerHost:     r.ServerHost,
		LatencyMs:      r.LatencyMs,
		JitterMs:       r.JitterMs,
		DownloadMbps:   r.DownloadMbps,
		UploadMbps:     r.UploadMbps,
		PacketLossPct:  r.PacketLossPct,
		SourceIP:       r.SourceIP,
		DSCP:           r.DSCP,
		Error:          r.Error,
		CreatedAt:      r.Timestamp,
	}
}

// ToSpeedtestResult converts a storage TestResult to a speedtest.Result.
func (r *TestResult) ToSpeedtestResult() *speedtest.Result {
	return &speedtest.Result{
		ConnectionName: r.ConnectionName,
		ServerID:       r.ServerID,
		ServerName:     r.ServerName,
		ServerCountry:  r.ServerCountry,
		ServerHost:     r.ServerHost,
		LatencyMs:      r.LatencyMs,
		JitterMs:       r.JitterMs,
		DownloadMbps:   r.DownloadMbps,
		UploadMbps:     r.UploadMbps,
		PacketLossPct:  r.PacketLossPct,
		SourceIP:       r.SourceIP,
		DSCP:           r.DSCP,
		Error:          r.Error,
		Timestamp:      r.CreatedAt,
	}
}

// IsError returns true if this result represents a failed test.
func (r *TestResult) IsError() bool {
	return r.Error != ""
}

