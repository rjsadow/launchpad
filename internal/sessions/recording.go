package sessions

import "time"

// SessionEventType represents the type of session lifecycle event.
type SessionEventType string

const (
	// EventSessionCreated is emitted when a new session is created.
	EventSessionCreated SessionEventType = "session.created"

	// EventSessionReady is emitted when a session's pod becomes ready.
	EventSessionReady SessionEventType = "session.ready"

	// EventSessionFailed is emitted when a session fails to start.
	EventSessionFailed SessionEventType = "session.failed"

	// EventSessionStopped is emitted when a session is stopped by the user.
	EventSessionStopped SessionEventType = "session.stopped"

	// EventSessionRestarted is emitted when a stopped session is restarted.
	EventSessionRestarted SessionEventType = "session.restarted"

	// EventSessionExpired is emitted when a session expires due to timeout.
	EventSessionExpired SessionEventType = "session.expired"

	// EventSessionTerminated is emitted when a session is terminated.
	EventSessionTerminated SessionEventType = "session.terminated"
)

// SessionEvent represents a single session lifecycle event.
type SessionEvent struct {
	Type      SessionEventType `json:"type"`
	SessionID string           `json:"session_id"`
	UserID    string           `json:"user_id,omitempty"`
	AppID     string           `json:"app_id,omitempty"`
	Reason    string           `json:"reason,omitempty"`
	Timestamp time.Time        `json:"timestamp"`
}

// SessionRecorder is the interface for recording session lifecycle events.
// Implementations can store events in files, send to webhooks, write to S3, etc.
type SessionRecorder interface {
	// RecordEvent records a session lifecycle event.
	RecordEvent(event SessionEvent) error

	// Close performs any cleanup needed by the recorder.
	Close() error
}

// RecordingConfig holds configuration for session recording.
type RecordingConfig struct {
	Enabled bool   // Whether recording is enabled
	Driver  string // Recording driver name (for future use: "file", "webhook", "s3", etc.)
}
