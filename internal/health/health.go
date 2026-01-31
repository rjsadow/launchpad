// Package health provides health check endpoints for Kubernetes probes
// and operational monitoring.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// Status represents the health status of a component.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// ComponentHealth represents the health of a single component.
type ComponentHealth struct {
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// HealthResponse represents the overall health check response.
type HealthResponse struct {
	Status     Status                     `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Version    string                     `json:"version,omitempty"`
	Components map[string]ComponentHealth `json:"components,omitempty"`
}

// Checker is a function that checks the health of a component.
type Checker func(ctx context.Context) ComponentHealth

// Handler provides HTTP handlers for health checks.
type Handler struct {
	mu       sync.RWMutex
	checkers map[string]Checker
	version  string
}

// NewHandler creates a new health check handler.
func NewHandler(version string) *Handler {
	return &Handler{
		checkers: make(map[string]Checker),
		version:  version,
	}
}

// RegisterChecker registers a health checker for a component.
func (h *Handler) RegisterChecker(name string, checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers[name] = checker
}

// HandleHealth handles the /api/health endpoint (full health check).
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response := h.runChecks(ctx)

	w.Header().Set("Content-Type", "application/json")
	if response.Status == StatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else if response.Status == StatusDegraded {
		w.WriteHeader(http.StatusOK) // Still OK for K8s, but indicates degradation
	}
	json.NewEncoder(w).Encode(response)
}

// HandleLive handles the /api/health/live endpoint (liveness probe).
// Returns 200 if the service is running, regardless of dependency health.
func (h *Handler) HandleLive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:    StatusHealthy,
		Timestamp: time.Now().UTC(),
		Version:   h.version,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleReady handles the /api/health/ready endpoint (readiness probe).
// Returns 200 only if all critical dependencies are healthy.
func (h *Handler) HandleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	response := h.runChecks(ctx)

	w.Header().Set("Content-Type", "application/json")
	if response.Status == StatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(response)
}

// runChecks executes all registered health checkers.
func (h *Handler) runChecks(ctx context.Context) HealthResponse {
	h.mu.RLock()
	defer h.mu.RUnlock()

	response := HealthResponse{
		Status:     StatusHealthy,
		Timestamp:  time.Now().UTC(),
		Version:    h.version,
		Components: make(map[string]ComponentHealth),
	}

	// Run all checkers concurrently
	results := make(chan struct {
		name   string
		health ComponentHealth
	}, len(h.checkers))

	var wg sync.WaitGroup
	for name, checker := range h.checkers {
		wg.Add(1)
		go func(name string, checker Checker) {
			defer wg.Done()
			results <- struct {
				name   string
				health ComponentHealth
			}{name, checker(ctx)}
		}(name, checker)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for result := range results {
		response.Components[result.name] = result.health

		// Update overall status based on component health
		switch result.health.Status {
		case StatusUnhealthy:
			response.Status = StatusUnhealthy
		case StatusDegraded:
			if response.Status != StatusUnhealthy {
				response.Status = StatusDegraded
			}
		}
	}

	return response
}

// RuntimeChecker returns a checker for Go runtime metrics.
func RuntimeChecker() Checker {
	return func(ctx context.Context) ComponentHealth {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		// Consider unhealthy if using more than 90% of available memory
		// This is a simple heuristic; adjust based on your needs
		if m.Sys > 0 && float64(m.Alloc)/float64(m.Sys) > 0.9 {
			return ComponentHealth{
				Status:  StatusDegraded,
				Message: "High memory usage",
			}
		}

		return ComponentHealth{
			Status:  StatusHealthy,
			Message: "Runtime OK",
		}
	}
}
