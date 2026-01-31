package handlers

import (
	"log"
	"net/http"

	"github.com/rjsadow/launchpad/internal/auth"
	"github.com/rjsadow/launchpad/internal/store"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	oidc     *auth.OIDCProvider
	sessions *auth.SessionManager
	db       *store.DB
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(oidc *auth.OIDCProvider, sessions *auth.SessionManager, db *store.DB) *AuthHandler {
	return &AuthHandler{
		oidc:     oidc,
		sessions: sessions,
		db:       db,
	}
}

// HandleLogin initiates the OIDC login flow
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if h.oidc == nil {
		http.Error(w, "Authentication not configured", http.StatusNotImplemented)
		return
	}

	state, err := auth.GenerateState()
	if err != nil {
		log.Printf("Failed to generate state: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	auth.SetStateCookie(w, state)
	http.Redirect(w, r, h.oidc.GetAuthURL(state), http.StatusTemporaryRedirect)
}

// HandleCallback handles the OIDC callback
func (h *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	if h.oidc == nil {
		http.Error(w, "Authentication not configured", http.StatusNotImplemented)
		return
	}

	// Verify state
	expectedState, err := auth.GetStateCookie(w, r)
	if err != nil {
		log.Printf("Missing state cookie: %v", err)
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != expectedState {
		http.Error(w, "State mismatch", http.StatusBadRequest)
		return
	}

	// Check for error from provider
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		log.Printf("OIDC error: %s - %s", errMsg, r.URL.Query().Get("error_description"))
		http.Error(w, "Authentication failed: "+errMsg, http.StatusBadRequest)
		return
	}

	// Exchange code for tokens
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	user, err := h.oidc.ExchangeCode(r.Context(), code)
	if err != nil {
		log.Printf("Failed to exchange code: %v", err)
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	// Store/update user in database
	if err := h.db.UpsertUser(user); err != nil {
		log.Printf("Failed to save user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create session
	if err := h.sessions.CreateSession(w, user); err != nil {
		log.Printf("Failed to create session: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Redirect to home
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// HandleLogout clears the session
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	h.sessions.ClearSession(w)

	// Return JSON response for API calls
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success": true}`))
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
