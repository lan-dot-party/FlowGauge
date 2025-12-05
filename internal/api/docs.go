package api

import (
	"net/http"

	"github.com/lan-dot-party/flowgauge/pkg/version"
)

// handleAPIDocs serves the API documentation page.
func (s *Server) handleAPIDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>FlowGauge API Documentation (Read-Only)</title>
    <style>
        :root {
            --bg-primary: #0d1117;
            --bg-secondary: #161b22;
            --bg-tertiary: #21262d;
            --text-primary: #c9d1d9;
            --text-secondary: #8b949e;
            --accent: #58a6ff;
            --accent-green: #3fb950;
            --accent-yellow: #d29922;
            --accent-red: #f85149;
            --accent-purple: #a371f7;
            --border: #30363d;
        }
        
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Noto Sans', Helvetica, Arial, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            line-height: 1.6;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 2rem;
        }
        
        header {
            background: linear-gradient(135deg, var(--bg-secondary) 0%, var(--bg-tertiary) 100%);
            border-bottom: 1px solid var(--border);
            padding: 2rem 0;
            margin-bottom: 2rem;
        }
        
        header .container {
            display: flex;
            align-items: center;
            gap: 1rem;
        }
        
        .logo {
            font-size: 2rem;
        }
        
        h1 {
            font-size: 1.8rem;
            font-weight: 600;
        }
        
        .version {
            background: var(--accent);
            color: var(--bg-primary);
            padding: 0.25rem 0.75rem;
            border-radius: 20px;
            font-size: 0.85rem;
            font-weight: 600;
        }
        
        .base-url {
            background: var(--bg-tertiary);
            border: 1px solid var(--border);
            border-radius: 8px;
            padding: 1rem 1.5rem;
            margin-bottom: 2rem;
            display: flex;
            align-items: center;
            gap: 1rem;
        }
        
        .base-url code {
            background: var(--bg-primary);
            padding: 0.5rem 1rem;
            border-radius: 6px;
            font-family: 'SF Mono', Monaco, 'Courier New', monospace;
            color: var(--accent);
        }
        
        .endpoint-group {
            margin-bottom: 2rem;
        }
        
        .endpoint-group h2 {
            font-size: 1.2rem;
            color: var(--text-secondary);
            margin-bottom: 1rem;
            padding-bottom: 0.5rem;
            border-bottom: 1px solid var(--border);
        }
        
        .endpoint {
            background: var(--bg-secondary);
            border: 1px solid var(--border);
            border-radius: 8px;
            margin-bottom: 0.75rem;
            overflow: hidden;
        }
        
        .endpoint-header {
            display: flex;
            align-items: center;
            gap: 1rem;
            padding: 1rem 1.5rem;
            cursor: pointer;
            transition: background 0.2s;
        }
        
        .endpoint-header:hover {
            background: var(--bg-tertiary);
        }
        
        .method {
            font-weight: 700;
            font-size: 0.75rem;
            padding: 0.25rem 0.5rem;
            border-radius: 4px;
            min-width: 60px;
            text-align: center;
        }
        
        .method.get { background: var(--accent-green); color: #000; }
        .method.post { background: var(--accent-yellow); color: #000; }
        .method.put { background: var(--accent); color: #000; }
        .method.delete { background: var(--accent-red); color: #000; }
        
        .path {
            font-family: 'SF Mono', Monaco, 'Courier New', monospace;
            color: var(--text-primary);
        }
        
        .description {
            color: var(--text-secondary);
            margin-left: auto;
            font-size: 0.9rem;
        }
        
        .endpoint-details {
            display: none;
            padding: 1.5rem;
            border-top: 1px solid var(--border);
            background: var(--bg-tertiary);
        }
        
        .endpoint.open .endpoint-details {
            display: block;
        }
        
        .params-table {
            width: 100%;
            border-collapse: collapse;
            margin: 1rem 0;
        }
        
        .params-table th,
        .params-table td {
            text-align: left;
            padding: 0.75rem;
            border-bottom: 1px solid var(--border);
        }
        
        .params-table th {
            color: var(--text-secondary);
            font-weight: 500;
            font-size: 0.85rem;
        }
        
        .param-name {
            font-family: 'SF Mono', Monaco, 'Courier New', monospace;
            color: var(--accent-purple);
        }
        
        .param-type {
            color: var(--text-secondary);
            font-size: 0.85rem;
        }
        
        .try-it {
            margin-top: 1rem;
        }
        
        .try-it button {
            background: var(--accent);
            color: var(--bg-primary);
            border: none;
            padding: 0.75rem 1.5rem;
            border-radius: 6px;
            font-weight: 600;
            cursor: pointer;
            transition: opacity 0.2s;
        }
        
        .try-it button:hover {
            opacity: 0.9;
        }
        
        .response-box {
            margin-top: 1rem;
            background: var(--bg-primary);
            border: 1px solid var(--border);
            border-radius: 6px;
            overflow: hidden;
        }
        
        .response-header {
            background: var(--bg-secondary);
            padding: 0.5rem 1rem;
            font-size: 0.85rem;
            color: var(--text-secondary);
            border-bottom: 1px solid var(--border);
        }
        
        .response-body {
            padding: 1rem;
            font-family: 'SF Mono', Monaco, 'Courier New', monospace;
            font-size: 0.85rem;
            white-space: pre-wrap;
            max-height: 400px;
            overflow: auto;
        }
        
        .status-ok { color: var(--accent-green); }
        .status-error { color: var(--accent-red); }
        
        footer {
            text-align: center;
            padding: 2rem;
            color: var(--text-secondary);
            font-size: 0.9rem;
        }
        
        footer a {
            color: var(--accent);
            text-decoration: none;
        }
    </style>
</head>
<body>
    <header>
        <div class="container">
            <span class="logo">🌊</span>
            <h1>FlowGauge API</h1>
            <span class="version">v` + version.GetShortVersion() + `</span>
        </div>
    </header>
    
    <div class="container">
        <div class="base-url">
            <span>Base URL:</span>
            <code id="baseUrl"></code>
        </div>
        
        <div class="endpoint-group">
            <h2>🏥 Health</h2>
            
            <div class="endpoint" data-method="GET" data-path="/health">
                <div class="endpoint-header" onclick="toggleEndpoint(this)">
                    <span class="method get">GET</span>
                    <span class="path">/health</span>
                    <span class="description">Health check endpoint</span>
                </div>
                <div class="endpoint-details">
                    <p>Returns the server health status and version.</p>
                    <h4>Response</h4>
                    <pre class="response-box"><code>{"status": "ok", "version": "1.0.0"}</code></pre>
                    <div class="try-it">
                        <button onclick="tryEndpoint('GET', '/health')">Try it</button>
                        <div class="response-box" style="display:none">
                            <div class="response-header">Response <span class="status"></span></div>
                            <pre class="response-body"></pre>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        
        <div class="endpoint-group">
            <h2>📊 Results</h2>
            
            <div class="endpoint" data-method="GET" data-path="/api/v1/results">
                <div class="endpoint-header" onclick="toggleEndpoint(this)">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/results</span>
                    <span class="description">List speedtest results</span>
                </div>
                <div class="endpoint-details">
                    <p>Returns a list of speedtest results with optional filtering.</p>
                    <h4>Query Parameters</h4>
                    <table class="params-table">
                        <tr><th>Name</th><th>Type</th><th>Description</th></tr>
                        <tr><td class="param-name">connection</td><td class="param-type">string</td><td>Filter by connection name</td></tr>
                        <tr><td class="param-name">since</td><td class="param-type">string</td><td>Filter results since (RFC3339 or duration like "24h")</td></tr>
                        <tr><td class="param-name">until</td><td class="param-type">string</td><td>Filter results until (RFC3339)</td></tr>
                        <tr><td class="param-name">limit</td><td class="param-type">integer</td><td>Maximum results (default: 100)</td></tr>
                        <tr><td class="param-name">offset</td><td class="param-type">integer</td><td>Offset for pagination</td></tr>
                    </table>
                    <div class="try-it">
                        <button onclick="tryEndpoint('GET', '/api/v1/results?limit=5')">Try it</button>
                        <div class="response-box" style="display:none">
                            <div class="response-header">Response <span class="status"></span></div>
                            <pre class="response-body"></pre>
                        </div>
                    </div>
                </div>
            </div>
            
            <div class="endpoint" data-method="GET" data-path="/api/v1/results/latest">
                <div class="endpoint-header" onclick="toggleEndpoint(this)">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/results/latest</span>
                    <span class="description">Get latest result per connection</span>
                </div>
                <div class="endpoint-details">
                    <p>Returns the most recent speedtest result for each configured connection.</p>
                    <div class="try-it">
                        <button onclick="tryEndpoint('GET', '/api/v1/results/latest')">Try it</button>
                        <div class="response-box" style="display:none">
                            <div class="response-header">Response <span class="status"></span></div>
                            <pre class="response-body"></pre>
                        </div>
                    </div>
                </div>
            </div>
            
            <div class="endpoint" data-method="GET" data-path="/api/v1/results/{id}">
                <div class="endpoint-header" onclick="toggleEndpoint(this)">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/results/{id}</span>
                    <span class="description">Get a specific result</span>
                </div>
                <div class="endpoint-details">
                    <p>Returns a single speedtest result by ID.</p>
                    <h4>Path Parameters</h4>
                    <table class="params-table">
                        <tr><th>Name</th><th>Type</th><th>Description</th></tr>
                        <tr><td class="param-name">id</td><td class="param-type">integer</td><td>Result ID</td></tr>
                    </table>
                    <div class="try-it">
                        <button onclick="tryEndpoint('GET', '/api/v1/results/1')">Try it (ID: 1)</button>
                        <div class="response-box" style="display:none">
                            <div class="response-header">Response <span class="status"></span></div>
                            <pre class="response-body"></pre>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        
        <div class="endpoint-group">
            <h2>🔌 Connections</h2>
            
            <div class="endpoint" data-method="GET" data-path="/api/v1/connections">
                <div class="endpoint-header" onclick="toggleEndpoint(this)">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/connections</span>
                    <span class="description">List all connections</span>
                </div>
                <div class="endpoint-details">
                    <p>Returns all configured network connections.</p>
                    <div class="try-it">
                        <button onclick="tryEndpoint('GET', '/api/v1/connections')">Try it</button>
                        <div class="response-box" style="display:none">
                            <div class="response-header">Response <span class="status"></span></div>
                            <pre class="response-body"></pre>
                        </div>
                    </div>
                </div>
            </div>
            
            <div class="endpoint" data-method="GET" data-path="/api/v1/connections/{name}/stats">
                <div class="endpoint-header" onclick="toggleEndpoint(this)">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/connections/{name}/stats</span>
                    <span class="description">Get connection statistics</span>
                </div>
                <div class="endpoint-details">
                    <p>Returns aggregated statistics for a specific connection.</p>
                    <h4>Path Parameters</h4>
                    <table class="params-table">
                        <tr><th>Name</th><th>Type</th><th>Description</th></tr>
                        <tr><td class="param-name">name</td><td class="param-type">string</td><td>Connection name</td></tr>
                    </table>
                    <h4>Query Parameters</h4>
                    <table class="params-table">
                        <tr><th>Name</th><th>Type</th><th>Description</th></tr>
                        <tr><td class="param-name">period</td><td class="param-type">string</td><td>Time period (e.g., "24h", "7d", "30d")</td></tr>
                    </table>
                    <div class="try-it">
                        <button onclick="tryEndpoint('GET', '/api/v1/connections/WAN1-Primary/stats?period=24h')">Try it</button>
                        <div class="response-box" style="display:none">
                            <div class="response-header">Response <span class="status"></span></div>
                            <pre class="response-body"></pre>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        
        <div class="endpoint-group">
            <h2>📈 Metrics</h2>
            
            <div class="endpoint" data-method="GET" data-path="/api/v1/metrics">
                <div class="endpoint-header" onclick="toggleEndpoint(this)">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/metrics</span>
                    <span class="description">Prometheus metrics</span>
                </div>
                <div class="endpoint-details">
                    <p>Returns Prometheus-formatted metrics for monitoring integration.</p>
                    <h4>Available Metrics</h4>
                    <table class="params-table">
                        <tr><th>Metric</th><th>Type</th><th>Description</th></tr>
                        <tr><td class="param-name">flowgauge_download_speed_mbps</td><td class="param-type">gauge</td><td>Download speed in Mbps</td></tr>
                        <tr><td class="param-name">flowgauge_upload_speed_mbps</td><td class="param-type">gauge</td><td>Upload speed in Mbps</td></tr>
                        <tr><td class="param-name">flowgauge_latency_ms</td><td class="param-type">gauge</td><td>Latency in milliseconds</td></tr>
                        <tr><td class="param-name">flowgauge_jitter_ms</td><td class="param-type">gauge</td><td>Jitter in milliseconds</td></tr>
                        <tr><td class="param-name">flowgauge_tests_total</td><td class="param-type">counter</td><td>Total tests run</td></tr>
                        <tr><td class="param-name">flowgauge_test_errors_total</td><td class="param-type">counter</td><td>Total test errors</td></tr>
                    </table>
                    <div class="try-it">
                        <button onclick="window.open('/api/v1/metrics', '_blank')">Open Metrics</button>
                    </div>
                </div>
            </div>
        </div>
    </div>
    
    <footer>
        <p>FlowGauge API v` + version.GetShortVersion() + ` • 
        <a href="https://github.com/lan-dot-party/flowgauge" target="_blank">GitHub</a></p>
    </footer>
    
    <script>
        // Set base URL
        document.getElementById('baseUrl').textContent = window.location.origin;
        
        function toggleEndpoint(header) {
            header.parentElement.classList.toggle('open');
        }
        
        async function tryEndpoint(method, path) {
            const button = event.target;
            const responseBox = button.parentElement.querySelector('.response-box');
            const statusEl = responseBox.querySelector('.status');
            const bodyEl = responseBox.querySelector('.response-body');
            
            responseBox.style.display = 'block';
            statusEl.textContent = 'Loading...';
            statusEl.className = 'status';
            bodyEl.textContent = '';
            
            try {
                const response = await fetch(path, { method });
                const data = await response.text();
                
                statusEl.textContent = response.status + ' ' + response.statusText;
                statusEl.className = 'status ' + (response.ok ? 'status-ok' : 'status-error');
                
                try {
                    bodyEl.textContent = JSON.stringify(JSON.parse(data), null, 2);
                } catch {
                    bodyEl.textContent = data;
                }
            } catch (err) {
                statusEl.textContent = 'Error';
                statusEl.className = 'status status-error';
                bodyEl.textContent = err.message;
            }
        }
    </script>
</body>
</html>`

	_, _ = w.Write([]byte(html))
}

// handleAPIRedirect redirects /api to the docs.
func (s *Server) handleAPIRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/api/", http.StatusMovedPermanently)
}


