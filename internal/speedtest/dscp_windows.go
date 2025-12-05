//go:build windows

// Package speedtest provides bandwidth testing functionality with DSCP and Multi-WAN support.
package speedtest

import (
	"syscall"

	"go.uber.org/zap"
)

// controlFunc is a no-op on Windows as DSCP marking requires elevated privileges
// and different Windows API calls (QoS API).
func (d *DSCPDialer) controlFunc(network, address string, c syscall.RawConn) error {
	if d.Logger != nil {
		d.Logger.Warn("DSCP marking is not supported on Windows, skipping",
			zap.Int("dscp", d.DSCP),
		)
	}
	return nil
}


