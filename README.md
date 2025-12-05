# FlowGauge

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://golang.org/)

> 🌐 A modular bandwidth testing tool with Multi-WAN support, DSCP flag configuration, and Grafana-compatible API.

## ✨ Features

- **Multi-WAN Support** - Test multiple internet connections with different source IPs
- **DSCP Tagging** - Set QoS flags for more realistic tests in prioritized networks
- **Scheduled Tests** - Automatic tests via cron syntax
- **Web Dashboard** - Modern dashboard with real-time updates and charts
- **REST API** - JSON API for Grafana and other tools
- **Prometheus Metrics** - Native Prometheus support for monitoring
- **Flexible Storage** - SQLite (default) or PostgreSQL

## 🚀 Quick Start

### Installation

```bash
# Via Go Install (recommended)
go install github.com/lan-dot-party/flowgauge/cmd/flowgauge@latest

# Or: Download binary
# See Releases page for .deb, .rpm and binaries
```

### Getting Started

```bash
# Create example configuration
flowgauge config init > /etc/flowgauge/config.yaml

# Edit configuration
nano /etc/flowgauge/config.yaml

# Run a single test
flowgauge test --once

# Start server with API and scheduler
flowgauge server
```

## ⚙️ Configuration

FlowGauge is configured via a YAML file. Default path: `/etc/flowgauge/config.yaml`

```yaml
general:
  log_level: info
  data_dir: /var/lib/flowgauge

storage:
  type: sqlite
  sqlite:
    path: /var/lib/flowgauge/results.db

api:
  enabled: true
  listen: 127.0.0.1:8080

connections:
  - name: WAN1-Telekom
    source_ip: 192.168.1.100
    dscp: 0
    enabled: true
  
  - name: WAN2-Vodafone
    source_ip: 192.168.2.100
    dscp: 46  # Expedited Forwarding
    enabled: true

scheduler:
  enabled: true
  schedule: "*/30 * * * *"  # Every 30 minutes
```

## 🎨 Web Dashboard

The integrated web dashboard offers:
- **Real-time overview** of all connections with current measurements
- **History charts** for download, upload, and latency (24h)
- **Auto-refresh** every 30 seconds

Accessible at `http://localhost:8080/` when the server is running.

## 📊 API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /` | Web Dashboard |
| `GET /health` | Health Check |
| `GET /api/` | Interactive API Documentation |
| `GET /api/v1/results` | All test results |
| `GET /api/v1/results/latest` | Latest results per connection |
| `GET /api/v1/connections` | Configured connections |
| `GET /api/v1/connections/{name}/stats` | Statistics for a connection |
| `GET /api/v1/metrics` | Prometheus Metrics |

## 🐳 Docker

```bash
docker run -d \
  --name flowgauge \
  --cap-add NET_ADMIN \
  -v ./config.yaml:/etc/flowgauge/config.yaml:ro \
  -v flowgauge-data:/var/lib/flowgauge \
  -p 8080:8080 \
  ghcr.io/lan-dot-party/flowgauge:latest
```

## 🔧 System Requirements

- **Go 1.22+** (for development/build)
- **Linux** (recommended) - Full DSCP/Source-IP support
- **CAP_NET_ADMIN** - Permission for DSCP marking

## 📖 Documentation

- [API Documentation](docs/api.md) - REST API details & examples
- [Grafana Integration](grafana/README.md) - Dashboard setup

## 🤝 Contributing

Contributions are welcome! Please open an issue or pull request.

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

---

*Developed with ❤️ for network enthusiasts*
