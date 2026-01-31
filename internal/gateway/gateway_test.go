package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// mockSessionResolver implements SessionResolver for testing
type mockSessionResolver struct {
	sessions map[string]string // sessionID -> upstreamURL
	errors   map[string]error  // sessionID -> error
}

func newMockResolver() *mockSessionResolver {
	return &mockSessionResolver{
		sessions: make(map[string]string),
		errors:   make(map[string]error),
	}
}

func (m *mockSessionResolver) ValidateSession(ctx context.Context, sessionID string) error {
	if err, ok := m.errors[sessionID]; ok {
		return err
	}
	if _, ok := m.sessions[sessionID]; !ok {
		return ErrSessionNotFound
	}
	return nil
}

func (m *mockSessionResolver) ResolveSession(ctx context.Context, sessionID string) (string, error) {
	if err, ok := m.errors[sessionID]; ok {
		return "", err
	}
	url, ok := m.sessions[sessionID]
	if !ok {
		return "", ErrSessionNotFound
	}
	return url, nil
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxConnectionsPerUser != 5 {
		t.Errorf("expected MaxConnectionsPerUser=5, got %d", cfg.MaxConnectionsPerUser)
	}
	if cfg.MaxConnectionsTotal != 1000 {
		t.Errorf("expected MaxConnectionsTotal=1000, got %d", cfg.MaxConnectionsTotal)
	}
	if cfg.ConnectionTimeout != 2*time.Hour {
		t.Errorf("expected ConnectionTimeout=2h, got %v", cfg.ConnectionTimeout)
	}
	if cfg.IdleTimeout != 30*time.Minute {
		t.Errorf("expected IdleTimeout=30m, got %v", cfg.IdleTimeout)
	}
	if cfg.HeartbeatInterval != 30*time.Second {
		t.Errorf("expected HeartbeatInterval=30s, got %v", cfg.HeartbeatInterval)
	}
}

func TestGatewayNew(t *testing.T) {
	cfg := DefaultConfig()
	resolver := newMockResolver()
	auth := NewNoopAuthenticator()

	gw := New(cfg, auth, resolver)

	if gw == nil {
		t.Fatal("expected non-nil gateway")
	}
	if gw.ConnectionCount() != 0 {
		t.Errorf("expected 0 connections, got %d", gw.ConnectionCount())
	}
}

func TestGatewayStartStop(t *testing.T) {
	cfg := DefaultConfig()
	cfg.HeartbeatInterval = 100 * time.Millisecond
	resolver := newMockResolver()

	gw := New(cfg, nil, resolver)
	gw.Start()

	// Give goroutines time to start
	time.Sleep(50 * time.Millisecond)

	gw.Stop()

	// Verify stopped
	if !gw.stopped {
		t.Error("expected gateway to be stopped")
	}
}

func TestConnectionAddRemove(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxConnectionsPerUser = 2
	cfg.MaxConnectionsTotal = 5
	resolver := newMockResolver()

	gw := New(cfg, nil, resolver)

	// Add first connection
	conn1 := &Connection{
		ID:        "conn-1",
		SessionID: "session-1",
		UserID:    "user-1",
		State:     ConnectionStatePending,
		closeCh:   make(chan struct{}),
	}

	err := gw.addConnection(conn1)
	if err != nil {
		t.Fatalf("unexpected error adding connection: %v", err)
	}
	if gw.ConnectionCount() != 1 {
		t.Errorf("expected 1 connection, got %d", gw.ConnectionCount())
	}
	if gw.UserConnectionCount("user-1") != 1 {
		t.Errorf("expected 1 user connection, got %d", gw.UserConnectionCount("user-1"))
	}

	// Add second connection for same user
	conn2 := &Connection{
		ID:        "conn-2",
		SessionID: "session-2",
		UserID:    "user-1",
		State:     ConnectionStatePending,
		closeCh:   make(chan struct{}),
	}

	err = gw.addConnection(conn2)
	if err != nil {
		t.Fatalf("unexpected error adding connection: %v", err)
	}
	if gw.UserConnectionCount("user-1") != 2 {
		t.Errorf("expected 2 user connections, got %d", gw.UserConnectionCount("user-1"))
	}

	// Try to add third connection for same user (should fail)
	conn3 := &Connection{
		ID:        "conn-3",
		SessionID: "session-3",
		UserID:    "user-1",
		State:     ConnectionStatePending,
		closeCh:   make(chan struct{}),
	}

	err = gw.addConnection(conn3)
	if err != ErrConnectionLimitExceeded {
		t.Errorf("expected ErrConnectionLimitExceeded, got %v", err)
	}

	// Add connection for different user (should work)
	conn4 := &Connection{
		ID:        "conn-4",
		SessionID: "session-4",
		UserID:    "user-2",
		State:     ConnectionStatePending,
		closeCh:   make(chan struct{}),
	}

	err = gw.addConnection(conn4)
	if err != nil {
		t.Fatalf("unexpected error adding connection: %v", err)
	}
	if gw.ConnectionCount() != 3 {
		t.Errorf("expected 3 connections, got %d", gw.ConnectionCount())
	}

	// Remove a connection
	gw.removeConnection(conn1)
	if gw.ConnectionCount() != 2 {
		t.Errorf("expected 2 connections after removal, got %d", gw.ConnectionCount())
	}
	if gw.UserConnectionCount("user-1") != 1 {
		t.Errorf("expected 1 user-1 connection after removal, got %d", gw.UserConnectionCount("user-1"))
	}
}

