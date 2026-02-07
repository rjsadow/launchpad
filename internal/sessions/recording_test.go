package sessions

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// mockRecorder captures events for testing.
type mockRecorder struct {
	mu     sync.Mutex
	events []SessionEvent
	err    error // if set, RecordEvent returns this error
	closed bool
}

func (m *mockRecorder) RecordEvent(event SessionEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, event)
	return nil
}

func (m *mockRecorder) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockRecorder) getEvents() []SessionEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]SessionEvent, len(m.events))
	copy(cp, m.events)
	return cp
}

func TestSessionEventTypes(t *testing.T) {
	expected := []SessionEventType{
		EventSessionCreated,
		EventSessionReady,
		EventSessionFailed,
		EventSessionStopped,
		EventSessionRestarted,
		EventSessionExpired,
		EventSessionTerminated,
	}
	for _, et := range expected {
		if et == "" {
			t.Error("event type should not be empty")
		}
	}
	if len(expected) != 7 {
		t.Errorf("expected 7 event types, got %d", len(expected))
	}
}

func TestNoopRecorder(t *testing.T) {
	rec := &NoopRecorder{}

	event := SessionEvent{
		Type:      EventSessionCreated,
		SessionID: "test-session",
		UserID:    "user1",
		AppID:     "app1",
		Timestamp: time.Now(),
	}

	if err := rec.RecordEvent(event); err != nil {
		t.Errorf("NoopRecorder.RecordEvent() returned error: %v", err)
	}

	if err := rec.Close(); err != nil {
		t.Errorf("NoopRecorder.Close() returned error: %v", err)
	}
}

func TestNoopRecorderImplementsInterface(t *testing.T) {
	var _ SessionRecorder = &NoopRecorder{}
}

func TestMockRecorderCapturesEvents(t *testing.T) {
	rec := &mockRecorder{}

	events := []SessionEvent{
		{Type: EventSessionCreated, SessionID: "s1", UserID: "u1", AppID: "a1", Timestamp: time.Now()},
		{Type: EventSessionReady, SessionID: "s1", Timestamp: time.Now()},
		{Type: EventSessionStopped, SessionID: "s1", UserID: "u1", Reason: "user stopped", Timestamp: time.Now()},
	}

	for _, e := range events {
		if err := rec.RecordEvent(e); err != nil {
			t.Errorf("RecordEvent() returned error: %v", err)
		}
	}

	captured := rec.getEvents()
	if len(captured) != 3 {
		t.Fatalf("expected 3 events, got %d", len(captured))
	}
	if captured[0].Type != EventSessionCreated {
		t.Errorf("expected first event type %q, got %q", EventSessionCreated, captured[0].Type)
	}
	if captured[1].Type != EventSessionReady {
		t.Errorf("expected second event type %q, got %q", EventSessionReady, captured[1].Type)
	}
	if captured[2].Type != EventSessionStopped {
		t.Errorf("expected third event type %q, got %q", EventSessionStopped, captured[2].Type)
	}
}

func TestRecorderErrorHandling(t *testing.T) {
	rec := &mockRecorder{err: errors.New("write failed")}

	event := SessionEvent{
		Type:      EventSessionCreated,
		SessionID: "s1",
		Timestamp: time.Now(),
	}

	err := rec.RecordEvent(event)
	if err == nil {
		t.Error("expected error from RecordEvent, got nil")
	}
	if err.Error() != "write failed" {
		t.Errorf("expected 'write failed', got %q", err.Error())
	}
}

func TestEmitEventDisabled(t *testing.T) {
	rec := &mockRecorder{}
	mgr := &Manager{
		recorder:         rec,
		recordingEnabled: false,
		stopCh:           make(chan struct{}),
	}

	mgr.emitEvent(EventSessionCreated, "s1", "u1", "a1", "test")

	if len(rec.getEvents()) != 0 {
		t.Error("expected no events when recording is disabled")
	}
}

func TestEmitEventEnabled(t *testing.T) {
	rec := &mockRecorder{}
	mgr := &Manager{
		recorder:         rec,
		recordingEnabled: true,
		stopCh:           make(chan struct{}),
	}

	mgr.emitEvent(EventSessionCreated, "s1", "u1", "a1", "test")

	events := rec.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != EventSessionCreated {
		t.Errorf("expected event type %q, got %q", EventSessionCreated, events[0].Type)
	}
	if events[0].SessionID != "s1" {
		t.Errorf("expected session ID 's1', got %q", events[0].SessionID)
	}
	if events[0].UserID != "u1" {
		t.Errorf("expected user ID 'u1', got %q", events[0].UserID)
	}
	if events[0].AppID != "a1" {
		t.Errorf("expected app ID 'a1', got %q", events[0].AppID)
	}
	if events[0].Reason != "test" {
		t.Errorf("expected reason 'test', got %q", events[0].Reason)
	}
	if events[0].Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestEmitEventErrorDoesNotPanic(t *testing.T) {
	rec := &mockRecorder{err: errors.New("record failed")}
	mgr := &Manager{
		recorder:         rec,
		recordingEnabled: true,
		stopCh:           make(chan struct{}),
	}

	// Should not panic even when recorder returns an error
	mgr.emitEvent(EventSessionFailed, "s1", "u1", "a1", "test failure")
}

func TestRecordingConfig(t *testing.T) {
	cfg := RecordingConfig{
		Enabled: true,
		Driver:  "file",
	}
	if !cfg.Enabled {
		t.Error("expected Enabled to be true")
	}
	if cfg.Driver != "file" {
		t.Errorf("expected Driver 'file', got %q", cfg.Driver)
	}

	cfgDefault := RecordingConfig{}
	if cfgDefault.Enabled {
		t.Error("expected Enabled to default to false")
	}
	if cfgDefault.Driver != "" {
		t.Errorf("expected Driver to default to empty, got %q", cfgDefault.Driver)
	}
}
