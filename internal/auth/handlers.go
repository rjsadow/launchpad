package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rjsadow/launchpad/internal/db"
)

// Handler provides HTTP handlers for authentication endpoints
type Handler struct {
	db           *db.DB
	tokenService *TokenService
	enabled      bool
}

// NewHandler creates a new auth handler
func NewHandler(database *db.DB, tokenService *TokenService, enabled bool) *Handler {
	return &Handler{
		db:           database,
		tokenService: tokenService,
		enabled:      enabled,
	}
}

// LoginRequest represents a login request body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
}

// RegisterRequest represents a registration request body
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password"`
}

// HandleLogin handles POST /api/auth/login
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.enabled {
		http.Error(w, "Authentication is disabled", http.StatusNotFound)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Validate credentials
	user, err := h.db.ValidateUserCredentials(req.Username, req.Password)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Generate token
	token, expiresAt, err := h.tokenService.GenerateToken(user.ID, user.Username, user.IsAdmin)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     h.tokenService.CookieName(),
		Value:    token,
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	// Log audit
	h.db.LogAudit(user.Username, "LOGIN", "User logged in")

	// Return response
	resp := LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User: UserInfo{
			UserID:   user.ID,
			Username: user.Username,
			IsAdmin:  user.IsAdmin,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleLogout handles POST /api/auth/logout
func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Clear the cookie
	http.SetCookie(w, &http.Cookie{
		Name:     h.tokenService.CookieName(),
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	// Log audit if user is authenticated
	user := GetUser(r)
	if user != nil {
		h.db.LogAudit(user.Username, "LOGOUT", "User logged out")
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
}

// HandleMe handles GET /api/auth/me
func (h *Handler) HandleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := GetUser(r)
	if user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// HandleRegister handles POST /api/auth/register (admin only)
func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.enabled {
		http.Error(w, "Authentication is disabled", http.StatusNotFound)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Check if username already exists
	existingUser, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existingUser != nil {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	// Create user
	newUser := db.User{
		ID:       uuid.New().String(),
		Username: req.Username,
		Email:    req.Email,
		IsAdmin:  false, // Regular users are not admins by default
	}

	if err := h.db.CreateUser(newUser, req.Password); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Log audit
	adminUser := GetUser(r)
	if adminUser != nil {
		h.db.LogAudit(adminUser.Username, "CREATE_USER", "Created user: "+req.Username)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       newUser.ID,
		"username": newUser.Username,
		"email":    newUser.Email,
		"is_admin": newUser.IsAdmin,
	})
}

// HandleAuthStatus handles GET /api/auth/status
// Returns whether auth is enabled (public endpoint)
func (h *Handler) HandleAuthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"enabled": h.enabled})
}

// EnsureDefaultAdmin creates a default admin user if no users exist
func (h *Handler) EnsureDefaultAdmin(username, password string) error {
	if username == "" || password == "" {
		return nil
	}

	count, err := h.db.UserCount()
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // Users already exist
	}

	// Create default admin user
	adminUser := db.User{
		ID:       uuid.New().String(),
		Username: username,
		IsAdmin:  true,
	}

	if err := h.db.CreateUser(adminUser, password); err != nil {
		return err
	}

	h.db.LogAudit("system", "CREATE_USER", "Created default admin user: "+username)
	return nil
}
