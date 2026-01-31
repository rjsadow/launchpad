// Package runner provides a pluggable interface for workload orchestration.
// It abstracts the underlying container orchestration platform (Kubernetes, Docker, Nomad, etc.)
// allowing Launchpad to run workloads on different backends.
package runner

import (
	"context"
	"errors"
	"time"
)

// Common errors returned by Runner implementations.
var (
	ErrWorkloadNotFound = errors.New("workload not found")
	ErrWorkloadFailed   = errors.New("workload failed")
	ErrTimeout          = errors.New("operation timed out")
)

// Runner defines the interface for workload orchestration backends.
// Implementations must be safe for concurrent use.
type Runner interface {
	// CreateWorkload creates a new workload (container/pod/job).
	// Returns the created Workload with its ID and initial status.
	CreateWorkload(ctx context.Context, config *WorkloadConfig) (*Workload, error)

	// DeleteWorkload deletes a workload by ID.
	// Returns nil if the workload is successfully deleted or doesn't exist.
	DeleteWorkload(ctx context.Context, workloadID string) error

	// GetWorkload returns the current state of a workload.
	// Returns ErrWorkloadNotFound if the workload doesn't exist.
	GetWorkload(ctx context.Context, workloadID string) (*Workload, error)

	// WaitForReady blocks until the workload is ready or times out.
	// Returns ErrWorkloadFailed if the workload enters a terminal failed state.
	// Returns ErrTimeout if the timeout is exceeded.
	WaitForReady(ctx context.Context, workloadID string, timeout time.Duration) error

	// ListWorkloads returns all workloads managed by this runner.
	// Can optionally filter by labels.
	ListWorkloads(ctx context.Context, labels map[string]string) ([]*Workload, error)

	// Type returns the runner type identifier (e.g., "kubernetes", "docker", "nomad").
	Type() string

	// Close releases any resources held by the runner.
	Close() error
}

// Factory creates a Runner instance from configuration.
type Factory func(cfg *Config) (Runner, error)

// registry holds registered runner factories.
var registry = make(map[string]Factory)

// Register adds a runner factory to the registry.
// Call this from runner implementation init() functions.
func Register(runnerType string, factory Factory) {
	registry[runnerType] = factory
}

// New creates a Runner of the specified type using the provided configuration.
func New(runnerType string, cfg *Config) (Runner, error) {
	factory, ok := registry[runnerType]
	if !ok {
		return nil, errors.New("unknown runner type: " + runnerType)
	}
	return factory(cfg)
}

// Available returns the list of registered runner types.
func Available() []string {
	types := make([]string, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	return types
}
