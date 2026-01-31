package sessions

import (
	"time"

	"github.com/rjsadow/launchpad/internal/db"
)

// CreateSessionRequest represents a request to create a new session
type CreateSessionRequest struct {
	AppID       string `json:"app_id"`
	UserID      string `json:"user_id"`
	TTL         int64  `json:"ttl,omitempty"`          // Session time-to-live in seconds (0 = use default)
	IdleTimeout int64  `json:"idle_timeout,omitempty"` // Idle timeout in seconds (0 = use default)
}

// SessionResponse represents a session in API responses
type SessionResponse struct {
	ID             string           `json:"id"`
	UserID         string           `json:"user_id"`
	AppID          string           `json:"app_id"`
	AppName        string           `json:"app_name,omitempty"`
	PodName        string           `json:"pod_name"`
	Status         db.SessionStatus `json:"status"`
	WebSocketURL   string           `json:"websocket_url,omitempty"`
	TTL            int64            `json:"ttl"`                        // Session time-to-live in seconds
	IdleTimeout    int64            `json:"idle_timeout"`               // Idle timeout in seconds
	ExpiresAt      *time.Time       `json:"expires_at,omitempty"`       // Absolute expiration time
	LastActivityAt time.Time        `json:"last_activity_at"`           // Last activity timestamp
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// SessionFromDB converts a database session to an API response
func SessionFromDB(session *db.Session, appName string, wsURL string) *SessionResponse {
	return &SessionResponse{
		ID:             session.ID,
		UserID:         session.UserID,
		AppID:          session.AppID,
		AppName:        appName,
		PodName:        session.PodName,
		Status:         session.Status,
		WebSocketURL:   wsURL,
		TTL:            session.TTL,
		IdleTimeout:    session.IdleTimeout,
		ExpiresAt:      session.ExpiresAt,
		LastActivityAt: session.LastActivityAt,
		CreatedAt:      session.CreatedAt,
		UpdatedAt:      session.UpdatedAt,
	}
}
