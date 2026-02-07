package sessions

import "context"

// NoopRecorder is the default SessionRecorder that discards all events.
// It is used when session recording is disabled or no driver is configured.
type NoopRecorder struct{}

// OnEvent discards the event and returns nil.
func (n *NoopRecorder) OnEvent(_ context.Context, _ SessionEvent) error {
	return nil
}

// Close is a no-op.
func (n *NoopRecorder) Close() error {
	return nil
}