func TestConnectionClose(t *testing.T) {
	conn := &Connection{
		ID:      "test-conn",
		State:   ConnectionStateConnected,
		closeCh: make(chan struct{}),
	}

	if conn.IsClosed() {
		t.Error("connection should not be closed initially")
	}

	conn.Close()

	if !conn.IsClosed() {
		t.Error("connection should be closed after Close()")
	}
	if conn.State != ConnectionStateClosed {
		t.Errorf("expected state Closed, got %s", conn.State)
	}

	// Second close should be no-op
	conn.Close()
}

func TestConnectionUpdateActivity(t *testing.T) {
	conn := &Connection{
		ID:           "test-conn",
		LastActivity: time.Now().Add(-1 * time.Hour),
	}

	before := conn.LastActivity
	conn.UpdateActivity()
	after := conn.LastActivity

	if !after.After(before) {
		t.Error("LastActivity should be updated")
	}
}

func TestExtractSessionID(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/ws/sessions/abc123", "abc123"},
		{"/ws/sessions/abc123/", "abc123"},
		{"/gateway/sessions/xyz789", "xyz789"},
		{"/gateway/sessions/xyz789/", "xyz789"},
		{"/ws/sessions/", ""},
		{"/other/path", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := extractSessionID(tt.path)
			if result != tt.expected {
				t.Errorf("extractSessionID(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestMetrics(t *testing.T) {
	metrics := NewMetrics()

	if metrics.TotalConnections != 0 {
		t.Errorf("expected TotalConnections=0, got %d", metrics.TotalConnections)
	}
	if metrics.ActiveConnections != 0 {
		t.Errorf("expected ActiveConnections=0, got %d", metrics.ActiveConnections)
	}
	if len(metrics.ConnectionsPerUser) != 0 {
		t.Errorf("expected empty ConnectionsPerUser, got %d entries", len(metrics.ConnectionsPerUser))
	}
}

func TestNoopAuthenticator(t *testing.T) {
	auth := NewNoopAuthenticator()
	req := httptest.NewRequest("GET", "/test", nil)

	userID, err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID != "anonymous" {
		t.Errorf("expected 'anonymous', got %q", userID)
	}
}

func TestHeaderAuthenticator(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		required   bool
		headerVal  string
		wantUser   string
		wantErr    bool
	}{
		{
			name:       "valid header",
			headerName: "X-User-ID",
			required:   true,
			headerVal:  "user123",
			wantUser:   "user123",
			wantErr:    false,
		},
		{
			name:       "missing required header",
			headerName: "X-User-ID",
			required:   true,
			headerVal:  "",
			wantUser:   "",
			wantErr:    true,
		},
		{
			name:       "missing optional header",
			headerName: "X-User-ID",
			required:   false,
			headerVal:  "",
			wantUser:   "anonymous",
			wantErr:    false,
		},
		{
			name:       "whitespace header",
			headerName: "X-User-ID",
			required:   true,
			headerVal:  "  ",
			wantUser:   "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewHeaderAuthenticator(tt.headerName, tt.required)
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.headerVal != "" {
				req.Header.Set(tt.headerName, tt.headerVal)
			}

			userID, err := auth.Authenticate(req)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if userID != tt.wantUser {
				t.Errorf("got user %q, want %q", userID, tt.wantUser)
			}
		})
	}
}

func TestTokenAuthenticator(t *testing.T) {
	validateFunc := func(token string) (string, error) {
		if token == "valid-token" {
			return "user123", nil
		}
		return "", ErrUnauthorized
	}

	auth := NewTokenAuthenticator(validateFunc)

	tests := []struct {
		name     string
		authHdr  string
		wantUser string
		wantErr  bool
	}{
		{
			name:     "valid token",
			authHdr:  "Bearer valid-token",
			wantUser: "user123",
			wantErr:  false,
		},
		{
			name:     "invalid token",
			authHdr:  "Bearer invalid-token",
			wantUser: "",
			wantErr:  true,
		},
		{
			name:     "missing header",
			authHdr:  "",
			wantUser: "",
			wantErr:  true,
		},
		{
			name:     "wrong scheme",
			authHdr:  "Basic dXNlcjpwYXNz",
			wantUser: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHdr != "" {
				req.Header.Set("Authorization", tt.authHdr)
			}

			userID, err := auth.Authenticate(req)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if userID != tt.wantUser {
				t.Errorf("got user %q, want %q", userID, tt.wantUser)
			}
		})
	}
}

