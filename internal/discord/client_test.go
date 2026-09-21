package discord

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// withTestServer points the package at a test server for the duration of a test.
func withTestServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	server := httptest.NewServer(handler)
	previous := apiBaseURL
	apiBaseURL = server.URL

	t.Cleanup(func() {
		apiBaseURL = previous
		server.Close()
	})
}

func TestSetChannelName(t *testing.T) {
	var gotMethod, gotPath, gotAuth, gotName string

	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")

		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		_ = json.Unmarshal(body, &payload)
		gotName = payload["name"]

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"42"}`))
	})

	client := NewClient("test-token", time.Second)
	if err := client.SetChannelName(context.Background(), "42", "📶: ↓ 370 MBit/s"); err != nil {
		t.Fatalf("SetChannelName() returned %v", err)
	}

	if gotMethod != http.MethodPatch {
		t.Errorf("method = %s, want PATCH", gotMethod)
	}
	if gotPath != "/channels/42" {
		t.Errorf("path = %s, want /channels/42", gotPath)
	}
	if gotAuth != "Bot test-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bot test-token")
	}
	if gotName != "📶: ↓ 370 MBit/s" {
		t.Errorf("name = %q", gotName)
	}
}

func TestSetChannelNameRetriesOnceAfterRateLimit(t *testing.T) {
	var calls int

	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"retry_after":0.01}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	client := NewClient("test-token", time.Second)
	if err := client.SetChannelName(context.Background(), "42", "name"); err != nil {
		t.Fatalf("SetChannelName() returned %v", err)
	}

	if calls != 2 {
		t.Errorf("server was called %d times, want 2", calls)
	}
}

func TestSetChannelNameErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantSubstr string
	}{
		{"unauthorized", http.StatusUnauthorized, "bot_token"},
		{"forbidden", http.StatusForbidden, "MANAGE_CHANNELS"},
		{"not found", http.StatusNotFound, "channel_id"},
		{"server error", http.StatusInternalServerError, "500"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			})

			client := NewClient("test-token", time.Second)
			err := client.SetChannelName(context.Background(), "42", "name")

			if err == nil {
				t.Fatalf("SetChannelName() returned no error for status %d", tc.statusCode)
			}
			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Errorf("error %q does not mention %q", err, tc.wantSubstr)
			}
			if strings.Contains(err.Error(), "test-token") {
				t.Errorf("error leaks the bot token: %q", err)
			}
		})
	}
}

// Names longer than Discord's limit must be truncated before they are sent,
// otherwise the API rejects the whole request.
func TestSetChannelNameTruncatesBeforeSending(t *testing.T) {
	var gotName string

	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		_ = json.Unmarshal(body, &payload)
		gotName = payload["name"]
		w.WriteHeader(http.StatusOK)
	})

	client := NewClient("test-token", time.Second)
	if err := client.SetChannelName(context.Background(), "42", strings.Repeat("a", 150)); err != nil {
		t.Fatalf("SetChannelName() returned %v", err)
	}

	if len(gotName) != MaxChannelNameLength {
		t.Errorf("sent a %d character name, want %d", len(gotName), MaxChannelNameLength)
	}
}
