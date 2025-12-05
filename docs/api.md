# FlowGauge REST API Documentation

> Version: 1.0  
> Base URL: `http://localhost:8080`

FlowGauge provides a read-only REST API for accessing speedtest results, connection statistics, and Prometheus metrics. The API is designed for integration with monitoring tools like Grafana.

---

## Table of Contents

- [Authentication](#authentication)
- [Response Format](#response-format)
- [Endpoints](#endpoints)
  - [Health Check](#health-check)
  - [Results](#results)
  - [Connections](#connections)
  - [Metrics](#metrics)
- [Filtering & Pagination](#filtering--pagination)
- [Error Handling](#error-handling)
- [Examples](#examples)

---

## Authentication

Authentication is optional and can be enabled in the configuration:

```yaml
webserver:
  auth:
    username: admin
    password: your-secure-password
```

When enabled, all endpoints (except `/health`) require HTTP Basic Authentication.

---

## Response Format

### Success Response

All successful API responses follow this format:

```json
{
  "status": "ok",
  "data": { ... }
}
```

### Error Response

```json
{
  "error": "Not Found",
  "code": 404,
  "message": "Result not found"
}
```

---

## Endpoints

### Health Check

#### `GET /health`

Returns the server health status and version. No authentication required.

**Response:**

```json
{
  "status": "ok",
  "version": "1.0.0"
}
```

**Status Codes:**
- `200 OK` - Server is healthy

---

### Results

#### `GET /api/v1/results`

Returns a list of speedtest results with optional filtering and pagination.

**Query Parameters:**

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `connection` | string | Filter by connection name | - |
| `since` | string | Results since (RFC3339 or duration like `24h`, `7d`) | - |
| `until` | string | Results until (RFC3339 format) | - |
| `limit` | integer | Maximum number of results | 100 |
| `offset` | integer | Offset for pagination | 0 |

**Example Request:**

```bash
curl "http://localhost:8080/api/v1/results?connection=WAN1-Primary&since=24h&limit=10"
```

**Response:**

```json
{
  "results": [
    {
      "id": 142,
      "connection_name": "WAN1-Primary",
      "server_id": 12345,
      "server_name": "Telekom Frankfurt",
      "server_country": "Germany",
      "server_host": "speedtest.telekom.de:8080",
      "latency_ms": 12.5,
      "jitter_ms": 2.1,
      "download_mbps": 245.67,
      "upload_mbps": 48.23,
      "packet_loss_pct": 0,
      "source_ip": "192.168.1.100",
      "dscp": 0,
      "created_at": "2024-01-15T14:30:00Z"
    }
  ],
  "meta": {
    "total": 1,
    "limit": 10,
    "offset": 0
  }
}
```

---

#### `GET /api/v1/results/latest`

Returns the most recent speedtest result for each configured connection.

**Example Request:**

```bash
curl "http://localhost:8080/api/v1/results/latest"
```

**Response:**

```json
{
  "status": "ok",
  "data": [
    {
      "id": 142,
      "connection_name": "WAN1-Primary",
      "server_name": "Telekom Frankfurt",
      "latency_ms": 12.5,
      "download_mbps": 245.67,
      "upload_mbps": 48.23,
      "created_at": "2024-01-15T14:30:00Z"
    },
    {
      "id": 143,
      "connection_name": "WAN2-Backup",
      "server_name": "Vodafone Berlin",
      "latency_ms": 18.2,
      "download_mbps": 98.45,
      "upload_mbps": 24.12,
      "created_at": "2024-01-15T14:31:00Z"
    }
  ]
}
```

---

#### `GET /api/v1/results/{id}`

Returns a single speedtest result by ID.

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | integer | Result ID |

**Example Request:**

```bash
curl "http://localhost:8080/api/v1/results/142"
```

**Response:**

```json
{
  "status": "ok",
  "data": {
    "id": 142,
    "connection_name": "WAN1-Primary",
    "server_id": 12345,
    "server_name": "Telekom Frankfurt",
    "server_country": "Germany",
    "server_host": "speedtest.telekom.de:8080",
    "latency_ms": 12.5,
    "jitter_ms": 2.1,
    "download_mbps": 245.67,
    "upload_mbps": 48.23,
    "packet_loss_pct": 0,
    "source_ip": "192.168.1.100",
    "dscp": 0,
    "created_at": "2024-01-15T14:30:00Z"
  }
}
```

**Status Codes:**
- `200 OK` - Result found
- `404 Not Found` - Result with given ID does not exist

---

### Connections

#### `GET /api/v1/connections`

Returns all configured network connections.

**Example Request:**

```bash
curl "http://localhost:8080/api/v1/connections"
```

**Response:**

```json
{
  "status": "ok",
  "data": [
    {
      "name": "WAN1-Primary",
      "source_ip": "192.168.1.100",
      "dscp": 0,
      "enabled": true
    },
    {
      "name": "WAN2-Backup",
      "source_ip": "192.168.2.100",
      "dscp": 46,
      "enabled": true
    }
  ]
}
```

**Connection Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Connection identifier |
| `source_ip` | string | Source IP address for binding |
| `dscp` | integer | DSCP value for QoS marking (0-63) |
| `enabled` | boolean | Whether the connection is active |

---

#### `GET /api/v1/connections/{name}/stats`

Returns aggregated statistics for a specific connection over a time period.

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `name` | string | Connection name |

**Query Parameters:**

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `period` | string | Time period (e.g., `1h`, `24h`, `7d`, `30d`) | `24h` |

**Example Request:**

```bash
curl "http://localhost:8080/api/v1/connections/WAN1-Primary/stats?period=7d"
```

**Response:**

```json
{
  "status": "ok",
  "data": {
    "connection_name": "WAN1-Primary",
    "avg_download_mbps": 238.45,
    "avg_upload_mbps": 45.67,
    "avg_latency_ms": 14.2,
    "min_download_mbps": 180.23,
    "max_download_mbps": 265.89,
    "min_upload_mbps": 38.12,
    "max_upload_mbps": 52.34,
    "min_latency_ms": 10.5,
    "max_latency_ms": 28.9,
    "test_count": 336,
    "error_count": 2,
    "period": 604800000000000,
    "since": "2024-01-08T14:30:00Z",
    "until": "2024-01-15T14:30:00Z"
  }
}
```

**Statistics Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `avg_download_mbps` | float | Average download speed |
| `avg_upload_mbps` | float | Average upload speed |
| `avg_latency_ms` | float | Average latency |
| `min_*` / `max_*` | float | Min/max values for each metric |
| `test_count` | integer | Total number of tests |
| `error_count` | integer | Number of failed tests |
| `period` | integer | Period in nanoseconds |
| `since` / `until` | string | Time range (RFC3339) |

---

### Metrics

#### `GET /api/v1/metrics`

Returns Prometheus-formatted metrics for monitoring integration.

**Example Request:**

```bash
curl "http://localhost:8080/api/v1/metrics"
```

**Response:**

```
# HELP flowgauge_download_speed_mbps Current download speed in Mbps
# TYPE flowgauge_download_speed_mbps gauge
flowgauge_download_speed_mbps{connection="WAN1-Primary"} 245.67
flowgauge_download_speed_mbps{connection="WAN2-Backup"} 98.45

# HELP flowgauge_upload_speed_mbps Current upload speed in Mbps
# TYPE flowgauge_upload_speed_mbps gauge
flowgauge_upload_speed_mbps{connection="WAN1-Primary"} 48.23
flowgauge_upload_speed_mbps{connection="WAN2-Backup"} 24.12

# HELP flowgauge_latency_ms Current latency in milliseconds
# TYPE flowgauge_latency_ms gauge
flowgauge_latency_ms{connection="WAN1-Primary"} 12.5
flowgauge_latency_ms{connection="WAN2-Backup"} 18.2

# HELP flowgauge_jitter_ms Current jitter in milliseconds
# TYPE flowgauge_jitter_ms gauge
flowgauge_jitter_ms{connection="WAN1-Primary"} 2.1
flowgauge_jitter_ms{connection="WAN2-Backup"} 3.4

# HELP flowgauge_tests_total Total number of speedtests run
# TYPE flowgauge_tests_total counter
flowgauge_tests_total{connection="WAN1-Primary"} 1842
flowgauge_tests_total{connection="WAN2-Backup"} 1840

# HELP flowgauge_test_errors_total Total number of failed speedtests
# TYPE flowgauge_test_errors_total counter
flowgauge_test_errors_total{connection="WAN1-Primary"} 5
flowgauge_test_errors_total{connection="WAN2-Backup"} 12
```

**Available Metrics:**

| Metric | Type | Description |
|--------|------|-------------|
| `flowgauge_download_speed_mbps` | Gauge | Current download speed |
| `flowgauge_upload_speed_mbps` | Gauge | Current upload speed |
| `flowgauge_latency_ms` | Gauge | Current latency |
| `flowgauge_jitter_ms` | Gauge | Current jitter |
| `flowgauge_tests_total` | Counter | Total tests run |
| `flowgauge_test_errors_total` | Counter | Total test errors |

All metrics include a `connection` label identifying the WAN connection.

---

## Filtering & Pagination

### Time-based Filtering

The `since` parameter supports two formats:

1. **RFC3339 timestamp**: `2024-01-15T00:00:00Z`
2. **Duration string**: `1h`, `24h`, `7d`, `30d`

Examples:
```bash
# Results from the last 24 hours
curl "http://localhost:8080/api/v1/results?since=24h"

# Results since a specific date
curl "http://localhost:8080/api/v1/results?since=2024-01-01T00:00:00Z"

# Results between two dates
curl "http://localhost:8080/api/v1/results?since=2024-01-01T00:00:00Z&until=2024-01-15T00:00:00Z"
```

### Pagination

Use `limit` and `offset` for pagination:

```bash
# First page (results 1-50)
curl "http://localhost:8080/api/v1/results?limit=50&offset=0"

# Second page (results 51-100)
curl "http://localhost:8080/api/v1/results?limit=50&offset=50"
```

---

## Error Handling

### HTTP Status Codes

| Code | Description |
|------|-------------|
| `200` | Success |
| `400` | Bad Request - Invalid parameters |
| `401` | Unauthorized - Authentication required |
| `404` | Not Found - Resource does not exist |
| `500` | Internal Server Error |

### Error Response Format

```json
{
  "error": "Bad Request",
  "code": 400,
  "message": "Invalid result ID"
}
```

---

## Examples

### Grafana Integration

Use the JSON API datasource in Grafana to query FlowGauge:

**Data Source Configuration:**
- URL: `http://flowgauge:8080`
- Authentication: Basic Auth (if enabled)

**Example Query (Table Panel):**
```
GET /api/v1/results?since=24h&limit=100
```

**JSONPath for Download Speed:**
```
$.results[*].download_mbps
```

### Prometheus/Grafana Stack

Add FlowGauge as a Prometheus scrape target:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'flowgauge'
    static_configs:
      - targets: ['flowgauge:8080']
    metrics_path: '/api/v1/metrics'
```

### Shell Script Example

```bash
#!/bin/bash
# Get latest download speed for WAN1

RESULT=$(curl -s "http://localhost:8080/api/v1/results/latest")
DOWNLOAD=$(echo "$RESULT" | jq -r '.data[] | select(.connection_name=="WAN1-Primary") | .download_mbps')

echo "WAN1 Download: ${DOWNLOAD} Mbps"

# Alert if below threshold
if (( $(echo "$DOWNLOAD < 100" | bc -l) )); then
    echo "WARNING: Download speed below 100 Mbps!"
fi
```

### Python Example

```python
import requests

BASE_URL = "http://localhost:8080"

# Get latest results
response = requests.get(f"{BASE_URL}/api/v1/results/latest")
data = response.json()

for result in data["data"]:
    print(f"{result['connection_name']}: {result['download_mbps']:.1f} Mbps down, "
          f"{result['upload_mbps']:.1f} Mbps up, {result['latency_ms']:.1f} ms latency")
```

---

## Interactive Documentation

FlowGauge includes an interactive API documentation page accessible at:

```
http://localhost:8080/api/
```

This page allows you to explore and test all API endpoints directly in your browser.

---

## Rate Limiting

Currently, FlowGauge does not implement rate limiting. For production deployments with public access, consider using a reverse proxy (nginx, Traefik) with rate limiting enabled.

---

## CORS

CORS is enabled by default, allowing requests from any origin. This can be restricted in future versions through configuration.

---

*For more information, see the [main documentation](../README.md) or the [Grafana integration guide](../grafana/README.md).*

