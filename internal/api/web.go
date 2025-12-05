package api

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/lan-dot-party/flowgauge/internal/storage"
	"github.com/lan-dot-party/flowgauge/pkg/version"
)

// DashboardData contains all data for the dashboard template.
type DashboardData struct {
	Version     string
	Connections []ConnectionData
	LastUpdate  string
}

// ConnectionData contains connection info with latest result and chart data.
type ConnectionData struct {
	Name         string
	SourceIP     string
	DSCP         int
	Enabled      bool
	LatestResult *storage.TestResult
	ChartData    ChartData
}

// ChartData contains data for the charts.
type ChartData struct {
	Labels   []string  `json:"labels"`
	Download []float64 `json:"download"`
	Upload   []float64 `json:"upload"`
	Latency  []float64 `json:"latency"`
}

// handleDashboard serves the main dashboard page.
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data := s.getDashboardData(r.Context(), 2*time.Hour) // Default: 2h for mini charts
	
	funcMap := template.FuncMap{
		"json": jsonFunc,
	}
	
	tmpl := template.Must(template.New("dashboard").Funcs(funcMap).Parse(dashboardTemplate))
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		s.logger.Error("Failed to render dashboard", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleDashboardPartial returns dashboard cards as HTML (for HTMX updates).
func (s *Server) handleDashboardPartial(w http.ResponseWriter, r *http.Request) {
	data := s.getDashboardData(r.Context(), 2*time.Hour)
	
	funcMap := template.FuncMap{
		"json": jsonFunc,
	}
	
	tmpl := template.Must(template.New("cards").Funcs(funcMap).Parse(dashboardCardsTemplate))
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleConnectionChartData returns chart data for a specific connection.
func (s *Server) handleConnectionChartData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	connectionName := chi.URLParam(r, "name")
	
	// Parse duration from query param (default: 24h for modal)
	durationStr := r.URL.Query().Get("duration")
	duration := 24 * time.Hour
	if durationStr != "" {
		if d, err := time.ParseDuration(durationStr); err == nil {
			duration = d
		}
	}
	
	chartData := s.getConnectionChartData(ctx, connectionName, duration)
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chartData); err != nil {
		s.logger.Error("Failed to encode chart data", zap.Error(err))
	}
}

// getConnectionChartData fetches chart data for a specific connection.
func (s *Server) getConnectionChartData(ctx context.Context, connectionName string, duration time.Duration) ChartData {
	filter := storage.ResultFilter{
		ConnectionName: connectionName,
		Since:          time.Now().Add(-duration),
		Limit:          200,
	}
	
	results, _ := s.storage.GetResults(ctx, filter)
	
	chartData := ChartData{
		Labels:   make([]string, 0, len(results)),
		Download: make([]float64, 0, len(results)),
		Upload:   make([]float64, 0, len(results)),
		Latency:  make([]float64, 0, len(results)),
	}
	
	// Reverse order for chronological display
	for i := len(results) - 1; i >= 0; i-- {
		r := results[i]
		if r.Error == "" {
			chartData.Labels = append(chartData.Labels, r.CreatedAt.Local().Format("15:04"))
			chartData.Download = append(chartData.Download, r.DownloadMbps)
			chartData.Upload = append(chartData.Upload, r.UploadMbps)
			chartData.Latency = append(chartData.Latency, r.LatencyMs)
		}
	}
	
	return chartData
}

// getDashboardData collects all data needed for the dashboard.
func (s *Server) getDashboardData(ctx context.Context, chartDuration time.Duration) DashboardData {
	data := DashboardData{
		Version:    version.GetShortVersion(),
		LastUpdate: time.Now().Local().Format("15:04:05"),
	}
	
	// Get latest results
	latestResults, _ := s.storage.GetLatestResults(ctx)
	
	// Build map for quick lookup
	latestMap := make(map[string]*storage.TestResult)
	for i := range latestResults {
		latestMap[latestResults[i].ConnectionName] = &latestResults[i]
	}
	
	// Build connection data with chart data for each
	for _, conn := range s.fullConfig.Connections {
		connData := ConnectionData{
			Name:      conn.Name,
			SourceIP:  conn.SourceIP,
			DSCP:      conn.DSCP,
			Enabled:   conn.Enabled,
			ChartData: s.getConnectionChartData(ctx, conn.Name, chartDuration),
		}
		if result, ok := latestMap[conn.Name]; ok {
			connData.LatestResult = result
		}
		data.Connections = append(data.Connections, connData)
	}
	
	return data
}

