#!/bin/bash
# FlowGauge post-installation script

set -e

# Create flowgauge user if it doesn't exist
if ! id -u flowgauge >/dev/null 2>&1; then
    useradd --system --no-create-home --shell /usr/sbin/nologin flowgauge
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
if [ -f /usr/bin/flowgauge ]; then
    setcap 'cap_net_admin,cap_net_raw+ep' /usr/bin/flowgauge 2>/dev/null || true
fi

# Reload systemd
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload
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


