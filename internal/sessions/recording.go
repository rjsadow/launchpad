package sessions

import (
	"context"
	"time"

	"github.com/rjsadow/launchpad/internal/db"
)

// SessionEventType identifies the kind of session lifecycle event.
type SessionEventType string

const (
	// SessionEventCreated is emitted when a new session is created.
	SessionEventCreated SessionEventType = "session.created"

	// SessionEventReady is emitted when a session's pod becomes ready.
	SessionEventReady SessionEventType = "session.ready"

	// SessionEventFailed is emitted when a session fails (pod timeout, etc.).
	SessionEventFailed SessionEventType = "session.failed"

	// SessionEventStopped is emitted when a user stops a session.
	SessionEventStopped SessionEventType = "session.stopped"

	// SessionEventRestarted is emitted when a stopped session is restarted.
	SessionEventRestarted SessionEventType = "session.restarted"

	// SessionEventExpired is emitted when a session expires due to timeout.
	SessionEventExpired SessionEventType = "session.expired"

	// SessionEventTerminated is emitted when a session is terminated by the user.
	SessionEventTerminated SessionEventType = "session.terminated"
)

// SessionEvent represents a session lifecycle event passed to recorders.
type SessionEvent struct {
	// Type identifies the lifecycle event.
	Type SessionEventType `json:"type"`

	// SessionID is the unique identifier of the session.
	SessionID string `json:"session_id"`

	// UserID is the owner of the session.
	UserID string `json:"user_id"`

	// AppID is the application being launched.
	AppID string `json:"app_id"`

	// Status is the session status after the event.
	Status db.SessionStatus `json:"status"`

	// Reason provides context about why the event occurred.
	Reason string `json:"reason,omitempty"`

	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`

	// Metadata holds optional key-value pairs for recorder-specific data.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// SessionRecorder defines the interface for session recording implementations.
// Implementations receive lifecycle events and can record, forward, or process
// them as needed. The default NoopRecorder discards all events.
//
// Recording is optional and off by default. To enable recording, set
// LAUNCHPAD_SESSION_RECORDING_ENABLED=true and configure a recorder
// implementation via the RecordingDriver config field.
type SessionRecorder interface {
	// OnEvent is called for each session lifecycle event.
	// Implementations should not block; long-running operations should be
	// performed asynchronously. Errors are logged but do not affect session
	// operations.
	OnEvent(ctx context.Context, event SessionEvent) error

	// Close releases any resources held by the recorder.
	// Called when the session manager stops.
	Close() error
}

// RecordingConfig holds configuration for session recording.
type RecordingConfig struct {
	// Enabled controls whether session recording hooks fire events.
	// Default: false
	Enabled bool

	// Driver identifies which recorder implementation to use.
	// Currently only "noop" is available. Future drivers may include
	// "file", "webhook", "s3", etc.
	Driver string
}
