// Package gateway provides a streaming gateway service for managing
// WebSocket connections, routing, authentication, and connection lifecycle.
package gateway

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Common errors
var (
	ErrConnectionLimitExceeded = errors.New("connection limit exceeded")
	ErrRateLimitExceeded       = errors.New("rate limit exceeded")
	ErrUnauthorized            = errors.New("unauthorized")
	ErrSessionNotFound         = errors.New("session not found")
	ErrSessionNotRunning       = errors.New("session is not running")
	ErrUpstreamUnavailable     = errors.New("upstream service unavailable")
)

// ConnectionState represents the state of a gateway connection
type ConnectionState string

const (
	ConnectionStatePending    ConnectionState = "pending"
	ConnectionStateConnecting ConnectionState = "connecting"
	ConnectionStateConnected  ConnectionState = "connected"
	ConnectionStateClosing    ConnectionState = "closing"
	ConnectionStateClosed     ConnectionState = "closed"
)

// Connection represents an active streaming connection through the gateway
type Connection struct {
	ID           string
	SessionID    string
	UserID       string
	State        ConnectionState
	CreatedAt    time.Time
	ConnectedAt  time.Time
	LastActivity time.Time
	BytesSent    int64
	BytesRecv    int64
	RemoteAddr   string
	UserAgent    string

	mu       sync.RWMutex
	closed   bool
	closeCh  chan struct{}
	clientWS *websocket.Conn
	upstreamWS *websocket.Conn
}

// Config holds gateway configuration
type Config struct {
	// MaxConnectionsPerUser limits connections per user (0 = unlimited)
	MaxConnectionsPerUser int

	// MaxConnectionsTotal limits total connections (0 = unlimited)
	MaxConnectionsTotal int

	// ConnectionTimeout is the max duration for a connection (0 = unlimited)
	ConnectionTimeout time.Duration

	// IdleTimeout closes connections after inactivity (0 = unlimited)
	IdleTimeout time.Duration

	// HeartbeatInterval for connection health checks (0 = disabled)
	HeartbeatInterval time.Duration

	// RateLimitPerSecond limits requests per second per user (0 = unlimited)
	RateLimitPerSecond int

	// ReadBufferSize for WebSocket connections
	ReadBufferSize int

	// WriteBufferSize for WebSocket connections
	WriteBufferSize int

	// EnableMetrics enables connection metrics collection
	EnableMetrics bool
}

// DefaultConfig returns default gateway configuration
func DefaultConfig() Config {
	return Config{
		MaxConnectionsPerUser: 5,
		MaxConnectionsTotal:   1000,
		ConnectionTimeout:     2 * time.Hour,
		IdleTimeout:           30 * time.Minute,
		HeartbeatInterval:     30 * time.Second,
		RateLimitPerSecond:    100,
		ReadBufferSize:        4096,
		WriteBufferSize:       4096,
		EnableMetrics:         true,
	}
}

// Authenticator is an interface for connection authentication
type Authenticator interface {
	// Authenticate validates the request and returns user ID if valid
	Authenticate(r *http.Request) (userID string, err error)
}

// SessionResolver resolves session information for routing
type SessionResolver interface {
	// ResolveSession returns the upstream WebSocket URL for a session
	ResolveSession(ctx context.Context, sessionID string) (upstreamURL string, err error)

	// ValidateSession checks if a session is valid and running
	ValidateSession(ctx context.Context, sessionID string) error
}

// Metrics collects gateway metrics
type Metrics struct {
	mu sync.RWMutex

	// Connection metrics
	TotalConnections    int64
	ActiveConnections   int64
	FailedConnections   int64
	ConnectionsPerUser  map[string]int64

	// Traffic metrics
	TotalBytesSent      int64
	TotalBytesRecv      int64

	// Error metrics
	AuthFailures        int64
	RateLimitHits       int64
	UpstreamErrors      int64
}

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		ConnectionsPerUser: make(map[string]int64),
	}
}

// MetricsSnapshot is an immutable snapshot of gateway metrics
type MetricsSnapshot struct {
	TotalConnections   int64
	ActiveConnections  int64
	FailedConnections  int64
	ConnectionsPerUser map[string]int64
	TotalBytesSent     int64
	TotalBytesRecv     int64
	AuthFailures       int64
	RateLimitHits      int64
	UpstreamErrors     int64
}

