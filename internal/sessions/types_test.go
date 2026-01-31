package sessions

import (
	"testing"
	"time"

	"github.com/rjsadow/launchpad/internal/db"
)

func TestSessionFromDB(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(time.Hour)

	session := &db.Session{
		ID:             "session-123",
		UserID:         "user-456",
		AppID:          "app-789",
		PodName:        "launchpad-session-123",
		PodIP:          "10.0.0.1",
		Status:         db.SessionStatusRunning,
		TTL:            3600,
		IdleTimeout:    1800,
		ExpiresAt:      &expiresAt,
		LastActivityAt: now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	resp := SessionFromDB(session, "My App", "/ws/sessions/session-123")

	if resp.ID != session.ID {
		t.Errorf("expected ID %s, got %s", session.ID, resp.ID)
	}
	if resp.UserID != session.UserID {
		t.Errorf("expected UserID %s, got %s", session.UserID, resp.UserID)
	}
	if resp.AppID != session.AppID {
		t.Errorf("expected AppID %s, got %s", session.AppID, resp.AppID)
	}
	if resp.AppName != "My App" {
		t.Errorf("expected AppName 'My App', got %s", resp.AppName)
	}
	if resp.PodName != session.PodName {
		t.Errorf("expected PodName %s, got %s", session.PodName, resp.PodName)
	}
	if resp.Status != session.Status {
		t.Errorf("expected Status %s, got %s", session.Status, resp.Status)
	}
	if resp.WebSocketURL != "/ws/sessions/session-123" {
		t.Errorf("expected WebSocketURL '/ws/sessions/session-123', got %s", resp.WebSocketURL)
	}
	if resp.TTL != session.TTL {
		t.Errorf("expected TTL %d, got %d", session.TTL, resp.TTL)
	}
	if resp.IdleTimeout != session.IdleTimeout {
		t.Errorf("expected IdleTimeout %d, got %d", session.IdleTimeout, resp.IdleTimeout)
	}
	if resp.ExpiresAt == nil || !resp.ExpiresAt.Equal(expiresAt) {
		t.Errorf("expected ExpiresAt %v, got %v", expiresAt, resp.ExpiresAt)
	}
	if !resp.LastActivityAt.Equal(now) {
		t.Errorf("expected LastActivityAt %v, got %v", now, resp.LastActivityAt)
	}
}

func TestSessionFromDB_NilExpiresAt(t *testing.T) {
	now := time.Now()

	session := &db.Session{
		ID:             "session-123",
		UserID:         "user-456",
		AppID:          "app-789",
		PodName:        "launchpad-session-123",
		Status:         db.SessionStatusRunning,
		TTL:            0, // No TTL set
		IdleTimeout:    0,
		ExpiresAt:      nil, // No expiration
		LastActivityAt: now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	resp := SessionFromDB(session, "My App", "")

	if resp.ExpiresAt != nil {
		t.Errorf("expected ExpiresAt to be nil, got %v", resp.ExpiresAt)
	}
	if resp.TTL != 0 {
		t.Errorf("expected TTL to be 0, got %d", resp.TTL)
	}
}

func TestSessionStatus_Stopped(t *testing.T) {
	// Verify the new stopped status is available
	session := &db.Session{
		Status: db.SessionStatusStopped,
	}

	if session.Status != db.SessionStatusStopped {
		t.Errorf("expected status to be 'stopped', got %s", session.Status)
	}
}