func TestChainAuthenticator(t *testing.T) {
	// First authenticator fails, second succeeds
	auth1 := NewHeaderAuthenticator("X-Token", true)
	auth2 := NewNoopAuthenticator()

	chain := NewChainAuthenticator(auth1, auth2)
	req := httptest.NewRequest("GET", "/test", nil)

	userID, err := chain.Authenticate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID != "anonymous" {
		t.Errorf("expected 'anonymous', got %q", userID)
	}
}

func TestGatewayServeHTTP_MissingSessionID(t *testing.T) {
	cfg := DefaultConfig()
	resolver := newMockResolver()
	gw := New(cfg, nil, resolver)

	req := httptest.NewRequest("GET", "/ws/sessions/", nil)
	w := httptest.NewRecorder()

	gw.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Missing session ID") {
		t.Errorf("expected 'Missing session ID' in body, got %q", w.Body.String())
	}
}

func TestGatewayServeHTTP_SessionNotFound(t *testing.T) {
	cfg := DefaultConfig()
	resolver := newMockResolver()
	gw := New(cfg, nil, resolver)

	req := httptest.NewRequest("GET", "/ws/sessions/nonexistent", nil)
	w := httptest.NewRecorder()

	gw.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestGatewayServeHTTP_AuthFailure(t *testing.T) {
	cfg := DefaultConfig()
	resolver := newMockResolver()
	resolver.sessions["test-session"] = "ws://localhost:5900"

	auth := NewHeaderAuthenticator("X-User-ID", true)
	gw := New(cfg, auth, resolver)

	req := httptest.NewRequest("GET", "/ws/sessions/test-session", nil)
	// Not setting X-User-ID header
	w := httptest.NewRecorder()

	gw.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	// Check metrics
	if gw.metrics.AuthFailures != 1 {
		t.Errorf("expected AuthFailures=1, got %d", gw.metrics.AuthFailures)
	}
}

// TestGatewayWebSocketConnection tests full WebSocket flow (requires real WS server)
func TestGatewayWebSocketConnection(t *testing.T) {
	// Create a mock upstream WebSocket server
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Echo messages back
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			if err := conn.WriteMessage(msgType, msg); err != nil {
				break
			}
		}
	}))
	defer upstreamServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(upstreamServer.URL, "http")

	// Create gateway
	cfg := DefaultConfig()
	resolver := newMockResolver()
	resolver.sessions["test-session"] = wsURL

	gw := New(cfg, nil, resolver)
	gw.Start()
	defer gw.Stop()

	// Create gateway HTTP server
	gatewayServer := httptest.NewServer(gw)
	defer gatewayServer.Close()

	// Connect to gateway
	gatewayWsURL := "ws" + strings.TrimPrefix(gatewayServer.URL, "http") + "/ws/sessions/test-session"
	conn, _, err := websocket.DefaultDialer.Dial(gatewayWsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to gateway: %v", err)
	}
	defer conn.Close()

	// Send a test message
	testMsg := []byte("hello gateway")
	if err := conn.WriteMessage(websocket.TextMessage, testMsg); err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	// Read the echo
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	if string(msg) != string(testMsg) {
		t.Errorf("expected %q, got %q", testMsg, msg)
	}

	// Verify metrics
	if gw.metrics.TotalConnections != 1 {
		t.Errorf("expected TotalConnections=1, got %d", gw.metrics.TotalConnections)
	}
}
