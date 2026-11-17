package speedtest

import (
	"context"
	"testing"
	"time"

	"github.com/lan-dot-party/flowgauge/internal/config"
)

func TestRunAbortsOnCancelledContext(t *testing.T) {
	r, err := NewRunner(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	res, err := r.Run(ctx, WANConnection{Name: "test", Enabled: true})
	if err == nil {
		t.Fatal("expected an error for a cancelled context")
	}
	if res == nil || !res.IsError() {
		t.Fatal("expected an error result")
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("cancelled context did not abort promptly")
	}
}

func TestFailedRunKeepsTimestamp(t *testing.T) {
	// A 1ns speedtest.timeout makes the test fail immediately, without network.
	m, err := NewMultiWANRunner([]config.ConnectionConfig{{Name: "WAN1", Enabled: true}},
		&config.SpeedtestConfig{Timeout: time.Nanosecond}, nil)
	if err != nil {
		t.Fatal(err)
	}

	before := time.Now()
	results, err := m.RunAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].IsError() {
		t.Fatalf("expected one error result, got %+v", results)
	}
	if results[0].Timestamp.Before(before) {
		t.Errorf("failed result has timestamp %v, want >= %v", results[0].Timestamp, before)
	}
}