// Snapshot returns a thread-safe copy of metrics
func (m *Metrics) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	connPerUser := make(map[string]int64, len(m.ConnectionsPerUser))
	for k, v := range m.ConnectionsPerUser {
		connPerUser[k] = v
	}

	return MetricsSnapshot{
		TotalConnections:   m.TotalConnections,
		ActiveConnections:  m.ActiveConnections,
		FailedConnections:  m.FailedConnections,
		ConnectionsPerUser: connPerUser,
		TotalBytesSent:     m.TotalBytesSent,
		TotalBytesRecv:     m.TotalBytesRecv,
		AuthFailures:       m.AuthFailures,
		RateLimitHits:      m.RateLimitHits,
		UpstreamErrors:     m.UpstreamErrors,
	}
}

// Gateway is the main streaming gateway service
type Gateway struct {
	config          Config
	authenticator   Authenticator
	sessionResolver SessionResolver
	metrics         *Metrics

	upgrader websocket.Upgrader

	mu          sync.RWMutex
	connections map[string]*Connection
	userConns   map[string]map[string]*Connection

	stopCh    chan struct{}
	stopped   bool
}

// New creates a new Gateway instance
func New(config Config, authenticator Authenticator, resolver SessionResolver) *Gateway {
	return &Gateway{
		config:          config,
		authenticator:   authenticator,
		sessionResolver: resolver,
		metrics:         NewMetrics(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				// TODO: Implement proper origin checking in production
				return true
			},
		},
		connections: make(map[string]*Connection),
		userConns:   make(map[string]map[string]*Connection),
		stopCh:      make(chan struct{}),
	}
}

// Start starts the gateway background processes
func (g *Gateway) Start() {
	go g.runHeartbeatLoop()
	go g.runCleanupLoop()
	log.Printf("Gateway started with config: MaxConnsTotal=%d, MaxConnsPerUser=%d",
		g.config.MaxConnectionsTotal, g.config.MaxConnectionsPerUser)
}

// Stop stops the gateway and closes all connections
func (g *Gateway) Stop() {
	g.mu.Lock()
	if g.stopped {
		g.mu.Unlock()
		return
	}
	g.stopped = true
	close(g.stopCh)

	// Close all connections
	for _, conn := range g.connections {
		conn.Close()
	}
	g.mu.Unlock()

	log.Println("Gateway stopped")
}

// GetMetrics returns current gateway metrics
func (g *Gateway) GetMetrics() *Metrics {
	return g.metrics
}

// GetConnection returns a connection by ID
func (g *Gateway) GetConnection(id string) *Connection {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.connections[id]
}

// ListConnections returns all active connections
func (g *Gateway) ListConnections() []*Connection {
	g.mu.RLock()
	defer g.mu.RUnlock()

	conns := make([]*Connection, 0, len(g.connections))
	for _, conn := range g.connections {
		conns = append(conns, conn)
	}
	return conns
}

// ConnectionCount returns the total number of active connections
func (g *Gateway) ConnectionCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.connections)
}

// UserConnectionCount returns connections for a specific user
func (g *Gateway) UserConnectionCount(userID string) int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if userConns, ok := g.userConns[userID]; ok {
		return len(userConns)
	}
	return 0
}

// Close closes a connection
func (c *Connection) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.State = ConnectionStateClosed
	close(c.closeCh)

	if c.clientWS != nil {
		c.clientWS.Close()
	}
	if c.upstreamWS != nil {
		c.upstreamWS.Close()
	}
	c.mu.Unlock()
}

// IsClosed returns whether the connection is closed
func (c *Connection) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// UpdateActivity updates the last activity timestamp
func (c *Connection) UpdateActivity() {
	c.mu.Lock()
	c.LastActivity = time.Now()
	c.mu.Unlock()
}

// Snapshot returns a thread-safe copy of connection stats
func (c *Connection) Snapshot() ConnectionSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return ConnectionSnapshot{
		ID:           c.ID,
		SessionID:    c.SessionID,
		UserID:       c.UserID,
		State:        c.State,
		CreatedAt:    c.CreatedAt,
		ConnectedAt:  c.ConnectedAt,
		LastActivity: c.LastActivity,
		BytesSent:    c.BytesSent,
		BytesRecv:    c.BytesRecv,
		RemoteAddr:   c.RemoteAddr,
		UserAgent:    c.UserAgent,
	}
}

// ConnectionSnapshot is an immutable snapshot of connection state
type ConnectionSnapshot struct {
	ID           string
	SessionID    string
	UserID       string
	State        ConnectionState
	CreatedAt    time.Time
	ConnectedAt  time.Time
	LastActivity time.Time
	BytesSent    int64
	BytesRecv    int64
	RemoteAddr   string
	UserAgent    string
}

