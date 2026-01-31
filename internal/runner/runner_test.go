package runner

import (
	"context"
	"testing"
	"time"
)

// MockRunner is a test implementation of the Runner interface.
type MockRunner struct {
	workloads map[string]*Workload
	createErr error
	deleteErr error
	getErr    error
	waitErr   error
	listErr   error
}

func NewMockRunner() *MockRunner {
	return &MockRunner{
		workloads: make(map[string]*Workload),
	}
}

func (m *MockRunner) CreateWorkload(ctx context.Context, cfg *WorkloadConfig) (*Workload, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	w := &Workload{
		ID:      cfg.ID,
		Name:    "mock-" + cfg.ID,
		Status:  StatusPending,
		VNCPort: 6080,
		Labels:  cfg.Labels,
	}
	m.workloads[cfg.ID] = w
	return w, nil
}

func (m *MockRunner) DeleteWorkload(ctx context.Context, workloadID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.workloads, workloadID)
	return nil
}

func (m *MockRunner) GetWorkload(ctx context.Context, workloadID string) (*Workload, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	w, ok := m.workloads[workloadID]
	if !ok {
		return nil, ErrWorkloadNotFound
	}
	return w, nil
}

func (m *MockRunner) WaitForReady(ctx context.Context, workloadID string, timeout time.Duration) error {
	if m.waitErr != nil {
		return m.waitErr
	}
	w, ok := m.workloads[workloadID]
	if !ok {
		return ErrWorkloadNotFound
	}
	w.Status = StatusRunning
	w.IP = "10.0.0.1"
	return nil
}

func (m *MockRunner) ListWorkloads(ctx context.Context, labels map[string]string) ([]*Workload, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*Workload
	for _, w := range m.workloads {
		result = append(result, w)
	}
	return result, nil
}

func (m *MockRunner) Type() string {
	return "mock"
}

func (m *MockRunner) Close() error {
	return nil
}

// SetWorkloadReady simulates a workload becoming ready
func (m *MockRunner) SetWorkloadReady(workloadID, ip string) {
	if w, ok := m.workloads[workloadID]; ok {
		w.Status = StatusRunning
		w.IP = ip
	}
}

func TestWorkloadStatus(t *testing.T) {
	tests := []struct {
		status     WorkloadStatus
		isTerminal bool
		isReady    bool
	}{
		{StatusPending, false, false},
		{StatusRunning, false, true},
		{StatusFailed, true, false},
		{StatusTerminated, true, false},
		{StatusUnknown, false, false},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			if got := tc.status.IsTerminal(); got != tc.isTerminal {
				t.Errorf("IsTerminal() = %v, want %v", got, tc.isTerminal)
			}
			if got := tc.status.IsReady(); got != tc.isReady {
				t.Errorf("IsReady() = %v, want %v", got, tc.isReady)
			}
		})
	}
}

func TestDefaultWorkloadConfig(t *testing.T) {
	cfg := DefaultWorkloadConfig("session-123", "app-456", "Test App", "nginx:latest")

	if cfg.ID != "session-123" {
		t.Errorf("ID = %q, want %q", cfg.ID, "session-123")
	}
	if cfg.AppID != "app-456" {
		t.Errorf("AppID = %q, want %q", cfg.AppID, "app-456")
	}
	if cfg.AppName != "Test App" {
		t.Errorf("AppName = %q, want %q", cfg.AppName, "Test App")
	}
	if cfg.ContainerImage != "nginx:latest" {
		t.Errorf("ContainerImage = %q, want %q", cfg.ContainerImage, "nginx:latest")
	}
	if cfg.CPULimit != "2" {
		t.Errorf("CPULimit = %q, want %q", cfg.CPULimit, "2")
	}
	if cfg.MemoryLimit != "2Gi" {
		t.Errorf("MemoryLimit = %q, want %q", cfg.MemoryLimit, "2Gi")
	}
	if !cfg.EnableVNC {
		t.Error("EnableVNC = false, want true")
	}
}

func TestMockRunner(t *testing.T) {
	ctx := context.Background()
	r := NewMockRunner()

	// Test CreateWorkload
	cfg := DefaultWorkloadConfig("test-1", "app-1", "Test", "nginx")
	w, err := r.CreateWorkload(ctx, cfg)
	if err != nil {
		t.Fatalf("CreateWorkload() error = %v", err)
	}
	if w.ID != "test-1" {
		t.Errorf("Workload.ID = %q, want %q", w.ID, "test-1")
	}
	if w.Status != StatusPending {
		t.Errorf("Workload.Status = %v, want %v", w.Status, StatusPending)
	}

	// Test GetWorkload
	w2, err := r.GetWorkload(ctx, "test-1")
	if err != nil {
		t.Fatalf("GetWorkload() error = %v", err)
	}
	if w2.ID != w.ID {
		t.Errorf("GetWorkload() returned different workload")
	}

	// Test GetWorkload not found
	_, err = r.GetWorkload(ctx, "nonexistent")
	if err != ErrWorkloadNotFound {
		t.Errorf("GetWorkload() error = %v, want %v", err, ErrWorkloadNotFound)
	}

	// Test WaitForReady
	err = r.WaitForReady(ctx, "test-1", time.Minute)
	if err != nil {
		t.Fatalf("WaitForReady() error = %v", err)
	}
	w3, _ := r.GetWorkload(ctx, "test-1")
	if w3.Status != StatusRunning {
		t.Errorf("Workload.Status = %v, want %v", w3.Status, StatusRunning)
	}

	// Test ListWorkloads
	workloads, err := r.ListWorkloads(ctx, nil)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}
	if len(workloads) != 1 {
		t.Errorf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	// Test DeleteWorkload
	err = r.DeleteWorkload(ctx, "test-1")
	if err != nil {
		t.Fatalf("DeleteWorkload() error = %v", err)
	}
	_, err = r.GetWorkload(ctx, "test-1")
	if err != ErrWorkloadNotFound {
		t.Errorf("Workload should be deleted, error = %v", err)
	}

	// Test Type
	if r.Type() != "mock" {
		t.Errorf("Type() = %q, want %q", r.Type(), "mock")
	}
}

func TestRegisterAndNew(t *testing.T) {
	// Register a test factory
	Register("test", func(cfg *Config) (Runner, error) {
		return NewMockRunner(), nil
	})

	// Check Available()
	types := Available()
	found := false
	for _, rt := range types {
		if rt == "test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Available() does not include 'test' runner")
	}

	// Create runner
	r, err := New("test", &Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if r == nil {
		t.Fatal("New() returned nil runner")
	}

	// Test unknown runner
	_, err = New("unknown", &Config{})
	if err == nil {
		t.Error("New('unknown') should return error")
	}
}
