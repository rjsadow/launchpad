package auth

import (
	"encoding/gob"
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/rjsadow/launchpad/internal/models"
)

const (
	sessionCookieName = "launchpad_session"
	sessionMaxAge     = 24 * time.Hour
)

func init() {
	// Register types for gob encoding
	gob.Register(&SessionData{})
}

// SessionData holds the session information
type SessionData struct {
	UserID    string
	Email     string
	Name      string
	Roles     []string
	IsAdmin   bool
	ExpiresAt time.Time
}

// SessionManager handles session creation and validation
type SessionManager struct {
	codec *securecookie.SecureCookie
}

// NewSessionManager creates a new session manager with the given secret
func NewSessionManager(secret []byte) *SessionManager {
	// Use the secret for both hash and block keys
	// Hash key should be 32 or 64 bytes, block key 16, 24, or 32 bytes
	hashKey := secret
	if len(hashKey) < 32 {
		// Pad if too short
		padded := make([]byte, 32)
		copy(padded, hashKey)
		hashKey = padded
	} else if len(hashKey) > 64 {
		hashKey = hashKey[:64]
	}

	// Block key for encryption (optional, use first 32 bytes of secret)
	var blockKey []byte
	if len(secret) >= 32 {
		blockKey = secret[:32]
	}

	return &SessionManager{
		codec: securecookie.New(hashKey, blockKey),
	}
}

// CreateSession creates a new session cookie for the user
func (sm *SessionManager) CreateSession(w http.ResponseWriter, user *models.User) error {
	session := &SessionData{
		UserID:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Roles:     user.Roles,
		IsAdmin:   user.IsAdmin,
		ExpiresAt: time.Now().Add(sessionMaxAge),
	}

	encoded, err := sm.codec.Encode(sessionCookieName, session)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionMaxAge.Seconds()),
	})

	return nil
}

// GetSession retrieves the session from the request
func (sm *SessionManager) GetSession(r *http.Request) (*SessionData, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, err
	}

	var session SessionData
	if err := sm.codec.Decode(sessionCookieName, cookie.Value, &session); err != nil {
		return nil, err
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		return nil, http.ErrNoCookie
	}

	return &session, nil
}

// ClearSession removes the session cookie
func (sm *SessionManager) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// ToUser converts session data to a User model
func (sd *SessionData) ToUser() *models.User {
	return &models.User{
		ID:      sd.UserID,
		Email:   sd.Email,
		Name:    sd.Name,
		Roles:   sd.Roles,
		IsAdmin: sd.IsAdmin,
	}
}
