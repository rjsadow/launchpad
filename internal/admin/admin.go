// Package admin provides administrative API handlers for system management
// and monitoring.
package admin

import (
	"encoding/json"
	"net/http"
	"time"
)

// SystemMetrics contains system-level metrics for the admin dashboard.
type SystemMetrics struct {
	Timestamp       time.Time       `json:"timestamp"`
	Uptime          string          `json:"uptime"`
	ActiveSessions  int             `json:"active_sessions"`
	TotalApps       int             `json:"total_apps"`
	TotalLaunches   int             `json:"total_launches"`
	RecentActivity  []ActivityEntry `json:"recent_activity"`
	SessionsByState map[string]int  `json:"sessions_by_state"`
}

// ActivityEntry represents a recent activity item.
type ActivityEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	User      string    `json:"user"`
	Details   string    `json:"details"`
}

// SessionInfo contains session information for admin view.
type SessionInfo struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	AppID     string    `json:"app_id"`
	AppName   string    `json:"app_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Duration  string    `json:"duration"`
}

// MetricsProvider interface for collecting metrics from various sources.
type MetricsProvider interface {
	GetAppCount() (int, error)
	GetActiveSessionCount() (int, error)
	GetTotalLaunches() (int, error)
	GetRecentAuditLogs(limit int) ([]ActivityEntry, error)
	GetSessionsByState() (map[string]int, error)
	GetAllSessions() ([]SessionInfo, error)
}

// Handler provides HTTP handlers for admin endpoints.
type Handler struct {
	provider  MetricsProvider
	startTime time.Time
}

// NewHandler creates a new admin handler.
func NewHandler(provider MetricsProvider) *Handler {
	return &Handler{
		provider:  provider,
		startTime: time.Now(),
	}
}

// HandleMetrics handles the /api/admin/metrics endpoint.
func (h *Handler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := SystemMetrics{
		Timestamp:       time.Now().UTC(),
		Uptime:          time.Since(h.startTime).Round(time.Second).String(),
		SessionsByState: make(map[string]int),
	}

	// Collect metrics from provider
	if h.provider != nil {
		if count, err := h.provider.GetAppCount(); err == nil {
			metrics.TotalApps = count
		}
		if count, err := h.provider.GetActiveSessionCount(); err == nil {
			metrics.ActiveSessions = count
		}
		if count, err := h.provider.GetTotalLaunches(); err == nil {
			metrics.TotalLaunches = count
		}
		if logs, err := h.provider.GetRecentAuditLogs(10); err == nil {
			metrics.RecentActivity = logs
		}
		if states, err := h.provider.GetSessionsByState(); err == nil {
			metrics.SessionsByState = states
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// HandleSessions handles the /api/admin/sessions endpoint.
func (h *Handler) HandleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var sessions []SessionInfo
	if h.provider != nil {
		var err error
		sessions, err = h.provider.GetAllSessions()
		if err != nil {
			http.Error(w, "Failed to get sessions", http.StatusInternalServerError)
			return
		}
	}

	if sessions == nil {
		sessions = []SessionInfo{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}
