package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rjsadow/launchpad/internal/db"
)

// Handlers provides HTTP handlers for authentication endpoints.
type Handlers struct {
	db         *db.DB
	jwtService *JWTService
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(database *db.DB, jwtService *JWTService) *Handlers {
	return &Handlers{
		db:         database,
		jwtService: jwtService,
	}
}

// RegisterRequest represents a user registration request.
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents a user login request.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse represents an authentication response with a token.
type AuthResponse struct {
	Token string  `json:"token"`
	User  UserDTO `json:"user"`
}

// UserDTO represents user data for API responses (without sensitive fields).
type UserDTO struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// HandleRegister handles user registration.
func (h *Handlers) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Username, email, and password are required"})
		return
	}

	if len(req.Password) < 8 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Password must be at least 8 characters"})
		return
	}

	if !strings.Contains(req.Email, "@") {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid email format"})
		return
	}

	// Check if user already exists
	exists, err := h.db.UserExists(req.Username, req.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Database error"})
		return
	}
	if exists {
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: "Username or email already exists"})
		return
	}

	// Hash password
	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to hash password"})
		return
	}

	// Create user
	now := time.Now()
	user := db.User{
		ID:           uuid.New().String(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         db.UserRoleUser,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.db.CreateUser(user); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to create user"})
		return
	}

	// Generate token
	token, err := h.jwtService.GenerateToken(user.ID, user.Username, string(user.Role))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to generate token"})
		return
	}

	// Log audit
	h.db.LogAudit(user.Username, "user_registered", "User registered: "+user.Email)

	writeJSON(w, http.StatusCreated, AuthResponse{
		Token: token,
		User:  userToDTO(user),
	})
}

// HandleLogin handles user login.
func (h *Handlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Username and password are required"})
		return
	}

	// Find user
	user, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Database error"})
		return
	}
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Invalid credentials"})
		return
	}

	// Check password
	if err := CheckPassword(req.Password, user.PasswordHash); err != nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Invalid credentials"})
		return
	}

	// Generate token
	token, err := h.jwtService.GenerateToken(user.ID, user.Username, string(user.Role))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to generate token"})
		return
	}

	// Log audit
	h.db.LogAudit(user.Username, "user_login", "User logged in")

	writeJSON(w, http.StatusOK, AuthResponse{
		Token: token,
		User:  userToDTO(*user),
	})
}

// HandleMe returns the current authenticated user's information.
func (h *Handlers) HandleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok || userID == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Not authenticated"})
		return
	}

	user, err := h.db.GetUser(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Database error"})
		return
	}
	if user == nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "User not found"})
		return
	}

	writeJSON(w, http.StatusOK, userToDTO(*user))
}

// userToDTO converts a User to a UserDTO.
func userToDTO(user db.User) UserDTO {
	return UserDTO{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