// runHeartbeatLoop periodically checks connection health
func (g *Gateway) runHeartbeatLoop() {
	if g.config.HeartbeatInterval <= 0 {
		return
	}

	ticker := time.NewTicker(g.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-g.stopCh:
			return
		case <-ticker.C:
			g.checkConnections()
		}
	}
}

// runCleanupLoop periodically cleans up stale connections
func (g *Gateway) runCleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-g.stopCh:
			return
		case <-ticker.C:
			g.cleanupStaleConnections()
		}
	}
}

// checkConnections sends heartbeats to all connections
func (g *Gateway) checkConnections() {
	g.mu.RLock()
	conns := make([]*Connection, 0, len(g.connections))
	for _, conn := range g.connections {
		conns = append(conns, conn)
	}
	g.mu.RUnlock()

	for _, conn := range conns {
		if conn.IsClosed() {
			continue
		}

		conn.mu.RLock()
		ws := conn.clientWS
		conn.mu.RUnlock()

		if ws != nil {
			if err := ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				log.Printf("Heartbeat failed for connection %s: %v", conn.ID, err)
				g.removeConnection(conn)
			}
		}
	}
}

// cleanupStaleConnections removes connections that have timed out
func (g *Gateway) cleanupStaleConnections() {
	now := time.Now()

	g.mu.RLock()
	conns := make([]*Connection, 0, len(g.connections))
	for _, conn := range g.connections {
		conns = append(conns, conn)
	}
	g.mu.RUnlock()

	for _, conn := range conns {
		if conn.IsClosed() {
			g.removeConnection(conn)
			continue
		}

		conn.mu.RLock()
		lastActivity := conn.LastActivity
		createdAt := conn.CreatedAt
		conn.mu.RUnlock()

		// Check idle timeout
		if g.config.IdleTimeout > 0 && now.Sub(lastActivity) > g.config.IdleTimeout {
			log.Printf("Connection %s idle timeout", conn.ID)
			conn.Close()
			g.removeConnection(conn)
			continue
		}

		// Check connection timeout
		if g.config.ConnectionTimeout > 0 && now.Sub(createdAt) > g.config.ConnectionTimeout {
			log.Printf("Connection %s connection timeout", conn.ID)
			conn.Close()
			g.removeConnection(conn)
		}
	}
}

// addConnection adds a connection to the gateway
func (g *Gateway) addConnection(conn *Connection) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Check total connection limit
	if g.config.MaxConnectionsTotal > 0 && len(g.connections) >= g.config.MaxConnectionsTotal {
		return ErrConnectionLimitExceeded
	}

	// Check per-user connection limit
	if g.config.MaxConnectionsPerUser > 0 {
		if userConns, ok := g.userConns[conn.UserID]; ok {
			if len(userConns) >= g.config.MaxConnectionsPerUser {
				return ErrConnectionLimitExceeded
			}
		}
	}

	g.connections[conn.ID] = conn

	if g.userConns[conn.UserID] == nil {
		g.userConns[conn.UserID] = make(map[string]*Connection)
	}
	g.userConns[conn.UserID][conn.ID] = conn

	// Update metrics
	g.metrics.mu.Lock()
	g.metrics.TotalConnections++
	g.metrics.ActiveConnections++
	g.metrics.ConnectionsPerUser[conn.UserID]++
	g.metrics.mu.Unlock()

	return nil
}

// removeConnection removes a connection from the gateway
func (g *Gateway) removeConnection(conn *Connection) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.connections, conn.ID)

	if userConns, ok := g.userConns[conn.UserID]; ok {
		delete(userConns, conn.ID)
		if len(userConns) == 0 {
			delete(g.userConns, conn.UserID)
		}
	}

	// Update metrics
	g.metrics.mu.Lock()
	g.metrics.ActiveConnections--
	if g.metrics.ConnectionsPerUser[conn.UserID] > 0 {
		g.metrics.ConnectionsPerUser[conn.UserID]--
	}
	g.metrics.mu.Unlock()
}

