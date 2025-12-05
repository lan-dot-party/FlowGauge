#!/bin/bash
# FlowGauge pre-removal script

set -e

# Stop service if running
if command -v systemctl >/dev/null 2>&1; then
    if systemctl is-active --quiet flowgauge; then
        echo "Stopping flowgauge service..."
        systemctl stop flowgauge
    fi
    
    if systemctl is-enabled --quiet flowgauge 2>/dev/null; then
        echo "Disabling flowgauge service..."
        systemctl disable flowgauge
    fi
fi

echo "FlowGauge service stopped"

# Note: We don't remove the user, config, or data directory
# to preserve data in case of reinstallation

exit 0


