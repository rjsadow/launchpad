package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/rjsadow/launchpad/internal/auth"
	"github.com/rjsadow/launchpad/internal/store"
)

// FavoritesHandler handles favorites-related endpoints
type FavoritesHandler struct {
	db *store.DB
}

// NewFavoritesHandler creates a new favorites handler
func NewFavoritesHandler(db *store.DB) *FavoritesHandler {
	return &FavoritesHandler{db: db}
}

// HandleGetFavorites returns the user's favorite apps
func (h *FavoritesHandler) HandleGetFavorites(w http.ResponseWriter, r *http.Request) {
	session := auth.GetSessionFromContext(r.Context())
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	favorites, err := h.db.GetFavorites(session.UserID)
	if err != nil {
		log.Printf("Failed to get favorites: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get full app info for favorites
	apps, err := h.db.GetApps()
	if err != nil {
		log.Printf("Failed to get apps: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Filter to only favorite apps
	favMap := make(map[string]bool)
	for _, f := range favorites {
		favMap[f.AppID] = true
	}

	var favoriteApps []interface{}
	filteredApps := store.FilterAppsByRoles(apps, session.Roles, session.IsAdmin)
	for _, app := range filteredApps {
		if favMap[app.ID] {
			app.IsFavorite = true
			favoriteApps = append(favoriteApps, app)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(favoriteApps)
}

// HandleAddFavorite adds an app to favorites
func (h *FavoritesHandler) HandleAddFavorite(w http.ResponseWriter, r *http.Request) {
	session := auth.GetSessionFromContext(r.Context())
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	appID := extractAppID(r.URL.Path)
	if appID == "" {
		http.Error(w, "Missing app ID", http.StatusBadRequest)
		return
	}

	// Verify app exists
	app, err := h.db.GetApp(appID)
	if err != nil {
		log.Printf("Failed to get app: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if app == nil {
		http.Error(w, "App not found", http.StatusNotFound)
		return
	}

	if err := h.db.AddFavorite(session.UserID, appID); err != nil {
		log.Printf("Failed to add favorite: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// HandleRemoveFavorite removes an app from favorites
func (h *FavoritesHandler) HandleRemoveFavorite(w http.ResponseWriter, r *http.Request) {
	session := auth.GetSessionFromContext(r.Context())
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	appID := extractAppID(r.URL.Path)
	if appID == "" {
		http.Error(w, "Missing app ID", http.StatusBadRequest)
		return
	}

	if err := h.db.RemoveFavorite(session.UserID, appID); err != nil {
		log.Printf("Failed to remove favorite: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// extractAppID extracts the app ID from a path like /api/favorites/{appId}
func extractAppID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 {
		return parts[2] // api/favorites/{appId}
	}
	return ""
}
