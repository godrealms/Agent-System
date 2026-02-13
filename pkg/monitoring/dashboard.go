package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"AI-agent/internal/config"
)

// Monitor tracks agent execution metrics and progress
type Monitor struct {
	config      *config.Config
	metrics     *Metrics
	server      *http.Server
	mu          sync.RWMutex
	isRunning   bool
	subscribers map[string]chan MetricsUpdate
}

// Metrics holds various monitoring metrics
type Metrics struct {
	TotalSessions      int             `json:"total_sessions"`
	SuccessfulSessions int             `json:"successful_sessions"`
	FailedSessions     int             `json:"failed_sessions"`
	ActiveSessions     int             `json:"active_sessions"`
	TotalFeatures      int             `json:"total_features"`
	CompletedFeatures  int             `json:"completed_features"`
	PendingFeatures    int             `json:"pending_features"`
	AvgSessionDuration time.Duration   `json:"avg_session_duration"`
	TotalTokensUsed    int             `json:"total_tokens_used"`
	StartTime          time.Time       `json:"start_time"`
	LastUpdateTime     time.Time       `json:"last_update_time"`
	SessionHistory     []SessionRecord `json:"session_history"`
}

// SessionRecord tracks individual session information
type SessionRecord struct {
	SessionID         string        `json:"session_id"`
	StartTime         time.Time     `json:"start_time"`
	EndTime           time.Time     `json:"end_time"`
	Duration          time.Duration `json:"duration"`
	Success           bool          `json:"success"`
	FeaturesCompleted []string      `json:"features_completed"`
	TokensUsed        int           `json:"tokens_used"`
	Error             string        `json:"error,omitempty"`
}

// MetricsUpdate represents a metrics update notification
type MetricsUpdate struct {
	Timestamp time.Time `json:"timestamp"`
	Metrics   Metrics   `json:"metrics"`
}

// NewMonitor creates a new monitor instance
func NewMonitor(cfg *config.Config) *Monitor {
	return &Monitor{
		config:      cfg,
		metrics:     &Metrics{StartTime: time.Now()},
		subscribers: make(map[string]chan MetricsUpdate),
	}
}

// Start begins monitoring and starts the web dashboard
func (m *Monitor) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return nil
	}

	m.isRunning = true

	// Start HTTP server for dashboard
	if m.config.MonitoringEnabled {
		go m.startDashboardServer()
	}

	log.Printf("Monitoring started on port %d", m.config.MonitoringPort)
	return nil
}

// Stop stops monitoring
func (m *Monitor) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return nil
	}

	m.isRunning = false

	// Stop HTTP server
	if m.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.server.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down monitoring server: %v", err)
		}
	}

	// Close all subscriber channels
	for _, ch := range m.subscribers {
		close(ch)
	}
	m.subscribers = make(map[string]chan MetricsUpdate)

	log.Println("Monitoring stopped")
	return nil
}

// RecordSession records a completed session
func (m *Monitor) RecordSession(sessionID string, success bool, duration time.Duration,
	featuresCompleted []string, tokensUsed int, err error) {

	m.mu.Lock()
	defer m.mu.Unlock()

	record := SessionRecord{
		SessionID:         sessionID,
		StartTime:         time.Now().Add(-duration),
		EndTime:           time.Now(),
		Duration:          duration,
		Success:           success,
		FeaturesCompleted: featuresCompleted,
		TokensUsed:        tokensUsed,
	}

	if err != nil {
		record.Error = err.Error()
	}

	m.metrics.SessionHistory = append(m.metrics.SessionHistory, record)

	// Update counters
	m.metrics.TotalSessions++
	if success {
		m.metrics.SuccessfulSessions++
		m.metrics.CompletedFeatures += len(featuresCompleted)
	} else {
		m.metrics.FailedSessions++
	}

	m.metrics.TotalTokensUsed += tokensUsed
	m.metrics.LastUpdateTime = time.Now()

	// Calculate average duration
	if m.metrics.TotalSessions > 0 {
		var totalDuration time.Duration
		for _, s := range m.metrics.SessionHistory {
			totalDuration += s.Duration
		}
		m.metrics.AvgSessionDuration = totalDuration / time.Duration(m.metrics.TotalSessions)
	}

	// Notify subscribers
	m.notifySubscribers()
}

// UpdateFeatures updates feature counts
func (m *Monitor) UpdateFeatures(total, completed, pending int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics.TotalFeatures = total
	m.metrics.CompletedFeatures = completed
	m.metrics.PendingFeatures = pending
	m.metrics.LastUpdateTime = time.Now()

	m.notifySubscribers()
}

// GetMetrics returns current metrics
func (m *Monitor) GetMetrics() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	metrics := *m.metrics
	sessionHistory := make([]SessionRecord, len(m.metrics.SessionHistory))
	copy(sessionHistory, m.metrics.SessionHistory)
	metrics.SessionHistory = sessionHistory

	return metrics
}

// Subscribe creates a subscription for metrics updates
func (m *Monitor) Subscribe(clientID string) <-chan MetricsUpdate {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan MetricsUpdate, 10) // Buffered channel
	m.subscribers[clientID] = ch

	// Send initial metrics
	select {
	case ch <- MetricsUpdate{
		Timestamp: time.Now(),
		Metrics:   m.GetMetrics(),
	}:
	default:
		// Channel is full, skip initial update
	}

	return ch
}

// Unsubscribe removes a subscription
func (m *Monitor) Unsubscribe(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ch, exists := m.subscribers[clientID]; exists {
		close(ch)
		delete(m.subscribers, clientID)
	}
}

