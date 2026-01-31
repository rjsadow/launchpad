package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/rjsadow/launchpad/internal/models"
	"github.com/rjsadow/launchpad/internal/store"
)

// AdminHandler handles admin-related endpoints
type AdminHandler struct {
	db *store.DB
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(db *store.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

// HandleGetAllApps returns all apps (including disabled) for admin
func (h *AdminHandler) HandleGetAllApps(w http.ResponseWriter, r *http.Request) {
	apps, err := h.db.GetAllApps()
	if err != nil {
		log.Printf("Failed to get apps: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apps)
}

// HandleCreateApp creates a new application
func (h *AdminHandler) HandleCreateApp(w http.ResponseWriter, r *http.Request) {
	var app models.Application
	if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if app.ID == "" || app.Name == "" || app.URL == "" || app.Category == "" {
		http.Error(w, "Missing required fields: id, name, url, category", http.StatusBadRequest)
		return
	}

	// Check if ID already exists
	existing, err := h.db.GetApp(app.ID)
	if err != nil {
		log.Printf("Failed to check app: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		http.Error(w, "Application with this ID already exists", http.StatusConflict)
		return
	}

	// Default enabled to true for new apps
	app.Enabled = true

	if err := h.db.CreateApp(&app); err != nil {
		log.Printf("Failed to create app: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(app)
}

// HandleUpdateApp updates an existing application
func (h *AdminHandler) HandleUpdateApp(w http.ResponseWriter, r *http.Request) {
	appID := extractAdminAppID(r.URL.Path)
	if appID == "" {
		http.Error(w, "Missing app ID", http.StatusBadRequest)
		return
	}

	var app models.Application
	if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Ensure ID matches URL
	app.ID = appID

	// Validate required fields
	if app.Name == "" || app.URL == "" || app.Category == "" {
		http.Error(w, "Missing required fields: name, url, category", http.StatusBadRequest)
		return
	}

	if err := h.db.UpdateApp(&app); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Application not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to update app: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app)
}

// HandleDeleteApp deletes an application
func (h *AdminHandler) HandleDeleteApp(w http.ResponseWriter, r *http.Request) {
	appID := extractAdminAppID(r.URL.Path)
	if appID == "" {
		http.Error(w, "Missing app ID", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteApp(appID); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Application not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to delete app: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// extractAdminAppID extracts the app ID from a path like /api/apps/{appId}
func extractAdminAppID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 {
		return parts[2] // api/apps/{appId}
	}
	return ""
}
