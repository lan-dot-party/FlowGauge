package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/lan-dot-party/flowgauge/internal/config"
)

func newTestServer(t *testing.T, cfg *config.Config) *Server {
	t.Helper()
	s, err := NewServer(cfg, nil, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return s
}

func TestCORSNeverAllowsCredentials(t *testing.T) {
	s := newTestServer(t, config.NewDefault())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want empty", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
	}

	pre := httptest.NewRequest(http.MethodOptions, "/api/v1/results", nil)
	pre.Header.Set("Origin", "https://evil.example")
	pre.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rec = httptest.NewRecorder()
	s.router.ServeHTTP(rec, pre)
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "" {
		t.Errorf("preflight for POST allowed methods %q, want none", got)
	}
}

func TestIsLoopbackListen(t *testing.T) {
	for listen, want := range map[string]bool{
		"127.0.0.1:9020": true,
		"[::1]:8080":     true,
		"localhost:8080": true,
		"0.0.0.0:8080":   false,
		":8080":          false,
		"[::]:8080":      false,
		"10.0.0.5:8080":  false,
		"garbage":        false,
	} {
		if got := isLoopbackListen(listen); got != want {
			t.Errorf("isLoopbackListen(%q) = %v, want %v", listen, got, want)
		}
	}
}
