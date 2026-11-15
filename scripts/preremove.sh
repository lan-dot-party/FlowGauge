#!/bin/sh
# FlowGauge pre-removal script (deb, rpm, archlinux, apk).
set -e

# Stop service if systemd is running
if [ -d /run/systemd/system ] && command -v systemctl >/dev/null 2>&1; then
    if systemctl is-active --quiet flowgauge; then
        echo "Stopping flowgauge service..."
        systemctl stop flowgauge || true
    fi

    if systemctl is-enabled --quiet flowgauge 2>/dev/null; then
        echo "Disabling flowgauge service..."
        systemctl disable flowgauge || true
    fi
fi

# Note: We don't remove the user, config, or data directory
# to preserve data in case of reinstallation

exit 0
