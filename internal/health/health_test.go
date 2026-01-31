package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHandler(t *testing.T) {
	h := NewHandler("1.0.0")
	if h == nil {
		t.Fatal("NewHandler returned nil")
	}
	if h.version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", h.version)
	}
}

func TestHandleLive(t *testing.T) {
	h := NewHandler("1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/api/health/live", nil)
	w := httptest.NewRecorder()

	h.HandleLive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != StatusHealthy {
		t.Errorf("expected status healthy, got %s", resp.Status)
	}
	if resp.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", resp.Version)
	}
}

func TestHandleReady(t *testing.T) {
	h := NewHandler("1.0.0")

	// Register a healthy checker
	h.RegisterChecker("test", func(ctx context.Context) ComponentHealth {
		return ComponentHealth{Status: StatusHealthy, Message: "OK"}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health/ready", nil)
	w := httptest.NewRecorder()

	h.HandleReady(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != StatusHealthy {
		t.Errorf("expected status healthy, got %s", resp.Status)
	}
}

func TestHandleReadyUnhealthy(t *testing.T) {
	h := NewHandler("1.0.0")

	// Register an unhealthy checker
	h.RegisterChecker("test", func(ctx context.Context) ComponentHealth {
		return ComponentHealth{Status: StatusUnhealthy, Message: "DB down"}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health/ready", nil)
	w := httptest.NewRecorder()

	h.HandleReady(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != StatusUnhealthy {
		t.Errorf("expected status unhealthy, got %s", resp.Status)
	}
}

func TestHandleHealth(t *testing.T) {
	h := NewHandler("1.0.0")

	// Register multiple checkers
	h.RegisterChecker("db", func(ctx context.Context) ComponentHealth {
		return ComponentHealth{Status: StatusHealthy, Message: "DB OK", Latency: "1ms"}
	})
	h.RegisterChecker("cache", func(ctx context.Context) ComponentHealth {
		return ComponentHealth{Status: StatusHealthy, Message: "Cache OK"}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	h.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Components) != 2 {
		t.Errorf("expected 2 components, got %d", len(resp.Components))
	}
}

func TestMethodNotAllowed(t *testing.T) {
	h := NewHandler("1.0.0")

	tests := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		path    string
	}{
		{"live", h.HandleLive, "/api/health/live"},
		{"ready", h.HandleReady, "/api/health/ready"},
		{"health", h.HandleHealth, "/api/health"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			w := httptest.NewRecorder()
			tt.handler(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status 405, got %d", w.Code)
			}
		})
	}
}

func TestRuntimeChecker(t *testing.T) {
	checker := RuntimeChecker()
	health := checker(context.Background())

	if health.Status != StatusHealthy && health.Status != StatusDegraded {
		t.Errorf("unexpected status: %s", health.Status)
	}
}

func TestCheckerTimeout(t *testing.T) {
	h := NewHandler("1.0.0")

	// Register a slow checker
	h.RegisterChecker("slow", func(ctx context.Context) ComponentHealth {
		select {
		case <-ctx.Done():
			return ComponentHealth{Status: StatusUnhealthy, Message: "timeout"}
		case <-time.After(10 * time.Second):
			return ComponentHealth{Status: StatusHealthy}
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health/ready", nil)
	w := httptest.NewRecorder()

	// This should complete within the timeout
	h.HandleReady(w, req)

	// We expect the checker to be cancelled
	var resp HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}
