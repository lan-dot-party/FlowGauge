package speedtest

import (
	"context"
	"fmt"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
	"go.uber.org/zap"

	"github.com/lan-dot-party/flowgauge/internal/config"
)

// Runner executes speedtests using speedtest-go.
type Runner struct {
	config *config.SpeedtestConfig
	logger *zap.Logger
}

// NewRunner creates a new speedtest Runner.
func NewRunner(cfg *config.SpeedtestConfig, logger *zap.Logger) (*Runner, error) {
	if cfg == nil {
		cfg = &config.SpeedtestConfig{
			Timeout:      60 * time.Second,
			DownloadSize: "auto",
			UploadSize:   "auto",
		}
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	return &Runner{
		config: cfg,
		logger: logger,
	}, nil
}

// Run executes a speedtest for the given WAN connection.
func (r *Runner) Run(ctx context.Context, conn WANConnection) (*Result, error) {
	startTime := time.Now()

	result := &Result{
		ConnectionName: conn.Name,
		SourceIP:       conn.SourceIP,
		DSCP:           conn.DSCP,
		Timestamp:      startTime,
	}

	// Create DSCP dialer for custom socket options
	dscpDialer, err := NewDSCPDialer(conn.DSCP, conn.SourceIP, r.logger)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create DSCP dialer: %v", err)
		return result, err
	}

	// Build UserConfig with DialerControl for DSCP marking
	// This is the proper way to inject custom socket options into speedtest-go
	userConfig := &speedtest.UserConfig{}
	
	// Set source IP if specified
	if conn.SourceIP != "" {
		userConfig.Source = conn.SourceIP
	}
	
	// Set DialerControl for DSCP marking (works with both Source IP and without)
	if conn.DSCP > 0 {
		userConfig.DialerControl = dscpDialer.controlFunc
	}
	
	// Create speedtest client with our custom config
	client := speedtest.New(
		speedtest.WithUserConfig(userConfig),
	)
	
	r.logger.Debug("Created speedtest client",
		zap.String("source_ip", conn.SourceIP),
		zap.Int("dscp", conn.DSCP),
		zap.Bool("has_dialer_control", conn.DSCP > 0),
	)

	// Fetch server list
	r.logger.Debug("Fetching speedtest servers")
	serverList, err := client.FetchServers()
	if err != nil {
		result.Error = fmt.Sprintf("failed to fetch servers: %v", err)
		return result, err
	}

	// Find servers (empty slice = auto-select based on latency)
	var serverIDs []int
	if len(r.config.ServerIDs) > 0 {
		serverIDs = r.config.ServerIDs
	}

	targets, err := serverList.FindServer(serverIDs)
	if err != nil {
		result.Error = fmt.Sprintf("failed to find server: %v", err)
		return result, err
	}

	if len(targets) == 0 {
		result.Error = "no speedtest servers available"
		return result, fmt.Errorf("%s", result.Error)
	}

	// Use the first (best) server
	server := targets[0]

	r.logger.Debug("Selected server",
		zap.String("name", server.Name),
		zap.String("country", server.Country),
		zap.String("host", server.Host),
		zap.String("id", server.ID),
	)

	// Store server info in result
	result.ServerName = server.Name
	result.ServerCountry = server.Country
	result.ServerHost = server.Host
	result.ServerID = parseServerID(server.ID)

	// Run ping test
	r.logger.Debug("Running latency test")
	if err := server.PingTest(nil); err != nil {
		r.logger.Warn("Ping test failed", zap.Error(err))
	} else {
		result.LatencyMs = float64(server.Latency.Milliseconds())
		result.JitterMs = float64(server.Jitter.Milliseconds())
	}

	// Run download test
	r.logger.Debug("Running download test")
	if err := server.DownloadTest(); err != nil {
		r.logger.Warn("Download test failed", zap.Error(err))
	}
	// Use ByteRate's Mbps() method for correct conversion
	result.DownloadMbps = server.DLSpeed.Mbps()
	r.logger.Debug("Download result",
		zap.Float64("raw_dlspeed", float64(server.DLSpeed)),
		zap.Float64("mbps", result.DownloadMbps),
	)

	// Run upload test
	r.logger.Debug("Running upload test")
	if err := server.UploadTest(); err != nil {
		r.logger.Warn("Upload test failed", zap.Error(err))
	}
	// Use ByteRate's Mbps() method for correct conversion
	result.UploadMbps = server.ULSpeed.Mbps()

	// Calculate duration
	result.Duration = time.Since(startTime).Seconds()

	r.logger.Debug("Speedtest completed",
		zap.String("connection", conn.Name),
		zap.Float64("download_mbps", result.DownloadMbps),
		zap.Float64("upload_mbps", result.UploadMbps),
		zap.Float64("latency_ms", result.LatencyMs),
		zap.Float64("duration_s", result.Duration),
	)

	return result, nil
}

// parseServerID converts server ID string to int.
func parseServerID(id string) int {
	var serverID int
	_, _ = fmt.Sscanf(id, "%d", &serverID)
	return serverID
}

// QuickTest performs a quick test to verify connectivity and returns basic results.
func (r *Runner) QuickTest(ctx context.Context) (*Result, error) {
	return r.Run(ctx, WANConnection{
		Name:    "default",
		Enabled: true,
	})
}
