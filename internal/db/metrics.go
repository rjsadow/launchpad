package db

import (
	"time"
)

// MetricsActivityEntry represents a recent activity item for metrics.
type MetricsActivityEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	User      string    `json:"user"`
	Details   string    `json:"details"`
}

// MetricsSessionInfo contains session information for admin view.
type MetricsSessionInfo struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	AppID     string    `json:"app_id"`
	AppName   string    `json:"app_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Duration  string    `json:"duration"`
}

// GetAppCount returns the total number of applications.
func (db *DB) GetAppCount() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM applications").Scan(&count)
	return count, err
}

// GetActiveSessionCount returns the number of active sessions.
func (db *DB) GetActiveSessionCount() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM sessions WHERE status IN ('creating', 'running')").Scan(&count)
	return count, err
}

// GetTotalLaunches returns the total number of app launches.
func (db *DB) GetTotalLaunches() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM analytics").Scan(&count)
	return count, err
}

// GetRecentAuditLogs returns recent audit log entries for metrics.
func (db *DB) GetRecentAuditLogs(limit int) ([]MetricsActivityEntry, error) {
	rows, err := db.conn.Query(
		"SELECT timestamp, action, user, details FROM audit_log ORDER BY timestamp DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []MetricsActivityEntry
	for rows.Next() {
		var entry MetricsActivityEntry
		if err := rows.Scan(&entry.Timestamp, &entry.Action, &entry.User, &entry.Details); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// GetSessionsByState returns a count of sessions grouped by status.
func (db *DB) GetSessionsByState() (map[string]int, error) {
	rows, err := db.conn.Query("SELECT status, COUNT(*) FROM sessions GROUP BY status")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	states := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		states[status] = count
	}

	return states, rows.Err()
}

// GetAllSessions returns all sessions with app names for admin view.
func (db *DB) GetAllSessions() ([]MetricsSessionInfo, error) {
	rows, err := db.conn.Query(`
		SELECT s.id, s.user_id, s.app_id, COALESCE(a.name, s.app_id) as app_name,
		       s.status, s.created_at, s.updated_at
		FROM sessions s
		LEFT JOIN applications a ON s.app_id = a.id
		ORDER BY s.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []MetricsSessionInfo
	for rows.Next() {
		var s MetricsSessionInfo
		if err := rows.Scan(&s.ID, &s.UserID, &s.AppID, &s.AppName, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		s.Duration = time.Since(s.CreatedAt).Round(time.Second).String()
		sessions = append(sessions, s)
	}

	return sessions, rows.Err()
}

// GetAuditLogCount returns the total number of audit log entries.
func (db *DB) GetAuditLogCount() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM audit_log").Scan(&count)
	return count, err
}

// GetSessionCount returns the total number of sessions.
func (db *DB) GetSessionCount() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&count)
	return count, err
}
