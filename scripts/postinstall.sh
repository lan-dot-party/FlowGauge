#!/bin/sh
# FlowGauge post-installation script (deb, rpm, archlinux, apk).
# Must also work without systemd (containers, chroots, Alpine/OpenRC) and
# without bash.
set -e

# Create flowgauge user if it doesn't exist
if ! id -u flowgauge >/dev/null 2>&1; then
    if command -v useradd >/dev/null 2>&1; then
        useradd --system --no-create-home --shell /usr/sbin/nologin flowgauge 2>/dev/null ||
            useradd --system --no-create-home --shell /sbin/nologin flowgauge
    else
        # Alpine / BusyBox
        addgroup -S flowgauge 2>/dev/null || true
        adduser -S -D -H -s /sbin/nologin -G flowgauge flowgauge
    fi
    echo "Created flowgauge user"
fi

# Create directories
mkdir -p /var/lib/flowgauge
mkdir -p /etc/flowgauge

# Set permissions
chown -R flowgauge:flowgauge /var/lib/flowgauge
chmod 750 /var/lib/flowgauge

chown -R root:flowgauge /etc/flowgauge
chmod 750 /etc/flowgauge

# Create default config if it doesn't exist
if [ ! -f /etc/flowgauge/config.yaml ]; then
    if [ -f /etc/flowgauge/config.yaml.example ]; then
        cp /etc/flowgauge/config.yaml.example /etc/flowgauge/config.yaml
        chown root:flowgauge /etc/flowgauge/config.yaml
        chmod 640 /etc/flowgauge/config.yaml
        echo "Created default configuration at /etc/flowgauge/config.yaml"
    fi
fi

# Set capabilities for DSCP marking (optional, only if binary exists)
if [ -f /usr/bin/flowgauge ] && command -v setcap >/dev/null 2>&1; then
    setcap 'cap_net_admin,cap_net_raw+ep' /usr/bin/flowgauge 2>/dev/null || true
fi

# Reload systemd only when it is actually running (not in containers/chroots)
if [ -d /run/systemd/system ] && command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload || true
    echo ""
    echo "FlowGauge installed successfully!"
    echo ""
    echo "Next steps:"
    echo "  1. Edit configuration: sudo nano /etc/flowgauge/config.yaml"
    echo "  2. Start the service:  sudo systemctl start flowgauge"
    echo "  3. Enable on boot:     sudo systemctl enable flowgauge"
    echo "  4. Check status:       sudo systemctl status flowgauge"
    echo ""
fi

exit 0