// notifySubscribers sends updates to all subscribers
func (m *Monitor) notifySubscribers() {
	update := MetricsUpdate{
		Timestamp: time.Now(),
		Metrics:   m.GetMetrics(),
	}

	for clientID, ch := range m.subscribers {
		select {
		case ch <- update:
		default:
			// Channel is full, remove subscriber
			log.Printf("Removing slow subscriber: %s", clientID)
			close(ch)
			delete(m.subscribers, clientID)
		}
	}
}

// startDashboardServer starts the web dashboard server
func (m *Monitor) startDashboardServer() {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/metrics", m.handleMetrics)
	mux.HandleFunc("/api/sessions", m.handleSessions)
	mux.HandleFunc("/api/stream", m.handleStream)

	// Static files and dashboard
	mux.HandleFunc("/", m.handleDashboard)

	addr := fmt.Sprintf(":%d", m.config.MonitoringPort)
	m.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("Dashboard server starting on %s", addr)
	if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("Dashboard server error: %v", err)
	}
}

// handleMetrics returns current metrics as JSON
func (m *Monitor) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	metrics := m.GetMetrics()
	json.NewEncoder(w).Encode(metrics)
}

// handleSessions returns session history
func (m *Monitor) handleSessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	metrics := m.GetMetrics()
	json.NewEncoder(w).Encode(metrics.SessionHistory)
}

// handleStream provides server-sent events for real-time updates
func (m *Monitor) handleStream(w http.ResponseWriter, r *http.Request) {
	clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Subscribe to updates
	updates := m.Subscribe(clientID)
	defer m.Unsubscribe(clientID)

	// Send heartbeat periodically
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	// Client disconnect detection
	clientGone := r.Context().Done()

	for {
		select {
		case update := <-updates:
			data, err := json.Marshal(update)
			if err != nil {
				log.Printf("Error marshaling update: %v", err)
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

		case <-heartbeat.C:
			fmt.Fprintf(w, "data: {\"type\":\"heartbeat\"}\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

		case <-clientGone:
			return
		}
	}
}

// handleDashboard serves the dashboard HTML
func (m *Monitor) handleDashboard(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Claude Agent Monitoring Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metrics-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .metric-card { border: 1px solid #ddd; padding: 20px; border-radius: 8px; }
        .metric-value { font-size: 2em; font-weight: bold; color: #333; }
        .metric-label { font-size: 0.9em; color: #666; margin-top: 5px; }
        .success { color: #4CAF50; }
        .warning { color: #FF9800; }
        .error { color: #F44336; }
        .chart-container { height: 300px; margin-top: 20px; }
        #sessionChart { width: 100%; height: 100%; }
    </style>
</head>
<body>
    <h1>Claude Agent Monitoring Dashboard</h1>
    
    <div class="metrics-grid">
        <div class="metric-card">
            <div class="metric-value" id="totalSessions">0</div>
            <div class="metric-label">Total Sessions</div>
        </div>
        <div class="metric-card">
            <div class="metric-value success" id="successfulSessions">0</div>
            <div class="metric-label">Successful Sessions</div>
        </div>
        <div class="metric-card">
            <div class="metric-value error" id="failedSessions">0</div>
            <div class="metric-label">Failed Sessions</div>
        </div>
        <div class="metric-card">
            <div class="metric-value" id="activeSessions">0</div>
            <div class="metric-label">Active Sessions</div>
        </div>
        <div class="metric-card">
            <div class="metric-value" id="completedFeatures">0</div>
            <div class="metric-label">Completed Features</div>
        </div>
        <div class="metric-card">
            <div class="metric-value warning" id="pendingFeatures">0</div>
            <div class="metric-label">Pending Features</div>
        </div>
    </div>

    <div id="sessionHistory">
        <h2>Recent Sessions</h2>
        <table id="sessionsTable">
            <thead>
                <tr>
                    <th>Session ID</th>
                    <th>Status</th>
                    <th>Duration</th>
                    <th>Features</th>
                    <th>Tokens</th>
                    <th>Time</th>
                </tr>
            </thead>
            <tbody></tbody>
        </table>
    </div>

    <script>
        // Connect to server-sent events
        const eventSource = new EventSource('/api/stream');
        
        eventSource.onmessage = function(event) {
            const data = JSON.parse(event.data);
            if (data.type === 'heartbeat') return;
            
            updateMetrics(data.metrics);
            updateSessionHistory(data.metrics.session_history);
        };
        
        function updateMetrics(metrics) {
            document.getElementById('totalSessions').textContent = metrics.total_sessions;
            document.getElementById('successfulSessions').textContent = metrics.successful_sessions;
            document.getElementById('failedSessions').textContent = metrics.failed_sessions;
            document.getElementById('activeSessions').textContent = metrics.active_sessions;
            document.getElementById('completedFeatures').textContent = metrics.completed_features;
            document.getElementById('pendingFeatures').textContent = metrics.pending_features;
        }
        
        function updateSessionHistory(sessions) {
            const tbody = document.querySelector('#sessionsTable tbody');
            tbody.innerHTML = '';
            
            sessions.slice(-10).reverse().forEach(session => {
                const row = tbody.insertRow();
                row.innerHTML = '<td>' + session.session_id + '</td>' +
                               '<td class="' + (session.success ? 'success' : 'error') + '">' + 
                               (session.success ? 'Success' : 'Failed') + '</td>' +
                               '<td>' + Math.round(session.duration / 1000000) + 'ms</td>' +
                               '<td>' + session.features_completed.length + '</td>' +
                               '<td>' + session.tokens_used + '</td>' +
                               '<td>' + new Date(session.end_time).toLocaleTimeString() + '</td>';
            });
        }
        
        // Initial load
        fetch('/api/metrics')
            .then(response => response.json())
            .then(updateMetrics);
            
        fetch('/api/sessions')
            .then(response => response.json())
            .then(updateSessionHistory);
    </script>
</body>
</html>
`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}
