package speedtest

import (
	"context"
	"testing"
	"time"
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
