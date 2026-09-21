package discord

import (
	"testing"

	"github.com/lan-dot-party/flowgauge/internal/config"
	"github.com/lan-dot-party/flowgauge/internal/speedtest"
)

func boolPtr(v bool) *bool { return &v }

func TestCountingNames(t *testing.T) {
	cfg := &config.Config{
		Connections: []config.ConnectionConfig{
			{Name: "WAN1", Enabled: true},                                // unset -> counts
			{Name: "WAN2", Enabled: true, CountsToTotal: boolPtr(true)},  // explicit yes
			{Name: "VoIP", Enabled: true, CountsToTotal: boolPtr(false)}, // explicit no
			{Name: "Old", Enabled: false, CountsToTotal: boolPtr(true)},  // disabled
		},
	}

	counting := CountingNames(cfg.GetTotalConnections())

	for name, want := range map[string]bool{
		"WAN1": true,
		"WAN2": true,
		"VoIP": false,
		"Old":  false,
	} {
		if got := counting[name]; got != want {
			t.Errorf("counting[%q] = %v, want %v", name, got, want)
		}
	}
}

func TestAggregate(t *testing.T) {
	counting := map[string]bool{"WAN1": true, "WAN2": true}

	results := []speedtest.Result{
		{ConnectionName: "WAN1", DownloadMbps: 120.5, UploadMbps: 40.25, LatencyMs: 11.8},
		{ConnectionName: "WAN2", DownloadMbps: 250.0, UploadMbps: 60.0, LatencyMs: 24.4},
		{ConnectionName: "VoIP", DownloadMbps: 118.0, UploadMbps: 39.0, LatencyMs: 90.0}, // not counting
		{ConnectionName: "WAN2", Error: "dial timeout", LatencyMs: 999.0},                // failed
	}

	total := Aggregate(results, counting)

	if total.Connections != 2 {
		t.Errorf("Connections = %d, want 2", total.Connections)
	}
	if total.Skipped != 2 {
		t.Errorf("Skipped = %d, want 2", total.Skipped)
	}
	if total.DownloadMbps != 370.5 {
		t.Errorf("DownloadMbps = %v, want 370.5", total.DownloadMbps)
	}
	if total.UploadMbps != 100.25 {
		t.Errorf("UploadMbps = %v, want 100.25", total.UploadMbps)
	}
	// The worst counting connection, not the excluded VoIP test and not the
	// failed run.
	if total.LatencyMs != 24.4 {
		t.Errorf("LatencyMs = %v, want 24.4 (the worst counting connection)", total.LatencyMs)
	}
}

// Latency must be the maximum, never a sum or an average.
func TestAggregateLatencyTakesTheWorstValue(t *testing.T) {
	results := []speedtest.Result{
		{ConnectionName: "WAN1", LatencyMs: 6.0},
		{ConnectionName: "WAN2", LatencyMs: 42.5},
		{ConnectionName: "WAN3", LatencyMs: 18.0},
	}

	total := Aggregate(results, map[string]bool{"WAN1": true, "WAN2": true, "WAN3": true})

	if total.LatencyMs != 42.5 {
		t.Errorf("LatencyMs = %v, want 42.5", total.LatencyMs)
	}
}

func TestAggregateEmpty(t *testing.T) {
	total := Aggregate(nil, map[string]bool{"WAN1": true})

	if total.Connections != 0 || total.DownloadMbps != 0 || total.UploadMbps != 0 || total.LatencyMs != 0 {
		t.Errorf("empty results gave %+v, want a zero total", total)
	}
}

// A run where every counting connection failed must report nothing, so the
// channel keeps its last good value instead of showing 0 MBit/s.
func TestAggregateAllFailed(t *testing.T) {
	results := []speedtest.Result{
		{ConnectionName: "WAN1", Error: "no route to host"},
		{ConnectionName: "WAN2", Error: "context deadline exceeded"},
	}

	total := Aggregate(results, map[string]bool{"WAN1": true, "WAN2": true})

	if total.Connections != 0 {
		t.Errorf("Connections = %d, want 0", total.Connections)
	}
	if total.Skipped != 2 {
		t.Errorf("Skipped = %d, want 2", total.Skipped)
	}
}
