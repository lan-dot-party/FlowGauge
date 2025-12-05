package speedtest

import (
	"context"
	"fmt"
	"net"
	"sync"

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
	parallel    bool
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
		parallel:    false, // Sequential by default to avoid bandwidth competition
	}, nil
}

// SetParallel enables or disables parallel testing.
// Warning: Parallel tests may interfere with each other's measurements.
func (m *MultiWANRunner) SetParallel(parallel bool) {
	m.parallel = parallel
}

// RunAll executes speedtests for all configured connections.
func (m *MultiWANRunner) RunAll(ctx context.Context) ([]Result, error) {
	if m.parallel {
		return m.runParallel(ctx)
	}
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
			// Create error result instead of failing completely
			result = &Result{
				ConnectionName: conn.Name,
				SourceIP:       conn.SourceIP,
				DSCP:           conn.DSCP,
				Error:          err.Error(),
			}
		}

		results = append(results, *result)
	}

	return results, nil
}

// runParallel executes tests concurrently.
func (m *MultiWANRunner) runParallel(ctx context.Context) ([]Result, error) {
	var wg sync.WaitGroup
	resultsChan := make(chan Result, len(m.connections))

	for _, conn := range m.connections {
		wg.Add(1)
		go func(c WANConnection) {
			defer wg.Done()

			m.logger.Info("Testing connection (parallel)",
				zap.String("name", c.Name),
			)

			result, err := m.runner.Run(ctx, c)
			if err != nil {
				m.logger.Error("Speedtest failed",
					zap.String("connection", c.Name),
					zap.Error(err),
				)
				result = &Result{
					ConnectionName: c.Name,
					SourceIP:       c.SourceIP,
					DSCP:           c.DSCP,
					Error:          err.Error(),
				}
			}

			resultsChan <- *result
		}(conn)
	}

	// Wait for all tests to complete
	wg.Wait()
	close(resultsChan)

	// Collect results
	results := make([]Result, 0, len(m.connections))
	for result := range resultsChan {
		results = append(results, result)
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

