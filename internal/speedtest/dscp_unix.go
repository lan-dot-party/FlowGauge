//go:build !windows

// Package speedtest provides bandwidth testing functionality with DSCP and Multi-WAN support.
package speedtest

import (
	"fmt"
	"syscall"

	"go.uber.org/zap"
)

// controlFunc is called after creating the socket but before connecting.
// This is where we set the DSCP/TOS value.
func (d *DSCPDialer) controlFunc(network, address string, c syscall.RawConn) error {
	var setsockoptErr error

	err := c.Control(func(fd uintptr) {
		// DSCP is the upper 6 bits of the TOS byte
		// TOS = DSCP << 2
		tos := d.DSCP << 2

		// Determine IP version from network string
		isIPv6 := network == "tcp6" || network == "udp6"

		if isIPv6 {
			// IPv6: use IPV6_TCLASS
			setsockoptErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_TCLASS, tos)
		} else {
			// IPv4: use IP_TOS
			setsockoptErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TOS, tos)
		}

		if setsockoptErr == nil && d.Logger != nil {
			d.Logger.Debug("DSCP set successfully",
				zap.Int("dscp", d.DSCP),
				zap.Int("tos", tos),
				zap.Bool("ipv6", isIPv6),
			)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to access raw connection: %w", err)
	}
	if setsockoptErr != nil {
		return fmt.Errorf("failed to set DSCP (TOS=%d): %w", d.DSCP<<2, setsockoptErr)
	}

	return nil
}


