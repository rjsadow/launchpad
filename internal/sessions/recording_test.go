package sessions

import (
	"context"
	"sync"
	"testing"

	"github.com/rjsadow/launchpad/internal/db"
)

// mockRecorder captures events for testing.
type mockRecorder struct {
	mu     sync.Mutex
	events []SessionEvent
	closed bool
}

func (m *mockRecorder) OnEvent(_ context.Context, event SessionEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *mockRecorder) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockRecorder) Events() []SessionEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]SessionEvent, len(m.events))
	copy(cp, m.events)
	return cp
}

func TestNoopRecorder(t *testing.T) {
	r := &NoopRecorder{}

	// OnEvent should not error
	err := r.OnEvent(context.Background(), SessionEvent{
		Type:      SessionEventCreated,
		SessionID: "test-1",
	})
	if err != nil {
		t.Errorf("NoopRecorder.OnEvent() error = %v, want nil", err)
	}

	// Close should not error
	err = r.Close()
	if err != nil {
		t.Errorf("NoopRecorder.Close() error = %v, want nil", err)
	}
}

func TestSessionEventTypes(t *testing.T) {
	// Verify all event type constants are distinct
	types := []SessionEventType{
		SessionEventCreated,
		SessionEventReady,
		SessionEventFailed,
		SessionEventStopped,
		SessionEventRestarted,
		SessionEventExpired,
		SessionEventTerminated,
	}

	seen := make(map[SessionEventType]bool)
	for _, et := range types {
		if seen[et] {
			t.Errorf("Duplicate event type: %s", et)
		}
		seen[et] = true
		if et == "" {
			t.Error("Event type should not be empty")
		}
	}
}

func TestEmitEventWhenDisabled(t *testing.T) {
	rec := &mockRecorder{}

	m := &Manager{
		recording: RecordingConfig{Enabled: false},
		recorder:  rec,
		sessions:  make(map[string]*db.Session),
		stopCh:    make(chan struct{}),
	}

	session := &db.Session{
		ID:     "test-1",
		UserID: "user-1",
		AppID:  "app-1",
		Status: db.SessionStatusCreating,
	}

	m.emitEvent(context.Background(), SessionEventCreated, session, "test")

	events := rec.Events()
	if len(events) != 0 {
		t.Errorf("Expected 0 events when recording disabled, got %d", len(events))
	}
}

func TestEmitEventWhenEnabled(t *testing.T) {
	rec := &mockRecorder{}

	m := &Manager{
		recording: RecordingConfig{Enabled: true, Driver: "mock"},
		recorder:  rec,
		sessions:  make(map[string]*db.Session),
		stopCh:    make(chan struct{}),
	}

	session := &db.Session{
		ID:     "test-1",
		UserID: "user-1",
		AppID:  "app-1",
		Status: db.SessionStatusRunning,
	}

	m.emitEvent(context.Background(), SessionEventReady, session, "pod ready")

	events := rec.Events()
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	e := events[0]
	if e.Type != SessionEventReady {
		t.Errorf("Event type = %s, want %s", e.Type, SessionEventReady)
	}
	if e.SessionID != "test-1" {
		t.Errorf("SessionID = %s, want test-1", e.SessionID)
	}
	if e.UserID != "user-1" {
		t.Errorf("UserID = %s, want user-1", e.UserID)
	}
	if e.AppID != "app-1" {
		t.Errorf("AppID = %s, want app-1", e.AppID)
	}
	if e.Status != db.SessionStatusRunning {
		t.Errorf("Status = %s, want %s", e.Status, db.SessionStatusRunning)
	}
	if e.Reason != "pod ready" {
		t.Errorf("Reason = %s, want 'pod ready'", e.Reason)
	}
	if e.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestEmitMultipleEvents(t *testing.T) {
	rec := &mockRecorder{}

	m := &Manager{
		recording: RecordingConfig{Enabled: true},
		recorder:  rec,
		sessions:  make(map[string]*db.Session),
		stopCh:    make(chan struct{}),
	}

	session := &db.Session{
		ID:     "test-1",
		UserID: "user-1",
		AppID:  "app-1",
		Status: db.SessionStatusCreating,
	}

	m.emitEvent(context.Background(), SessionEventCreated, session, "created")

	session.Status = db.SessionStatusRunning
	m.emitEvent(context.Background(), SessionEventReady, session, "pod ready")

	session.Status = db.SessionStatusStopped
	m.emitEvent(context.Background(), SessionEventStopped, session, "user stopped")

	events := rec.Events()
	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	expected := []SessionEventType{SessionEventCreated, SessionEventReady, SessionEventStopped}
	for i, want := range expected {
		if events[i].Type != want {
			t.Errorf("Event[%d].Type = %s, want %s", i, events[i].Type, want)
		}
	}
}

func TestRecordingConfig(t *testing.T) {
	cfg := RecordingConfig{
		Enabled: false,
		Driver:  "noop",
	}

	if cfg.Enabled {
		t.Error("Default recording config should be disabled")
	}
	if cfg.Driver != "noop" {
		t.Errorf("Default driver = %s, want noop", cfg.Driver)
	}
}

func TestManagerDefaultsToNoopRecorder(t *testing.T) {
	m := NewManagerWithConfig(nil, ManagerConfig{})

	if m.recorder == nil {
		t.Fatal("Manager recorder should not be nil")
	}

	// Should be a NoopRecorder
	if _, ok := m.recorder.(*NoopRecorder); !ok {
		t.Errorf("Expected NoopRecorder, got %T", m.recorder)
	}
}

func TestManagerUsesProvidedRecorder(t *testing.T) {
	rec := &mockRecorder{}
	m := NewManagerWithConfig(nil, ManagerConfig{
		Recorder: rec,
	})

	if m.recorder != rec {
		t.Error("Manager should use the provided recorder")
	}
}