// ServeHTTP handles incoming WebSocket connections
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract session ID from URL path
	sessionID := extractSessionID(r.URL.Path)
	if sessionID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	// Authenticate the request
	userID := "anonymous"
	if g.authenticator != nil {
		var err error
		userID, err = g.authenticator.Authenticate(r)
		if err != nil {
			g.metrics.mu.Lock()
			g.metrics.AuthFailures++
			g.metrics.mu.Unlock()
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Validate and resolve the session
	if err := g.sessionResolver.ValidateSession(r.Context(), sessionID); err != nil {
		switch {
		case errors.Is(err, ErrSessionNotFound):
			http.Error(w, "Session not found", http.StatusNotFound)
		case errors.Is(err, ErrSessionNotRunning):
			http.Error(w, "Session is not running", http.StatusBadRequest)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	upstreamURL, err := g.sessionResolver.ResolveSession(r.Context(), sessionID)
	if err != nil {
		g.metrics.mu.Lock()
		g.metrics.UpstreamErrors++
		g.metrics.mu.Unlock()
		http.Error(w, "Upstream unavailable", http.StatusServiceUnavailable)
		return
	}

	// Create connection record
	conn := &Connection{
		ID:           fmt.Sprintf("%s-%d", sessionID, time.Now().UnixNano()),
		SessionID:    sessionID,
		UserID:       userID,
		State:        ConnectionStatePending,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		RemoteAddr:   r.RemoteAddr,
		UserAgent:    r.UserAgent(),
		closeCh:      make(chan struct{}),
	}

	// Add connection (checks limits)
	if err := g.addConnection(conn); err != nil {
		g.metrics.mu.Lock()
		g.metrics.FailedConnections++
		g.metrics.mu.Unlock()
		if errors.Is(err, ErrConnectionLimitExceeded) {
			http.Error(w, "Connection limit exceeded", http.StatusTooManyRequests)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Upgrade to WebSocket
	conn.State = ConnectionStateConnecting
	clientWS, err := g.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		g.removeConnection(conn)
		return
	}
	conn.clientWS = clientWS

	// Connect to upstream
	upstreamWS, _, err := websocket.DefaultDialer.Dial(upstreamURL, nil)
	if err != nil {
		log.Printf("Failed to connect to upstream %s: %v", upstreamURL, err)
		g.metrics.mu.Lock()
		g.metrics.UpstreamErrors++
		g.metrics.mu.Unlock()
		clientWS.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "Upstream unavailable"))
		clientWS.Close()
		g.removeConnection(conn)
		return
	}
	conn.upstreamWS = upstreamWS
	conn.State = ConnectionStateConnected
	conn.ConnectedAt = time.Now()

	log.Printf("Gateway connection established: %s (session=%s, user=%s)", conn.ID, sessionID, userID)

	// Start bidirectional proxying
	g.proxyConnection(conn)

	// Cleanup
	conn.Close()
	g.removeConnection(conn)
	log.Printf("Gateway connection closed: %s", conn.ID)
}

// proxyConnection handles bidirectional WebSocket proxying
func (g *Gateway) proxyConnection(conn *Connection) {
	done := make(chan struct{}, 2)

	// Client -> Upstream
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			select {
			case <-conn.closeCh:
				return
			default:
			}

			msgType, msg, err := conn.clientWS.ReadMessage()
			if err != nil {
				return
			}

			conn.UpdateActivity()
			conn.mu.Lock()
			conn.BytesRecv += int64(len(msg))
			conn.mu.Unlock()

			if err := conn.upstreamWS.WriteMessage(msgType, msg); err != nil {
				return
			}
		}
	}()

	// Upstream -> Client
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			select {
			case <-conn.closeCh:
				return
			default:
			}

			msgType, msg, err := conn.upstreamWS.ReadMessage()
			if err != nil {
				return
			}

			conn.UpdateActivity()
			conn.mu.Lock()
			conn.BytesSent += int64(len(msg))
			conn.mu.Unlock()

			// Update global metrics
			g.metrics.mu.Lock()
			g.metrics.TotalBytesSent += int64(len(msg))
			g.metrics.mu.Unlock()

			if err := conn.clientWS.WriteMessage(msgType, msg); err != nil {
				return
			}
		}
	}()

	// Wait for either direction to finish
	<-done
}

// extractSessionID extracts the session ID from the URL path
func extractSessionID(path string) string {
	// Expected format: /ws/sessions/{id} or /gateway/sessions/{id}
	parts := make([]string, 0)
	for _, p := range []string{"/ws/sessions/", "/gateway/sessions/"} {
		if len(path) > len(p) && path[:len(p)] == p {
			parts = append(parts, path[len(p):])
		}
	}
	if len(parts) > 0 {
		// Remove trailing slash if present
		id := parts[0]
		if len(id) > 0 && id[len(id)-1] == '/' {
			id = id[:len(id)-1]
		}
		return id
	}
	return ""
}
