# FlowGauge

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://golang.org/)

> 🌐 A modular bandwidth testing tool with Multi-WAN support, DSCP flag configuration, and Grafana-compatible API.

## ✨ Features

- **Multi-WAN Support** - Test multiple internet connections with different source IPs
- **DSCP Tagging** - Set QoS flags for more realistic tests in prioritized networks
- **Scheduled Tests** - Automatic tests via cron syntax
- **Web Dashboard** - Modern dashboard with real-time updates and charts
- **REST API** - JSON API for Grafana and other tools
- **Prometheus Metrics** - Native Prometheus support for monitoring
- **Discord Integration** - Shows the total bandwidth in a Discord channel name
- **Flexible Storage** - SQLite (default) or PostgreSQL

## 🚀 Quick Start

### Installation

#### From the lan.party package repository (recommended)

Set up the repository once, then install and update the package with your system's package manager.
Instructions for Debian/Ubuntu, Fedora/RHEL/Rocky, Arch Linux and Alpine: **https://repo.lan.party**

```sh
curl -fsSL https://repo.lan.party/setup.sh | sudo sh   # or follow the manual steps on repo.lan.party

sudo apt install flowgauge      # Debian / Ubuntu
sudo dnf install flowgauge      # Fedora / RHEL / Rocky
sudo pacman -S flowgauge        # Arch Linux
sudo apk add flowgauge          # Alpine
```

New releases then arrive with your normal system updates (`apt upgrade`, `dnf upgrade`, `pacman -Syu`, `apk upgrade`).

#### Other installation methods

```bash
# Via Go Install
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
    counts_to_total: true   # counts towards the total bandwidth (default)
  
  - name: WAN2-Vodafone
    source_ip: 192.168.2.100
    dscp: 46  # Expedited Forwarding
    enabled: true
    counts_to_total: false  # measured only, not added to the total

scheduler:
  enabled: true
  schedule: "*/30 * * * *"  # Every 30 minutes
```

See [`configs/flowgauge.example.yaml`](configs/flowgauge.example.yaml) for the fully
commented configuration.

## 📶 Discord Integration

After every test run FlowGauge can rename a Discord channel to the measured total
bandwidth and ping, so the current line quality is visible in the channel list:

```
📶: ↓ 370 MBit/s | ↑ 100 MBit/s | 🏓 12 ms
```

```yaml
discord:
  enabled: true
  bot_token: "your-bot-token"
  channel_id: "123456789012345678"
  name_template: "📶: ↓ {download} MBit/s | ↑ {upload} MBit/s | 🏓 {ping} ms"
  timeout: 15s
```

Download and upload are the **sum** of all enabled connections with
`counts_to_total: true` (the default); the ping is the **worst (highest)** latency
of those connections, because latency does not add up. Set `counts_to_total: false`
on connections that measure the same physical line again — for example a second
test with a different DSCP marking — so they do not distort the overview. Failed
tests are left out; if no counting connection succeeded, the channel keeps its
previous name.

**Setup:**

1. Create an application at [discord.com/developers](https://discord.com/developers/applications),
   add a **Bot** and copy its token
2. Invite the bot to your server with the **Manage Channels** permission
3. Enable Discord's developer mode and copy the channel ID
   (right-click the channel → *Copy Channel ID*)

**Use a voice channel or a category.** Text channels normalize their name
(lowercase, spaces turned into dashes), which mangles the arrows and spacing.

Available template placeholders: `{download}`, `{upload}`, `{download_mbps}`,
`{upload_mbps}`, `{download_gbps}`, `{upload_gbps}`, `{ping}`, `{ping_ms}`,
`{connections}`, `{time}`, `{date}` (`{latency}` and `{latency_ms}` are aliases
for `{ping}` and `{ping_ms}`). The rendered name must not exceed 100 characters.
Other emoji that read well for the ping: ⏱️, 📡, ⚡.

> **Note:** Discord allows only two channel renames per 10 minutes per channel.
> Scheduled runs are unaffected, but repeated manual `flowgauge test` runs will be
> rate limited. Use `flowgauge test --no-discord` to skip the update. A failed
> Discord update never fails the speedtest itself — results are still stored.

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

### Docker Compose

Two ready-to-use examples are included. They differ only in where the data
(SQLite database `results.db`) is stored:

| File | Data | When to use |
|------|------|-------------|
| [`compose.yaml`](compose.yaml) | named volume `flowgauge-data` | Simplest; Docker manages permissions |
| [`compose.bind.yaml`](compose.bind.yaml) | local directory `./data` | Easy access and backup; directory must belong to UID `100` / GID `101` (the container user) |

**1. Create the configuration.** The image only ships an example, so create
`config.yaml` next to the compose file and adjust it:

```bash
docker run --rm --entrypoint cat ghcr.io/lan-dot-party/flowgauge:latest \
  /etc/flowgauge/config.yaml.example > config.yaml
```

Set `webserver.listen: 0.0.0.0:8080` - the default `127.0.0.1:8080` is not
reachable through Docker's port mapping. The compose files publish the port on
`127.0.0.1` only (for a reverse proxy on the same host); change it to
`"8080:8080"` to expose it directly.
To log real client IPs behind that proxy, add the address the proxy connects
from (inside the container usually the Docker network gateway, e.g.
`172.16.0.0/12`) to `webserver.trusted_proxies`, see [docs/api.md](docs/api.md#client-ip-behind-a-reverse-proxy).

**2. Start, logs, update** (for the local path variant add `-f compose.bind.yaml`
and first run `mkdir -p data && sudo chown 100:101 data`):

```bash
docker compose up -d
docker compose logs -f
docker compose pull && docker compose up -d   # update
```

**Backup.** FlowGauge has no built-in backup command. Stop the container first
so the SQLite database is consistent:

```bash
docker compose stop
# named volume
docker run --rm -v flowgauge-data:/data -v "$PWD":/backup alpine \
  tar czf /backup/flowgauge-data.tgz -C /data .
# local path variant: back up ./data and config.yaml
tar czf flowgauge-backup.tgz data config.yaml
docker compose start
```

**Capabilities.** `NET_ADMIN` and `NET_RAW` are only needed for DSCP marking.
For Multi-WAN setups with `source_ip`, the address must exist inside the
container; use `network_mode: host` (see the comment in the compose files).
Pin a version tag instead of `latest` for production.

## 🔧 System Requirements

- **Go 1.26+** (for development/build)
- **Linux** (recommended) - Full DSCP/Source-IP support
- **CAP_NET_ADMIN** - Permission for DSCP marking

## 📖 Documentation

- [API Documentation](docs/api.md) - REST API details & examples
- [Grafana Integration](grafana/README.md) - Dashboard setup

## 🔗 Related projects

FlowGauge is part of the lan-dot-party tool suite with
[ERS-Proxy](https://github.com/lan-dot-party/ERS-Proxy),
[LANLeecher](https://github.com/lan-dot-party/LANLeecher),
[LANPilot](https://github.com/lan-dot-party/LANPilot) and
[LANventory](https://github.com/lan-dot-party/LANventory).

## 🤝 Contributing

Contributions are welcome! Please open an issue or pull request.

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

---

*Developed with ❤️ for network enthusiasts*
