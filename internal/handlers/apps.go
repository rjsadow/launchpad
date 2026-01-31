package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/rjsadow/launchpad/internal/auth"
	"github.com/rjsadow/launchpad/internal/store"
)

// AppsHandler handles application-related endpoints
type AppsHandler struct {
	db *store.DB
}

// NewAppsHandler creates a new apps handler
func NewAppsHandler(db *store.DB) *AppsHandler {
	return &AppsHandler{db: db}
}

// HandleGetApps returns the list of applications filtered by user roles
func (h *AppsHandler) HandleGetApps(w http.ResponseWriter, r *http.Request) {
	session := auth.GetSessionFromContext(r.Context())

	apps, err := h.db.GetApps()
	if err != nil {
		log.Printf("Failed to get apps: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// If no session (unauthenticated), only show apps without role restrictions
	var userRoles []string
	isAdmin := false
	userID := ""

	if session != nil {
		userRoles = session.Roles
		isAdmin = session.IsAdmin
		userID = session.UserID
	}

	// Filter by roles
	filteredApps := store.FilterAppsByRoles(apps, userRoles, isAdmin)

	// Mark favorites if user is authenticated
	if userID != "" {
		favs, err := h.db.GetFavoriteAppIDs(userID)
		if err != nil {
			log.Printf("Failed to get favorites: %v", err)
		} else {
			for i := range filteredApps {
				filteredApps[i].IsFavorite = favs[filteredApps[i].ID]
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filteredApps)
}

// HandleGetMe returns the current user info
func (h *AppsHandler) HandleGetMe(w http.ResponseWriter, r *http.Request) {
	session := auth.GetSessionFromContext(r.Context())
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user := session.ToUser()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user.ToUserInfo())
}
