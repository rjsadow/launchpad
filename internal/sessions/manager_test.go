package sessions

import (
	"testing"
	"time"
)

func TestManagerConfig_Defaults(t *testing.T) {
	cfg := ManagerConfig{}

	// Zero values should trigger default application in NewManagerWithConfig
	if cfg.SessionTimeout != 0 {
		t.Errorf("expected SessionTimeout to be zero, got %v", cfg.SessionTimeout)
	}
	if cfg.CleanupInterval != 0 {
		t.Errorf("expected CleanupInterval to be zero, got %v", cfg.CleanupInterval)
	}
	if cfg.PodReadyTimeout != 0 {
		t.Errorf("expected PodReadyTimeout to be zero, got %v", cfg.PodReadyTimeout)
	}
}

func TestDefaultConstants(t *testing.T) {
	if DefaultSessionTimeout != 2*time.Hour {
		t.Errorf("expected DefaultSessionTimeout to be 2h, got %v", DefaultSessionTimeout)
	}
	if DefaultCleanupInterval != 5*time.Minute {
		t.Errorf("expected DefaultCleanupInterval to be 5m, got %v", DefaultCleanupInterval)
	}
	if DefaultPodReadyTimeout != 2*time.Minute {
		t.Errorf("expected DefaultPodReadyTimeout to be 2m, got %v", DefaultPodReadyTimeout)
	}
}

func TestCreateSessionRequest_Fields(t *testing.T) {
	req := CreateSessionRequest{
		AppID:       "app-1",
		UserID:      "user-1",
		TTL:         3600,
		IdleTimeout: 1800,
	}

	if req.AppID != "app-1" {
		t.Errorf("expected AppID to be app-1, got %s", req.AppID)
	}
	if req.UserID != "user-1" {
		t.Errorf("expected UserID to be user-1, got %s", req.UserID)
	}
	if req.TTL != 3600 {
		t.Errorf("expected TTL to be 3600, got %d", req.TTL)
	}
	if req.IdleTimeout != 1800 {
		t.Errorf("expected IdleTimeout to be 1800, got %d", req.IdleTimeout)
	}
}
