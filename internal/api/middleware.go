package api

import (
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// loggingMiddleware logs HTTP requests using zap.
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			s.logger.Debug("HTTP request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.Status()),
				zap.Int("bytes", ww.BytesWritten()),
				zap.Duration("duration", time.Since(start)),
				zap.String("remote", r.RemoteAddr),
				zap.String("request_id", middleware.GetReqID(r.Context())),
			)
		}()

		next.ServeHTTP(ww, r)
	})
}

// basicAuthMiddleware implements HTTP Basic Authentication.
func (s *Server) basicAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health endpoint
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		user, pass, ok := r.BasicAuth()
		if !ok {
			s.unauthorized(w)
			return
		}

		// Constant-time comparison to prevent timing attacks
		userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(s.config.Auth.Username)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(s.config.Auth.Password)) == 1

		if !userMatch || !passMatch {
			s.logger.Warn("Authentication failed",
				zap.String("user", user),
				zap.String("remote", r.RemoteAddr),
			)
			s.unauthorized(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// unauthorized sends a 401 response with WWW-Authenticate header.
func (s *Server) unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="FlowGauge API"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}


