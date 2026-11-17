package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIPMiddleware(t *testing.T) {
	localProxy := []string{"127.0.0.1/32", "::1/128"}

	tests := []struct {
		name    string
		trusted []string
		remote  string
		headers map[string]string
		want    string
	}{
		{
			name:   "no trusted proxies uses TCP peer",
			remote: "203.0.113.7:4711",
			want:   "203.0.113.7",
		},
		{
			name:   "no trusted proxies ignores spoofed headers",
			remote: "203.0.113.7:4711",
			headers: map[string]string{
				"True-Client-IP":  "198.51.100.1",
				"X-Real-IP":       "198.51.100.2",
				"X-Forwarded-For": "198.51.100.3",
			},
			want: "203.0.113.7",
		},
		{
			name:    "trusted proxy: client from X-Forwarded-For",
			trusted: localProxy,
			remote:  "127.0.0.1:50000",
			headers: map[string]string{"X-Forwarded-For": "192.0.2.10"},
			want:    "192.0.2.10",
		},
		{
			name:    "trusted proxy: client-supplied XFF prefix is skipped",
			trusted: localProxy,
			remote:  "127.0.0.1:50000",
			// nginx appends the real peer to whatever the client sent.
			headers: map[string]string{"X-Forwarded-For": "198.51.100.3, 192.0.2.10"},
			want:    "192.0.2.10",
		},
		{
			name:    "trusted proxy: True-Client-IP is never read",
			trusted: localProxy,
			remote:  "127.0.0.1:50000",
			headers: map[string]string{
				"True-Client-IP":  "198.51.100.1",
				"X-Forwarded-For": "192.0.2.10",
			},
			want: "192.0.2.10",
		},
		{
			name:    "untrusted peer cannot spoof via XFF",
			trusted: localProxy,
			remote:  "203.0.113.7:4711",
			headers: map[string]string{"X-Forwarded-For": "198.51.100.3"},
			want:    "203.0.113.7",
		},
		{
			name:    "malformed XFF falls back to raw RemoteAddr",
			trusted: localProxy,
			remote:  "127.0.0.1:50000",
			headers: map[string]string{"X-Forwarded-For": "not-an-ip"},
			want:    "127.0.0.1:50000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			h := clientIPMiddleware(tt.trusted)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = clientIP(r)
				if r.RemoteAddr != tt.remote {
					t.Errorf("RemoteAddr mutated to %q", r.RemoteAddr)
				}
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remote
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if got != tt.want {
				t.Errorf("clientIP = %q, want %q", got, tt.want)
			}
		})
	}
}
