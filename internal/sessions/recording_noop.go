package sessions

// NoopRecorder is the default SessionRecorder that discards all events.
// Used when recording is disabled (the default).
type NoopRecorder struct{}

// RecordEvent is a no-op; it always returns nil.
func (n *NoopRecorder) RecordEvent(_ SessionEvent) error {
	return nil
}

// Close is a no-op; it always returns nil.
func (n *NoopRecorder) Close() error {
	return nil
}
