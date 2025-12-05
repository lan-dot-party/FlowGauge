// Package api provides the REST API server for FlowGauge.
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"

	"github.com/lan-dot-party/flowgauge/internal/config"
	"github.com/lan-dot-party/flowgauge/internal/speedtest"
	"github.com/lan-dot-party/flowgauge/internal/storage"
	"github.com/lan-dot-party/flowgauge/pkg/version"
)

// Server represents the HTTP web server (Dashboard + API).
type Server struct {
	config     *config.WebserverConfig
	fullConfig *config.Config
	storage    storage.Storage
	runner     *speedtest.MultiWANRunner
	logger     *zap.Logger
	router     chi.Router
	httpServer *http.Server
}

// NewServer creates a new API server instance.
func NewServer(cfg *config.Config, store storage.Storage, runner *speedtest.MultiWANRunner, logger *zap.Logger) (*Server, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	s := &Server{
		config:     &cfg.Webserver,
		fullConfig: cfg,
		storage:    store,
		runner:     runner,
		logger:     logger,
	}

	s.setupRouter()
	return s, nil
}

// setupRouter configures the Chi router with all routes and middleware.
func (s *Server) setupRouter() {
	r := chi.NewRouter()

	// Middleware stack
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(s.loggingMiddleware)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Basic Auth (if configured)
	if s.config.Auth != nil && s.config.Auth.Username != "" {
		r.Use(s.basicAuthMiddleware)
	}

	// Health check (no auth required)
	r.Get("/health", s.handleHealth)

	// Dashboard (Web UI)
	r.Get("/", s.handleDashboard)
	r.Get("/dashboard", s.handleDashboard)
	r.Get("/dashboard/cards", s.handleDashboardPartial)
	r.Get("/dashboard/connection/{name}/chart", s.handleConnectionChartData)

	// API Documentation
	r.Get("/api", s.handleAPIRedirect)
	r.Get("/api/", s.handleAPIDocs)

	// API v1 routes (Read-Only)
	r.Route("/api/v1", func(r chi.Router) {
		// Results
		r.Get("/results", s.handleGetResults)
		r.Get("/results/latest", s.handleGetLatestResults)
		r.Get("/results/{id}", s.handleGetResult)

		// Connections
		r.Get("/connections", s.handleGetConnections)
		r.Get("/connections/{name}/stats", s.handleGetConnectionStats)

		// Metrics
		r.Get("/metrics", s.handlePrometheusMetrics)
	})

	s.router = r
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:         s.config.Listen,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("Starting web server",
		zap.String("listen", s.config.Listen),
		zap.String("version", version.GetShortVersion()),
	)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down web server")
	return s.httpServer.Shutdown(ctx)
}

// Router returns the chi router (useful for testing).
func (s *Server) Router() chi.Router {
	return s.router
}

