package speedtest

import (
	"context"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"

	"github.com/lan-dot-party/flowgauge/internal/config"
)

// WANConnection represents a network connection configuration for testing.
type WANConnection struct {
	Name     string
	SourceIP string
	DSCP     int
	Enabled  bool
}

// WANConnectionFromConfig converts a config.ConnectionConfig to WANConnection.
func WANConnectionFromConfig(cfg config.ConnectionConfig) WANConnection {
	return WANConnection{
		Name:     cfg.Name,
		SourceIP: cfg.SourceIP,
		DSCP:     cfg.DSCP,
		Enabled:  cfg.Enabled,
	}
}

// MultiWANRunner manages speedtests across multiple WAN connections.
type MultiWANRunner struct {
	connections []WANConnection
	runner      *Runner
	logger      *zap.Logger
}

// NewMultiWANRunner creates a new MultiWANRunner from configuration.
func NewMultiWANRunner(connections []config.ConnectionConfig, cfg *config.SpeedtestConfig, logger *zap.Logger) (*MultiWANRunner, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Convert config connections to WANConnections
	wanConns := make([]WANConnection, 0, len(connections))
	for _, conn := range connections {
		if !conn.Enabled {
			continue
		}

		wanConn := WANConnectionFromConfig(conn)

		// Validate source IP exists on this system (if specified)
		if wanConn.SourceIP != "" {
			if err := validateSourceIP(wanConn.SourceIP); err != nil {
				logger.Warn("Source IP not available on system",
					zap.String("connection", wanConn.Name),
					zap.String("source_ip", wanConn.SourceIP),
					zap.Error(err),
				)
				// Continue anyway - might be valid later or on different system
			}
		}

		wanConns = append(wanConns, wanConn)
	}

	if len(wanConns) == 0 {
		return nil, fmt.Errorf("no enabled connections found")
	}

	runner, err := NewRunner(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create speedtest runner: %w", err)
	}

	return &MultiWANRunner{
		connections: wanConns,
		runner:      runner,
		logger:      logger,
	}, nil
}

// RunAll executes speedtests for all configured connections.
func (m *MultiWANRunner) RunAll(ctx context.Context) ([]Result, error) {
	return m.runSequential(ctx)
}

// runSequential executes tests one after another.
func (m *MultiWANRunner) runSequential(ctx context.Context) ([]Result, error) {
	results := make([]Result, 0, len(m.connections))

	for _, conn := range m.connections {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		m.logger.Info("Testing connection",
			zap.String("name", conn.Name),
			zap.String("source_ip", conn.SourceIP),
			zap.Int("dscp", conn.DSCP),
		)

		result, err := m.runner.Run(ctx, conn)
		if err != nil {
			m.logger.Error("Speedtest failed",
				zap.String("connection", conn.Name),
				zap.Error(err),
			)
			// Keep the runner's partial result (timestamp, server) and record
			// the error instead of failing completely.
			if result == nil {
				result = &Result{
					ConnectionName: conn.Name,
					SourceIP:       conn.SourceIP,
					DSCP:           conn.DSCP,
				}
			}
			if result.Error == "" {
				result.Error = err.Error()
			}
		}
		if result.Timestamp.IsZero() {
			result.Timestamp = time.Now()
		}

		results = append(results, *result)
	}

	return results, nil
}

// RunConnection executes a speedtest for a specific connection by name.
func (m *MultiWANRunner) RunConnection(ctx context.Context, name string) (*Result, error) {
	for _, conn := range m.connections {
		if conn.Name == name {
			return m.runner.Run(ctx, conn)
		}
	}
	return nil, fmt.Errorf("connection %q not found", name)
}

// GetConnections returns all configured connections.
func (m *MultiWANRunner) GetConnections() []WANConnection {
	return m.connections
}

// validateSourceIP checks if the given IP address is available on this system.
func validateSourceIP(ip string) error {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address format")
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return fmt.Errorf("failed to get interface addresses: %w", err)
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			if ipNet.IP.Equal(parsedIP) {
				return nil
			}
		}
	}

	return fmt.Errorf("IP %s not found on any interface", ip)
}