const dashboardCardsTemplate = `
{{range $idx, $conn := .Connections}}
<div class="connection-card {{if not $conn.Enabled}}disabled{{end}}" data-connection="{{$conn.Name}}">
    <div class="card-header">
        <span class="connection-name">{{$conn.Name}}</span>
        {{if $conn.Enabled}}<span class="status-badge active">Active</span>{{else}}<span class="status-badge">Disabled</span>{{end}}
    </div>
    {{if $conn.LatestResult}}
    <div class="metrics-row">
        <div class="metric">
            <span class="metric-value download">{{printf "%.1f" $conn.LatestResult.DownloadMbps}}</span>
            <span class="metric-label">↓ Mbps</span>
        </div>
        <div class="metric">
            <span class="metric-value upload">{{printf "%.1f" $conn.LatestResult.UploadMbps}}</span>
            <span class="metric-label">↑ Mbps</span>
        </div>
        <div class="metric">
            <span class="metric-value latency">{{printf "%.0f" $conn.LatestResult.LatencyMs}}</span>
            <span class="metric-label">ms</span>
        </div>
    </div>
    <div class="mini-chart-container" onclick="openModal('{{$conn.Name}}')">
        <canvas id="chart-{{$idx}}"></canvas>
        <div class="chart-overlay">
            <span>🔍 Click to expand</span>
        </div>
    </div>
    <div class="card-footer">
        <span class="server-info">{{$conn.LatestResult.ServerName}}</span>
        <span class="timestamp">{{$conn.LatestResult.CreatedAt.Local.Format "15:04"}}</span>
    </div>
    {{else}}
    <div class="card-body empty">
        <p>No test results yet</p>
    </div>
    {{end}}
</div>
{{end}}
`

const dashboardTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>FlowGauge Dashboard</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.1/dist/chart.umd.min.js"></script>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;700&family=Space+Grotesk:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-dark: #0a0a0f;
            --bg-card: #12121a;
            --bg-card-hover: #1a1a25;
            --bg-modal: rgba(0, 0, 0, 0.9);
            --text-primary: #e4e4e7;
            --text-secondary: #a1a1aa;
            --text-muted: #71717a;
            --accent-cyan: #06b6d4;
            --accent-green: #10b981;
            --accent-amber: #f59e0b;
            --accent-rose: #f43f5e;
            --accent-violet: #8b5cf6;
            --border: #27272a;
            --download-color: #10b981;
            --upload-color: #06b6d4;
            --latency-color: #f59e0b;
            --glow-green: 0 0 20px rgba(16, 185, 129, 0.3);
            --glow-cyan: 0 0 20px rgba(6, 182, 212, 0.3);
        }
        
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            font-family: 'Space Grotesk', -apple-system, BlinkMacSystemFont, sans-serif;
            background: var(--bg-dark);
            color: var(--text-primary);
            min-height: 100vh;
            background-image: 
                radial-gradient(ellipse at top, rgba(6, 182, 212, 0.1) 0%, transparent 50%),
                radial-gradient(ellipse at bottom right, rgba(139, 92, 246, 0.05) 0%, transparent 50%);
        }
        
        .container {
            max-width: 1600px;
            margin: 0 auto;
            padding: 2rem;
        }
        
        header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 2.5rem;
            padding-bottom: 1.5rem;
            border-bottom: 1px solid var(--border);
        }
        
        .logo {
            display: flex;
            align-items: center;
            gap: 1rem;
        }
        
        .logo-icon {
            font-size: 2.5rem;
            filter: drop-shadow(var(--glow-cyan));
        }
        
        .logo h1 {
            font-size: 1.75rem;
            font-weight: 700;
            background: linear-gradient(135deg, var(--accent-cyan), var(--accent-violet));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }
        
        .logo .version {
            background: linear-gradient(135deg, var(--accent-cyan), var(--accent-violet));
            color: white;
            padding: 0.25rem 0.75rem;
            border-radius: 2rem;
            font-size: 0.75rem;
            font-weight: 600;
            font-family: 'JetBrains Mono', monospace;
        }
        
        .header-info {
            display: flex;
            align-items: center;
            gap: 1.5rem;
            color: var(--text-secondary);
            font-size: 0.875rem;
        }
        
        .update-indicator {
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }
        
        .pulse {
            width: 8px;
            height: 8px;
            background: var(--accent-green);
            border-radius: 50%;
            box-shadow: var(--glow-green);
            animation: pulse 2s infinite;
        }
        
        @keyframes pulse {
            0%, 100% { opacity: 1; transform: scale(1); }
            50% { opacity: 0.6; transform: scale(0.9); }
        }
        
        .connections-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
            gap: 1.5rem;
            margin-bottom: 2rem;
        }
        
        .connection-card {
            background: var(--bg-card);
            border-radius: 1rem;
            border: 1px solid var(--border);
            overflow: hidden;
            transition: all 0.3s ease;
        }
        
        .connection-card:hover {
            transform: translateY(-4px);
            border-color: var(--accent-cyan);
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4), var(--glow-cyan);
        }
        
        .connection-card.disabled {
            opacity: 0.4;
        }
        
        .card-header {
            padding: 1rem 1.5rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            border-bottom: 1px solid var(--border);
            background: linear-gradient(180deg, rgba(255,255,255,0.02) 0%, transparent 100%);
        }
        
        .connection-name {
            font-weight: 600;
            font-size: 1.125rem;
            font-family: 'JetBrains Mono', monospace;
        }
        
        .status-badge {
            padding: 0.25rem 0.75rem;
            border-radius: 2rem;
            font-size: 0.7rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            background: rgba(113, 113, 122, 0.2);
            color: var(--text-muted);
        }
        
        .status-badge.active {
            background: rgba(16, 185, 129, 0.15);
            color: var(--accent-green);
            box-shadow: inset 0 0 10px rgba(16, 185, 129, 0.1);
        }
        
        .metrics-row {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 1rem;
            padding: 1.25rem 1.5rem;
            background: linear-gradient(180deg, transparent 0%, rgba(0,0,0,0.2) 100%);
        }
        
        .metric {
            text-align: center;
        }
        
        .metric-value {
            font-size: 2rem;
            font-weight: 700;
            font-family: 'JetBrains Mono', monospace;
            display: block;
            line-height: 1;
        }
        
        .metric-label {
            font-size: 0.75rem;
            color: var(--text-muted);
            text-transform: uppercase;
            letter-spacing: 0.05em;
            margin-top: 0.25rem;
            display: block;
        }
        
        .metric-value.download { color: var(--download-color); text-shadow: var(--glow-green); }
        .metric-value.upload { color: var(--upload-color); text-shadow: var(--glow-cyan); }
        .metric-value.latency { color: var(--latency-color); }
        
        .mini-chart-container {
            position: relative;
            height: 120px;
            padding: 0.5rem 1rem;
            cursor: pointer;
            transition: all 0.3s ease;
        }
        
        .mini-chart-container:hover {
            background: rgba(6, 182, 212, 0.05);
        }
        
        .mini-chart-container:hover .chart-overlay {
            opacity: 1;
        }
        
        .chart-overlay {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            display: flex;
            align-items: center;
            justify-content: center;
            background: rgba(0, 0, 0, 0.6);
            opacity: 0;
            transition: opacity 0.3s ease;
            border-radius: 0.5rem;
        }
        
        .chart-overlay span {
            background: var(--accent-cyan);
            color: white;
            padding: 0.5rem 1rem;
            border-radius: 2rem;
            font-size: 0.875rem;
            font-weight: 500;
        }
        
        .card-footer {
            display: flex;
            justify-content: space-between;
            padding: 0.75rem 1.5rem;
            font-size: 0.75rem;
            color: var(--text-muted);
            border-top: 1px solid var(--border);
            background: rgba(0, 0, 0, 0.2);
        }
        
        .card-body.empty {
            padding: 3rem;
            text-align: center;
            color: var(--text-muted);
        }
        
        /* Modal Styles */
        .modal {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: var(--bg-modal);
            z-index: 1000;
            backdrop-filter: blur(10px);
            animation: fadeIn 0.3s ease;
        }
        
        .modal.active {
            display: flex;
            align-items: center;
            justify-content: center;
        }
        
        @keyframes fadeIn {
            from { opacity: 0; }
            to { opacity: 1; }
        }
        
        .modal-content {
            background: var(--bg-card);
            border: 1px solid var(--border);
            border-radius: 1.5rem;
            width: 90%;
            max-width: 1200px;
            max-height: 90vh;
            overflow: hidden;
            animation: slideUp 0.3s ease;
        }
        
        @keyframes slideUp {
            from { transform: translateY(20px); opacity: 0; }
            to { transform: translateY(0); opacity: 1; }
        }
        
        .modal-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 1.5rem 2rem;
            border-bottom: 1px solid var(--border);
        }
        
        .modal-title {
            font-size: 1.5rem;
            font-weight: 700;
            font-family: 'JetBrains Mono', monospace;
        }
        
        .modal-close {
            background: none;
            border: none;
            color: var(--text-secondary);
            font-size: 1.5rem;
            cursor: pointer;
            padding: 0.5rem;
            border-radius: 0.5rem;
            transition: all 0.2s ease;
        }
        
        .modal-close:hover {
            background: rgba(255, 255, 255, 0.1);
            color: var(--text-primary);
        }
        
        .modal-body {
            padding: 2rem;
        }
        
        .time-selector {
            display: flex;
            gap: 0.5rem;
            margin-bottom: 1.5rem;
        }
        
        .time-btn {
            background: var(--bg-dark);
            border: 1px solid var(--border);
            color: var(--text-secondary);
            padding: 0.5rem 1rem;
            border-radius: 0.5rem;
            cursor: pointer;
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.875rem;
            transition: all 0.2s ease;
        }
        
        .time-btn:hover {
            border-color: var(--accent-cyan);
            color: var(--text-primary);
        }
        
        .time-btn.active {
            background: var(--accent-cyan);
            border-color: var(--accent-cyan);
            color: white;
        }
        
        .modal-chart-container {
            height: 400px;
            position: relative;
        }
        
        .chart-legend {
            display: flex;
            justify-content: center;
            gap: 2rem;
            margin-top: 1rem;
            padding-top: 1rem;
            border-top: 1px solid var(--border);
        }
        
        .legend-item {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            font-size: 0.875rem;
            color: var(--text-secondary);
        }
        
        .legend-dot {
            width: 12px;
            height: 12px;
            border-radius: 50%;
        }
        
        .legend-dot.download { background: var(--download-color); box-shadow: 0 0 10px var(--download-color); }
        .legend-dot.upload { background: var(--upload-color); box-shadow: 0 0 10px var(--upload-color); }
        .legend-dot.latency { background: var(--latency-color); box-shadow: 0 0 10px var(--latency-color); }
        
        footer {
            text-align: center;
            padding: 2rem;
            color: var(--text-muted);
            font-size: 0.875rem;
        }
        
        footer a {
            color: var(--accent-cyan);
            text-decoration: none;
            transition: color 0.2s ease;
        }
        
        footer a:hover {
            color: var(--accent-violet);
        }
        
        @media (max-width: 768px) {
            .container { padding: 1rem; }
            header { flex-direction: column; gap: 1rem; text-align: center; }
            .connections-grid { grid-template-columns: 1fr; }
            .modal-content { width: 95%; border-radius: 1rem; }
            .time-selector { flex-wrap: wrap; justify-content: center; }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div class="logo">
                <span class="logo-icon">🌊</span>
                <h1>FlowGauge</h1>
                <span class="version">v{{.Version}}</span>
            </div>
            <div class="header-info">
                <div class="update-indicator">
                    <span class="pulse"></span>
                    <span>Live</span>
                </div>
                <span id="last-update">{{.LastUpdate}}</span>
            </div>
        </header>
        
        <div id="connections" class="connections-grid" 
             hx-get="/dashboard/cards" 
             hx-trigger="every 30s"
             hx-swap="innerHTML">
            {{range $idx, $conn := .Connections}}
            <div class="connection-card {{if not $conn.Enabled}}disabled{{end}}" data-connection="{{$conn.Name}}">
                <div class="card-header">
                    <span class="connection-name">{{$conn.Name}}</span>
                    {{if $conn.Enabled}}<span class="status-badge active">Active</span>{{else}}<span class="status-badge">Disabled</span>{{end}}
                </div>
                {{if $conn.LatestResult}}
                <div class="metrics-row">
                    <div class="metric">
                        <span class="metric-value download">{{printf "%.1f" $conn.LatestResult.DownloadMbps}}</span>
                        <span class="metric-label">↓ Mbps</span>
                    </div>
                    <div class="metric">
                        <span class="metric-value upload">{{printf "%.1f" $conn.LatestResult.UploadMbps}}</span>
                        <span class="metric-label">↑ Mbps</span>
                    </div>
                    <div class="metric">
                        <span class="metric-value latency">{{printf "%.0f" $conn.LatestResult.LatencyMs}}</span>
                        <span class="metric-label">ms</span>
                    </div>
                </div>
                <div class="mini-chart-container" onclick="openModal('{{$conn.Name}}')">
                    <canvas id="chart-{{$idx}}"></canvas>
                    <div class="chart-overlay">
                        <span>🔍 Click to expand</span>
                    </div>
                </div>
                <div class="card-footer">
                    <span class="server-info">{{$conn.LatestResult.ServerName}}</span>
                    <span class="timestamp">{{$conn.LatestResult.CreatedAt.Local.Format "15:04"}}</span>
                </div>
                {{else}}
                <div class="card-body empty">
                    <p>No test results yet</p>
                </div>
                {{end}}
            </div>
            {{end}}
        </div>
        
        <footer>
            <p>FlowGauge v{{.Version}} • 
            <a href="/api/">API Documentation</a> • 
            <a href="https://github.com/lan-dot-party/flowgauge" target="_blank">GitHub</a></p>
        </footer>
    </div>
    
    <!-- Modal for expanded chart -->
    <div id="chart-modal" class="modal" onclick="closeModal(event)">
        <div class="modal-content" onclick="event.stopPropagation()">
            <div class="modal-header">
                <h2 class="modal-title" id="modal-title">Connection</h2>
                <button class="modal-close" onclick="closeModal()">&times;</button>
            </div>
            <div class="modal-body">
                <div class="time-selector">
                    <button class="time-btn" data-duration="1h">1h</button>
                    <button class="time-btn" data-duration="2h">2h</button>
                    <button class="time-btn" data-duration="6h">6h</button>
                    <button class="time-btn active" data-duration="24h">24h</button>
                    <button class="time-btn" data-duration="48h">48h</button>
                    <button class="time-btn" data-duration="168h">7d</button>
                </div>
                <div class="modal-chart-container">
                    <canvas id="modal-chart"></canvas>
                </div>
                <div class="chart-legend">
                    <div class="legend-item">
                        <span class="legend-dot download"></span>
                        <span>Download (Mbps)</span>
                    </div>
                    <div class="legend-item">
                        <span class="legend-dot upload"></span>
                        <span>Upload (Mbps)</span>
                    </div>
                    <div class="legend-item">
                        <span class="legend-dot latency"></span>
                        <span>Latency (ms)</span>
                    </div>
                </div>
            </div>
        </div>
    </div>
    
    <script>
        // Connection chart data from server
        const connectionData = {
            {{range $idx, $conn := .Connections}}
            "{{$conn.Name}}": {
                labels: {{$conn.ChartData.Labels | json}},
                download: {{$conn.ChartData.Download | json}},
                upload: {{$conn.ChartData.Upload | json}},
                latency: {{$conn.ChartData.Latency | json}}
            },
            {{end}}
        };
        
        // Mini chart configuration
        const miniChartConfig = {
            type: 'line',
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: { legend: { display: false }, tooltip: { enabled: false } },
                scales: {
                    x: { display: false },
                    y: { display: false }
                },
                elements: {
                    point: { radius: 0 },
                    line: { tension: 0.4, borderWidth: 2 }
                },
                animation: false
            }
        };
        
        // Create mini charts for each connection
        const miniCharts = {};
        {{range $idx, $conn := .Connections}}
        {{if $conn.LatestResult}}
        (function() {
            const ctx = document.getElementById('chart-{{$idx}}');
            if (ctx) {
                const data = connectionData["{{$conn.Name}}"];
                miniCharts["{{$conn.Name}}"] = new Chart(ctx, {
                    ...miniChartConfig,
                    data: {
                        labels: data.labels,
                        datasets: [
                            {
                                data: data.download,
                                borderColor: '#10b981',
                                backgroundColor: 'rgba(16, 185, 129, 0.1)',
                                fill: true
                            },
                            {
                                data: data.upload,
                                borderColor: '#06b6d4',
                                backgroundColor: 'transparent',
                                fill: false
                            }
                        ]
                    }
                });
            }
        })();
        {{end}}
        {{end}}
        
        // Modal chart
        let modalChart = null;
        let currentConnection = null;
        let currentDuration = '24h';
        
        function openModal(connectionName) {
            currentConnection = connectionName;
            document.getElementById('modal-title').textContent = connectionName;
            document.getElementById('chart-modal').classList.add('active');
            loadModalChart(connectionName, currentDuration);
        }
        
        function closeModal(event) {
            if (event && event.target !== event.currentTarget) return;
            document.getElementById('chart-modal').classList.remove('active');
        }
        
        async function loadModalChart(connectionName, duration) {
            try {
                const response = await fetch('/dashboard/connection/' + encodeURIComponent(connectionName) + '/chart?duration=' + duration);
                const data = await response.json();
                
                const ctx = document.getElementById('modal-chart');
                
                if (modalChart) {
                    modalChart.destroy();
                }
                
                modalChart = new Chart(ctx, {
                    type: 'line',
                    data: {
                        labels: data.labels,
                        datasets: [
                            {
                                label: 'Download (Mbps)',
                                data: data.download,
                                borderColor: '#10b981',
                                backgroundColor: 'rgba(16, 185, 129, 0.1)',
                                fill: true,
                                tension: 0.4,
                                yAxisID: 'y'
                            },
                            {
                                label: 'Upload (Mbps)',
                                data: data.upload,
                                borderColor: '#06b6d4',
                                backgroundColor: 'rgba(6, 182, 212, 0.1)',
                                fill: true,
                                tension: 0.4,
                                yAxisID: 'y'
                            },
                            {
                                label: 'Latency (ms)',
                                data: data.latency,
                                borderColor: '#f59e0b',
                                backgroundColor: 'transparent',
                                fill: false,
                                tension: 0.4,
                                yAxisID: 'y1'
                            }
                        ]
                    },
                    options: {
                        responsive: true,
                        maintainAspectRatio: false,
                        interaction: { mode: 'index', intersect: false },
                        plugins: {
                            legend: { display: false },
                            tooltip: {
                                backgroundColor: '#12121a',
                                titleColor: '#e4e4e7',
                                bodyColor: '#a1a1aa',
                                borderColor: '#27272a',
                                borderWidth: 1,
                                padding: 12,
                                displayColors: true
                            }
                        },
                        scales: {
                            x: {
                                grid: { color: 'rgba(39, 39, 42, 0.5)' },
                                ticks: { color: '#71717a', maxTicksLimit: 12 }
                            },
                            y: {
                                type: 'linear',
                                display: true,
                                position: 'left',
                                title: { display: true, text: 'Speed (Mbps)', color: '#71717a' },
                                grid: { color: 'rgba(39, 39, 42, 0.5)' },
                                ticks: { color: '#71717a' }
                            },
                            y1: {
                                type: 'linear',
                                display: true,
                                position: 'right',
                                title: { display: true, text: 'Latency (ms)', color: '#71717a' },
                                grid: { drawOnChartArea: false },
                                ticks: { color: '#71717a' }
                            }
                        }
                    }
                });
            } catch (e) {
                console.error('Failed to load chart data:', e);
            }
        }
        
        // Time selector buttons
        document.querySelectorAll('.time-btn').forEach(btn => {
            btn.addEventListener('click', function() {
                document.querySelectorAll('.time-btn').forEach(b => b.classList.remove('active'));
                this.classList.add('active');
                currentDuration = this.dataset.duration;
                if (currentConnection) {
                    loadModalChart(currentConnection, currentDuration);
                }
            });
        });
        
        // Close modal on Escape key
        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') closeModal();
        });
        
        // Update timestamp on HTMX refresh
        document.body.addEventListener('htmx:afterSwap', function(evt) {
            document.getElementById('last-update').textContent = new Date().toLocaleTimeString('de-DE', {hour: '2-digit', minute: '2-digit', second: '2-digit'});
            // Reinitialize mini charts after HTMX swap
            setTimeout(() => location.reload(), 100); // Simple reload for now
        });
        
        // Refresh mini charts periodically
        setInterval(async () => {
            for (const [name, chart] of Object.entries(miniCharts)) {
                try {
                    const response = await fetch('/dashboard/connection/' + encodeURIComponent(name) + '/chart?duration=2h');
                    const data = await response.json();
                    
                    chart.data.labels = data.labels;
                    chart.data.datasets[0].data = data.download;
                    chart.data.datasets[1].data = data.upload;
                    chart.update('none');
                } catch (e) {
                    console.error('Failed to update chart for', name, e);
                }
            }
        }, 60000);
    </script>
</body>
</html>`

// jsonFunc is a template function to convert data to JSON.
func jsonFunc(v interface{}) template.JS {
	b, _ := json.Marshal(v)
	return template.JS(b)
}
