// Package speedtest provides bandwidth testing functionality with DSCP and Multi-WAN support.
package speedtest

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
)

// DSCPDialer creates network connections with DSCP marking and optional source IP binding.
type DSCPDialer struct {
	// DSCP is the Differentiated Services Code Point value (0-63)
	DSCP int
	// SourceIP is the local IP address to bind to (optional)
	SourceIP string
	// Logger for debug output
	Logger *zap.Logger
}

// Dial creates a new connection to the address on the named network.
func (d *DSCPDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

// DialContext creates a new connection to the address on the named network with context.
func (d *DSCPDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	// Create base dialer with optional source IP
	dialer := &net.Dialer{}

	if d.SourceIP != "" {
		ip := net.ParseIP(d.SourceIP)
		if ip == nil {
			return nil, fmt.Errorf("invalid source IP: %s", d.SourceIP)
		}

		// Determine if we need TCP or UDP local address
		switch network {
		case "tcp", "tcp4", "tcp6":
			dialer.LocalAddr = &net.TCPAddr{IP: ip}
		case "udp", "udp4", "udp6":
			dialer.LocalAddr = &net.UDPAddr{IP: ip}
		default:
			dialer.LocalAddr = &net.TCPAddr{IP: ip}
		}

		if d.Logger != nil {
			d.Logger.Debug("Binding to source IP", zap.String("ip", d.SourceIP))
		}
	}

	// Set up control function to apply DSCP before connection
	if d.DSCP > 0 {
		dialer.Control = d.controlFunc
	}

	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// NewDSCPDialer creates a new DSCPDialer with the given settings.
func NewDSCPDialer(dscp int, sourceIP string, logger *zap.Logger) (*DSCPDialer, error) {
	if dscp < 0 || dscp > 63 {
		return nil, fmt.Errorf("DSCP value must be between 0 and 63, got %d", dscp)
	}

	if sourceIP != "" {
		if ip := net.ParseIP(sourceIP); ip == nil {
			return nil, fmt.Errorf("invalid source IP address: %s", sourceIP)
		}
	}

	return &DSCPDialer{
		DSCP:     dscp,
		SourceIP: sourceIP,
		Logger:   logger,
	}, nil
}

// DSCPToTOS converts a DSCP value to the TOS byte value.
func DSCPToTOS(dscp int) int {
	return dscp << 2
}

// TOSToDSCP converts a TOS byte value to DSCP.
func TOSToDSCP(tos int) int {
	return tos >> 2
}

