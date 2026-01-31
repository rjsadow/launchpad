package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/rjsadow/launchpad/internal/config"
	"github.com/rjsadow/launchpad/internal/db"
	"github.com/rjsadow/launchpad/internal/k8s"
	"github.com/rjsadow/launchpad/internal/middleware"
	"github.com/rjsadow/launchpad/internal/rbac"
	"github.com/rjsadow/launchpad/internal/sessions"
	"github.com/rjsadow/launchpad/internal/websocket"
)

//go:embed web/dist/*
var embeddedFiles embed.FS

var database *db.DB
var sessionManager *sessions.Manager
var appConfig *config.Config
var rbacStore *rbac.DBStore
var rbacMiddleware *rbac.Middleware

func main() {
	// Parse command-line flags (can override env vars)
	port := flag.Int("port", config.DefaultPort, "Port to listen on")
	dbPath := flag.String("db", config.DefaultDBPath, "Path to SQLite database")
	seedPath := flag.String("seed", "", "Path to apps.json for initial seeding")
	flag.Parse()

	// Load configuration (env vars + flag overrides)
	var err error
	appConfig, err = config.LoadWithFlags(*port, *dbPath, *seedPath)
	if err != nil {
		log.Fatalf("Configuration error:\n%v\n\nSee .env.example for configuration options.", err)
	}

	// Initialize Kubernetes configuration
	k8s.Configure(appConfig.Namespace, appConfig.Kubeconfig, appConfig.VNCSidecarImage)

	// Initialize database
	database, err = db.Open(appConfig.DB)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer database.Close()

	// Seed from JSON if provided and database is empty
	if appConfig.Seed != "" {
		if err := database.SeedFromJSON(appConfig.Seed); err != nil {
			log.Printf("Warning: failed to seed from JSON: %v", err)
		}
	}

	// Initialize RBAC store and middleware
	rbacStore, err = rbac.NewDBStore(database.Conn())
	if err != nil {
		log.Fatal("Failed to initialize RBAC store:", err)
	}

	rbacMiddleware = rbac.NewMiddleware(rbacStore,
		rbac.WithBypassPaths([]string{
			"/",
			"/index.html",
			"/assets/*",
			"/favicon.ico",
		}),
		rbac.WithAuthEnabled(true),
	)

	// Initialize session manager with config
	sessionManager = sessions.NewManagerWithConfig(database, sessions.ManagerConfig{
		SessionTimeout:  appConfig.SessionTimeout,
		CleanupInterval: appConfig.SessionCleanupInterval,
		PodReadyTimeout: appConfig.PodReadyTimeout,
	})
	sessionManager.Start()
	defer sessionManager.Stop()

	// Initialize WebSocket handler
	wsHandler := websocket.NewHandler(sessionManager)

	// Get the subdirectory from the embedded filesystem
	distFS, err := fs.Sub(embeddedFiles, "web/dist")
	if err != nil {
		log.Fatal("Failed to access embedded files:", err)
	}

	// Create file server handler
	fileServer := http.FileServer(http.FS(distFS))

	// Create a new ServeMux for API routes with RBAC
	mux := http.NewServeMux()

	// API routes with RBAC enforcement
	mux.HandleFunc("/api/apps", handleApps)
	mux.HandleFunc("/api/apps/", handleAppByID)
	mux.HandleFunc("/api/audit", handleAuditLogs)
	mux.HandleFunc("/api/analytics/launch", handleAnalyticsLaunch)
	mux.HandleFunc("/api/analytics/stats", handleAnalyticsStats)
	mux.HandleFunc("/api/config", handleConfig)

	// Session API routes
	mux.HandleFunc("/api/sessions", handleSessions)
	mux.HandleFunc("/api/sessions/", handleSessionByID)

	// RBAC management routes (admin only)
	mux.HandleFunc("/api/rbac/roles", handleRBACRoles)
	mux.HandleFunc("/api/rbac/users", handleRBACUsers)
	mux.HandleFunc("/api/rbac/users/", handleRBACUserByID)

	// WebSocket route for session VNC streams
	mux.Handle("/ws/sessions/", wsHandler)

	// Serve apps.json from database (for frontend compatibility)
	mux.HandleFunc("/apps.json", handleAppsJSON)

	// Handle static files and SPA routing
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if file exists
		if _, err := fs.Stat(distFS, path[1:]); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// For SPA routing, serve index.html for non-existent paths
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf(":%d", appConfig.Port)
	log.Printf("Launchpad server starting on http://localhost%s", addr)
	log.Printf("RBAC enabled - roles: admin, app-author, user")

	// Chain middleware: security headers -> RBAC authentication
	handler := middleware.SecurityHeaders(rbacMiddleware.Authenticate(mux))

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal("Server error:", err)
		os.Exit(1)
	}
}

