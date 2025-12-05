# FlowGauge Grafana Integration

## Prerequisites

1. Grafana 9.0+ installed
2. FlowGauge running and accessible

## Installation

### 1. Install Infinity Plugin

```bash
# Via grafana-cli
grafana-cli plugins install yesoreyeram-infinity-datasource

# Or in Docker
docker exec -it grafana grafana-cli plugins install yesoreyeram-infinity-datasource

# Then restart Grafana
sudo systemctl restart grafana-server
```

### 2. Configure Datasource

#### Option A: Manually in Grafana UI

1. Grafana → Configuration → Data Sources → Add data source
2. Search for "Infinity" and select it
3. Name: `FlowGauge`
4. Base URL: `http://flowgauge-server:8080`
5. Save & Test

#### Option B: Provisioning (automatic)

Copy `provisioning/datasources/flowgauge.yaml` to `/etc/grafana/provisioning/datasources/`

### 3. Import Dashboard

1. Grafana → Dashboards → Import
2. Upload `dashboards/flowgauge-dashboard.json`
3. Select "FlowGauge" as Datasource
4. Import

## Dashboard Features

- **Current Values**: Download, Upload, Latency per connection
- **Speed History**: Time series of speeds (24h)
- **Latency History**: Time series of latency
- **Connection Table**: Overview of all connections with gauges
- **Auto-Refresh**: Every 30 seconds

## API Endpoints for Custom Panels

| Endpoint | Description |
|----------|-------------|
| `/api/v1/results/latest` | Latest results per connection |
| `/api/v1/results?limit=100` | Last 100 results |
| `/api/v1/results?connection=WAN1` | Filter by connection |
| `/api/v1/connections/{name}/stats` | Statistics for a connection |

## Example: Create Custom Panel

### Download Speed Gauge

1. Add Panel → Visualization: Stat
2. Query:
   - Type: JSON
   - URL: `/api/v1/results/latest`
   - Format: Table
3. Field: `download_mbps`
4. Unit: Mbps

### Speed Over Time (Timeseries)

1. Add Panel → Visualization: Time series
2. Query:
   - Type: JSON
   - URL: `/api/v1/results?limit=200`
   - Format: Timeseries
   - Columns:
     - `created_at` → Time (timestamp)
     - `download_mbps` → Download (number)
     - `upload_mbps` → Upload (number)
     - `connection_name` → Connection (string)

## Troubleshooting

### "No data" displayed

1. Check if FlowGauge is running: `curl http://localhost:8080/health`
2. Check if data exists: `curl http://localhost:8080/api/v1/results/latest`
3. Verify datasource URL in Grafana

### Incorrect time format

If timestamps are not recognized, set in the query:
- Column Type for `created_at`: `timestamp`

### CORS errors

FlowGauge has CORS enabled by default. If issues occur:
- Use `access: proxy` in the datasource (not `direct`)
