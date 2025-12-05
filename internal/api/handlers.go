package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/lan-dot-party/flowgauge/internal/storage"
	"github.com/lan-dot-party/flowgauge/pkg/version"
)

// Response helpers

type errorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

type successResponse struct {
	Status  string      `json:"status"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type resultsResponse struct {
	Results []storage.TestResult `json:"results"`
	Meta    struct {
		Total   int `json:"total"`
		Limit   int `json:"limit"`
		Offset  int `json:"offset"`
	} `json:"meta"`
}

type connectionResponse struct {
	Name     string `json:"name"`
	SourceIP string `json:"source_ip,omitempty"`
	DSCP     int    `json:"dscp"`
	Enabled  bool   `json:"enabled"`
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, errorResponse{
		Error:   http.StatusText(status),
		Code:    status,
		Message: message,
	})
}

// Handlers

// handleHealth returns the server health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, healthResponse{
		Status:  "ok",
		Version: version.GetShortVersion(),
	})
}

// handleGetResults returns speedtest results with optional filtering.
func (s *Server) handleGetResults(w http.ResponseWriter, r *http.Request) {
	filter := storage.ResultFilter{}

	// Parse query parameters
	if conn := r.URL.Query().Get("connection"); conn != "" {
		filter.ConnectionName = conn
	}

	if since := r.URL.Query().Get("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			filter.Since = t
		} else if d, err := time.ParseDuration(since); err == nil {
			filter.Since = time.Now().Add(-d)
		}
	}

	if until := r.URL.Query().Get("until"); until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			filter.Until = t
		}
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			filter.Limit = l
		}
	} else {
		filter.Limit = 100 // Default limit
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	results, err := s.storage.GetResults(r.Context(), filter)
	if err != nil {
		s.logger.Error("Failed to get results", zap.Error(err))
		s.writeError(w, http.StatusInternalServerError, "Failed to retrieve results")
		return
	}

	response := resultsResponse{
		Results: results,
	}
	response.Meta.Total = len(results)
	response.Meta.Limit = filter.Limit
	response.Meta.Offset = filter.Offset

	s.writeJSON(w, http.StatusOK, response)
}

// handleGetLatestResults returns the most recent result for each connection.
func (s *Server) handleGetLatestResults(w http.ResponseWriter, r *http.Request) {
	results, err := s.storage.GetLatestResults(r.Context())
	if err != nil {
		s.logger.Error("Failed to get latest results", zap.Error(err))
		s.writeError(w, http.StatusInternalServerError, "Failed to retrieve latest results")
		return
	}

	s.writeJSON(w, http.StatusOK, successResponse{
		Status: "ok",
		Data:   results,
	})
}

// handleGetResult returns a single result by ID.
func (s *Server) handleGetResult(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid result ID")
		return
	}

	result, err := s.storage.GetResult(r.Context(), id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Result not found")
		return
	}

	s.writeJSON(w, http.StatusOK, successResponse{
		Status: "ok",
		Data:   result,
	})
}

// handleGetConnections returns all configured connections.
func (s *Server) handleGetConnections(w http.ResponseWriter, r *http.Request) {
	connections := make([]connectionResponse, 0, len(s.fullConfig.Connections))
	for _, conn := range s.fullConfig.Connections {
		connections = append(connections, connectionResponse{
			Name:     conn.Name,
			SourceIP: conn.SourceIP,
			DSCP:     conn.DSCP,
			Enabled:  conn.Enabled,
		})
	}

	s.writeJSON(w, http.StatusOK, successResponse{
		Status: "ok",
		Data:   connections,
	})
}

// handleGetConnectionStats returns statistics for a specific connection.
func (s *Server) handleGetConnectionStats(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "Connection name required")
		return
	}

	// Parse period (default 24h)
	period := 24 * time.Hour
	if p := r.URL.Query().Get("period"); p != "" {
		if d, err := time.ParseDuration(p); err == nil {
			period = d
		}
	}

	stats, err := s.storage.GetStats(r.Context(), name, period)
	if err != nil {
		s.logger.Error("Failed to get stats", zap.String("connection", name), zap.Error(err))
		s.writeError(w, http.StatusInternalServerError, "Failed to retrieve statistics")
		return
	}

	s.writeJSON(w, http.StatusOK, successResponse{
		Status: "ok",
		Data:   stats,
	})
}

// handleTriggerTest triggers a speedtest for all connections.
func (s *Server) handleTriggerTest(w http.ResponseWriter, r *http.Request) {
	if s.runner == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Speedtest runner not available")
		return
	}

	// Run test in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		s.logger.Info("API triggered speedtest for all connections")
		results, err := s.runner.RunAll(ctx)
		if err != nil {
			s.logger.Error("API triggered speedtest failed", zap.Error(err))
			return
		}

		// Save results
		for _, result := range results {
			dbResult := storage.FromSpeedtestResult(&result)
			if err := s.storage.SaveResult(ctx, dbResult); err != nil {
				s.logger.Error("Failed to save result", zap.Error(err))
			}
		}

		// Update Prometheus metrics
		UpdateMetrics(results)

		s.logger.Info("API triggered speedtest completed", zap.Int("results", len(results)))
	}()

	s.writeJSON(w, http.StatusAccepted, successResponse{
		Status:  "started",
		Message: "Speedtest started for all connections",
	})
}

// handleTriggerConnectionTest triggers a speedtest for a specific connection.
func (s *Server) handleTriggerConnectionTest(w http.ResponseWriter, r *http.Request) {
	if s.runner == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Speedtest runner not available")
		return
	}

	connName := chi.URLParam(r, "connection")
	if connName == "" {
		s.writeError(w, http.StatusBadRequest, "Connection name required")
		return
	}

	// Verify connection exists
	conn := s.fullConfig.GetConnectionByName(connName)
	if conn == nil {
		s.writeError(w, http.StatusNotFound, "Connection not found")
		return
	}

	// Run test in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		s.logger.Info("API triggered speedtest", zap.String("connection", connName))
		result, err := s.runner.RunConnection(ctx, connName)
		if err != nil {
			s.logger.Error("API triggered speedtest failed",
				zap.String("connection", connName),
				zap.Error(err),
			)
			return
		}

		// Save result
		dbResult := storage.FromSpeedtestResult(result)
		if err := s.storage.SaveResult(ctx, dbResult); err != nil {
			s.logger.Error("Failed to save result", zap.Error(err))
		}

		// Update Prometheus metrics
		UpdateMetricsForResult(result)

		s.logger.Info("API triggered speedtest completed", zap.String("connection", connName))
	}()

	s.writeJSON(w, http.StatusAccepted, successResponse{
		Status:  "started",
		Message: "Speedtest started for connection: " + connName,
	})
}