// handleAppsJSON serves apps in the legacy JSON format for frontend compatibility
func handleAppsJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apps, err := database.ListApps()
	if err != nil {
		log.Printf("Error listing apps: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return in the format the frontend expects
	response := db.AppConfig{Applications: apps}
	if response.Applications == nil {
		response.Applications = []db.Application{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleApps handles GET (list) and POST (create) for /api/apps
func handleApps(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Requires app:read permission
		if !rbac.CheckPermission(w, r, rbac.PermissionAppRead) {
			return
		}

		apps, err := database.ListApps()
		if err != nil {
			log.Printf("Error listing apps: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if apps == nil {
			apps = []db.Application{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apps)

	case http.MethodPost:
		// Requires app:create permission (admin or app-author)
		if !rbac.CheckPermission(w, r, rbac.PermissionAppCreate) {
			return
		}

		var app db.Application
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		if err := json.Unmarshal(body, &app); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if app.ID == "" || app.Name == "" || app.URL == "" {
			http.Error(w, "Missing required fields: id, name, url", http.StatusBadRequest)
			return
		}

		if err := database.CreateApp(app); err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				http.Error(w, "Application with this ID already exists", http.StatusConflict)
				return
			}
			log.Printf("Error creating app: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Log the action with actual user
		user := rbac.GetUserFromRequest(r)
		userID := "unknown"
		if user != nil {
			userID = user.ID
		}
		details := fmt.Sprintf("Created app: %s (%s)", app.Name, app.ID)
		database.LogAudit(userID, "CREATE_APP", details)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(app)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAppByID handles GET, PUT, DELETE for /api/apps/{id}
func handleAppByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	id := strings.TrimPrefix(r.URL.Path, "/api/apps/")
	if id == "" {
		http.Error(w, "Missing app ID", http.StatusBadRequest)
		return
	}

	// Get user for audit logging
	user := rbac.GetUserFromRequest(r)
	userID := "unknown"
	if user != nil {
		userID = user.ID
	}

	switch r.Method {
	case http.MethodGet:
		// Requires app:read permission
		if !rbac.CheckPermission(w, r, rbac.PermissionAppRead) {
			return
		}

		app, err := database.GetApp(id)
		if err != nil {
			log.Printf("Error getting app: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if app == nil {
			http.Error(w, "Application not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)

	case http.MethodPut:
		// Requires app:update permission (admin or app-author)
		if !rbac.CheckPermission(w, r, rbac.PermissionAppUpdate) {
			return
		}

		var app db.Application
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		if err := json.Unmarshal(body, &app); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Use ID from URL path
		app.ID = id

		if app.Name == "" || app.URL == "" {
			http.Error(w, "Missing required fields: name, url", http.StatusBadRequest)
			return
		}

		if err := database.UpdateApp(app); err != nil {
			if err.Error() == "sql: no rows in result set" {
				http.Error(w, "Application not found", http.StatusNotFound)
				return
			}
			log.Printf("Error updating app: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Log the action
		details := fmt.Sprintf("Updated app: %s (%s)", app.Name, app.ID)
		database.LogAudit(userID, "UPDATE_APP", details)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)

	case http.MethodDelete:
		// Requires app:delete permission (admin only)
		if !rbac.CheckPermission(w, r, rbac.PermissionAppDelete) {
			return
		}

		// Get app name before deleting for audit log
		app, err := database.GetApp(id)
		if err != nil {
			log.Printf("Error getting app: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if app == nil {
			http.Error(w, "Application not found", http.StatusNotFound)
			return
		}

		if err := database.DeleteApp(id); err != nil {
			log.Printf("Error deleting app: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Log the action
		details := fmt.Sprintf("Deleted app: %s (%s)", app.Name, id)
		database.LogAudit(userID, "DELETE_APP", details)

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAuditLogs returns recent audit log entries
func handleAuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Requires audit:read permission (admin only)
	if !rbac.CheckPermission(w, r, rbac.PermissionAuditRead) {
		return
	}

	logs, err := database.GetAuditLogs(100)
	if err != nil {
		log.Printf("Error getting audit logs: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if logs == nil {
		logs = []db.AuditLog{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// handleAnalyticsLaunch records an app launch
func handleAnalyticsLaunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Requires analytics:write permission (all authenticated users)
	if !rbac.CheckPermission(w, r, rbac.PermissionAnalyticsWrite) {
		return
	}

	var req struct {
		AppID string `json:"app_id"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.AppID == "" {
		http.Error(w, "Missing required field: app_id", http.StatusBadRequest)
		return
	}

	if err := database.RecordLaunch(req.AppID); err != nil {
		log.Printf("Error recording launch: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "recorded"})
}

// handleAnalyticsStats returns analytics statistics
func handleAnalyticsStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Requires analytics:read permission (admin and app-author)
	if !rbac.CheckPermission(w, r, rbac.PermissionAnalyticsRead) {
		return
	}

	stats, err := database.GetAnalyticsStats()
	if err != nil {
		log.Printf("Error getting analytics stats: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// BrandingConfig represents tenant branding configuration
type BrandingConfig struct {
	LogoURL        string `json:"logo_url"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	TenantName     string `json:"tenant_name"`
}

// handleConfig returns tenant-specific branding configuration
func handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Requires config:read permission (all authenticated users)
	if !rbac.CheckPermission(w, r, rbac.PermissionConfigRead) {
		return
	}

	// Use centralized config for branding
	brandingCfg := BrandingConfig{
		LogoURL:        appConfig.LogoURL,
		PrimaryColor:   appConfig.PrimaryColor,
		SecondaryColor: appConfig.SecondaryColor,
		TenantName:     appConfig.TenantName,
	}

	// Try to load overrides from config file if it exists
	if data, err := os.ReadFile(appConfig.BrandingConfigPath); err == nil {
		json.Unmarshal(data, &brandingCfg)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(brandingCfg)
}

// handleSessions handles GET (list) and POST (create) for /api/sessions
func handleSessions(w http.ResponseWriter, r *http.Request) {
	// Get user from context for RBAC
	user := rbac.GetUserFromRequest(r)
	userID := "anonymous"
	if user != nil {
		userID = user.ID
	}

	switch r.Method {
	case http.MethodGet:
		// Requires session:read permission
		if !rbac.CheckPermission(w, r, rbac.PermissionSessionRead) {
			return
		}

		var sessionList []db.Session
		var err error

		// Admin can see all sessions, others only see their own
		if user != nil && user.HasPermission(rbac.PermissionSessionAll) {
			// Admin: optional filter by user_id or list all
			filterUserID := r.URL.Query().Get("user_id")
			if filterUserID != "" {
				sessionList, err = sessionManager.ListSessionsByUser(r.Context(), filterUserID)
			} else {
				sessionList, err = sessionManager.ListSessions(r.Context())
			}
		} else {
			// Non-admin: only their own sessions
			sessionList, err = sessionManager.ListSessionsByUser(r.Context(), userID)
		}

		if err != nil {
			log.Printf("Error listing sessions: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if sessionList == nil {
			sessionList = []db.Session{}
		}

		// Convert to response format with WebSocket URLs
		responses := make([]sessions.SessionResponse, len(sessionList))
		for i, s := range sessionList {
			app, _ := database.GetApp(s.AppID)
			appName := ""
			if app != nil {
				appName = app.Name
			}
			responses[i] = *sessions.SessionFromDB(&s, appName, sessionManager.GetSessionWebSocketURL(&s))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responses)

	case http.MethodPost:
		// Requires session:create permission
		if !rbac.CheckPermission(w, r, rbac.PermissionSessionCreate) {
			return
		}

		var req sessions.CreateSessionRequest
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.AppID == "" {
			http.Error(w, "Missing required field: app_id", http.StatusBadRequest)
			return
		}

		// Use authenticated user ID
		req.UserID = userID

		session, err := sessionManager.CreateSession(r.Context(), &req)
		if err != nil {
			log.Printf("Error creating session: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Get app name for response
		app, _ := database.GetApp(session.AppID)
		appName := ""
		if app != nil {
			appName = app.Name
		}

		response := sessions.SessionFromDB(session, appName, sessionManager.GetSessionWebSocketURL(session))

		// Log the action
		details := fmt.Sprintf("Created session %s for app %s", session.ID, session.AppID)
		database.LogAudit(userID, "CREATE_SESSION", details)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSessionByID handles GET and DELETE for /api/sessions/{id}
func handleSessionByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	id := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if id == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	// Get user from context for RBAC
	user := rbac.GetUserFromRequest(r)
	userID := "anonymous"
	if user != nil {
		userID = user.ID
	}

	switch r.Method {
	case http.MethodGet:
		// Requires session:read permission
		if !rbac.CheckPermission(w, r, rbac.PermissionSessionRead) {
			return
		}

		session, err := sessionManager.GetSession(r.Context(), id)
		if err != nil {
			log.Printf("Error getting session: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if session == nil {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}

		// Non-admin users can only view their own sessions
		if user != nil && !user.HasPermission(rbac.PermissionSessionAll) && session.UserID != userID {
			rbac.WriteError(w, http.StatusForbidden, "forbidden", "cannot access other users' sessions")
			return
		}

		// Get app name for response
		app, _ := database.GetApp(session.AppID)
		appName := ""
		if app != nil {
			appName = app.Name
		}

		response := sessions.SessionFromDB(session, appName, sessionManager.GetSessionWebSocketURL(session))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case http.MethodDelete:
		// Requires session:delete permission
		if !rbac.CheckPermission(w, r, rbac.PermissionSessionDelete) {
			return
		}

		session, err := sessionManager.GetSession(r.Context(), id)
		if err != nil {
			log.Printf("Error getting session: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if session == nil {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}

		// Non-admin users can only terminate their own sessions
		if user != nil && !user.HasPermission(rbac.PermissionSessionAll) && session.UserID != userID {
			rbac.WriteError(w, http.StatusForbidden, "forbidden", "cannot terminate other users' sessions")
			return
		}

		if err := sessionManager.TerminateSession(r.Context(), id); err != nil {
			log.Printf("Error terminating session: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Log the action
		details := fmt.Sprintf("Terminated session %s", id)
		database.LogAudit(userID, "TERMINATE_SESSION", details)

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleRBACRoles returns available roles and their permissions
func handleRBACRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Any authenticated user can see the role definitions
	user := rbac.GetUserFromRequest(r)
	if user == nil {
		rbac.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	type RoleInfo struct {
		Name        rbac.Role         `json:"name"`
		Permissions []rbac.Permission `json:"permissions"`
	}

	roles := []RoleInfo{
		{Name: rbac.RoleAdmin, Permissions: rbac.GetPermissions(rbac.RoleAdmin)},
		{Name: rbac.RoleAppAuthor, Permissions: rbac.GetPermissions(rbac.RoleAppAuthor)},
		{Name: rbac.RoleUser, Permissions: rbac.GetPermissions(rbac.RoleUser)},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roles)
}

// handleRBACUsers handles listing user roles (admin only)
func handleRBACUsers(w http.ResponseWriter, r *http.Request) {
	// Requires admin role
	user := rbac.GetUserFromRequest(r)
	if user == nil || !user.IsAdmin() {
		rbac.WriteError(w, http.StatusForbidden, "forbidden", "admin role required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		// List all user role assignments
		userRoles, err := rbacStore.ListUserRoles()
		if err != nil {
			log.Printf("Error listing user roles: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if userRoles == nil {
			userRoles = []rbac.UserRole{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userRoles)

	case http.MethodPost:
		// Create a new user role assignment
		var req struct {
			UserID string    `json:"user_id"`
			Role   rbac.Role `json:"role"`
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.UserID == "" {
			http.Error(w, "Missing required field: user_id", http.StatusBadRequest)
			return
		}

		if !rbac.IsValidRole(string(req.Role)) {
			http.Error(w, "Invalid role. Valid roles: admin, app-author, user", http.StatusBadRequest)
			return
		}

		if err := rbacStore.SetUserRole(req.UserID, req.Role); err != nil {
			log.Printf("Error setting user role: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Log the action
		details := fmt.Sprintf("Assigned role %s to user %s", req.Role, req.UserID)
		database.LogAudit(user.ID, "ASSIGN_ROLE", details)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_id": req.UserID,
			"role":    req.Role,
			"status":  "assigned",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleRBACUserByID handles GET, PUT, DELETE for /api/rbac/users/{user_id}
func handleRBACUserByID(w http.ResponseWriter, r *http.Request) {
	// Requires admin role
	user := rbac.GetUserFromRequest(r)
	if user == nil || !user.IsAdmin() {
		rbac.WriteError(w, http.StatusForbidden, "forbidden", "admin role required")
		return
	}

	// Extract user_id from path
	targetUserID := strings.TrimPrefix(r.URL.Path, "/api/rbac/users/")
	if targetUserID == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		role, err := rbacStore.GetUserRole(targetUserID)
		if err != nil {
			log.Printf("Error getting user role: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_id": targetUserID,
			"role":    role,
		})

	case http.MethodPut:
		var req struct {
			Role rbac.Role `json:"role"`
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if !rbac.IsValidRole(string(req.Role)) {
			http.Error(w, "Invalid role. Valid roles: admin, app-author, user", http.StatusBadRequest)
			return
		}

		if err := rbacStore.SetUserRole(targetUserID, req.Role); err != nil {
			log.Printf("Error updating user role: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Log the action
		details := fmt.Sprintf("Updated role for user %s to %s", targetUserID, req.Role)
		database.LogAudit(user.ID, "UPDATE_ROLE", details)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_id": targetUserID,
			"role":    req.Role,
			"status":  "updated",
		})

	case http.MethodDelete:
		if err := rbacStore.DeleteUserRole(targetUserID); err != nil {
			log.Printf("Error deleting user role: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Log the action
		details := fmt.Sprintf("Removed role assignment for user %s", targetUserID)
		database.LogAudit(user.ID, "REMOVE_ROLE", details)

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
