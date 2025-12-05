package api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/lan-dot-party/flowgauge/internal/speedtest"
)

var (
	// Speedtest metrics
	downloadSpeed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "flowgauge",
			Name:      "download_speed_mbps",
			Help:      "Download speed in Mbps",
		},
		[]string{"connection", "server"},
	)

	uploadSpeed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "flowgauge",
			Name:      "upload_speed_mbps",
			Help:      "Upload speed in Mbps",
		},
		[]string{"connection", "server"},
	)

	latency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "flowgauge",
			Name:      "latency_ms",
			Help:      "Latency in milliseconds",
		},
		[]string{"connection", "server"},
	)

	jitter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "flowgauge",
			Name:      "jitter_ms",
			Help:      "Jitter in milliseconds",
		},
		[]string{"connection", "server"},
	)

	testTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "flowgauge",
			Name:      "last_test_timestamp",
			Help:      "Timestamp of the last speedtest (Unix timestamp)",
		},
		[]string{"connection"},
	)

	testDuration = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "flowgauge",
			Name:      "test_duration_seconds",
			Help:      "Duration of the speedtest in seconds",
		},
		[]string{"connection"},
	)

	testErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "flowgauge",
			Name:      "test_errors_total",
			Help:      "Total number of speedtest errors",
		},
		[]string{"connection"},
	)

	testsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "flowgauge",
			Name:      "tests_total",
			Help:      "Total number of speedtests run",
		},
		[]string{"connection"},
	)
)

func init() {
	// Register all metrics
	prometheus.MustRegister(
		downloadSpeed,
		uploadSpeed,
		latency,
		jitter,
		testTimestamp,
		testDuration,
		testErrors,
		testsTotal,
	)
}

// handlePrometheusMetrics exposes Prometheus metrics.
func (s *Server) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}

// UpdateMetrics updates Prometheus metrics for multiple results.
// Exported so it can be called from the scheduler.
func UpdateMetrics(results []speedtest.Result) {
	for _, result := range results {
		UpdateMetricsForResult(&result)
	}
}

// UpdateMetricsForResult updates Prometheus metrics for a single result.
// Exported so it can be called from the scheduler.
func UpdateMetricsForResult(result *speedtest.Result) {
	labels := prometheus.Labels{
		"connection": result.ConnectionName,
		"server":     result.ServerName,
	}

	testsTotal.WithLabelValues(result.ConnectionName).Inc()

	if result.IsError() {
		testErrors.WithLabelValues(result.ConnectionName).Inc()
		return
	}

	downloadSpeed.With(labels).Set(result.DownloadMbps)
	uploadSpeed.With(labels).Set(result.UploadMbps)
	latency.With(labels).Set(result.LatencyMs)
	jitter.With(labels).Set(result.JitterMs)

	testTimestamp.WithLabelValues(result.ConnectionName).Set(float64(result.Timestamp.Unix()))
	testDuration.WithLabelValues(result.ConnectionName).Set(result.Duration)
}


