package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rjsadow/launchpad/internal/auth"
	"github.com/rjsadow/launchpad/internal/config"
	"github.com/rjsadow/launchpad/internal/handlers"
	"github.com/rjsadow/launchpad/internal/store"
)

//go:embed web/dist/*
var embeddedFiles embed.FS

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := store.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Seed database from apps.json if empty
	seedPath := filepath.Join("web", "dist", "apps.json")
	if err := db.SeedFromJSON(seedPath); err != nil {
		log.Printf("Warning: Failed to seed database: %v", err)
	}

	// Initialize session manager
	sessions := auth.NewSessionManager(cfg.SessionSecret)

	// Initialize OIDC provider (may be nil if not configured)
	var oidcProvider *auth.OIDCProvider
	if cfg.AuthEnabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		oidcProvider, err = auth.NewOIDCProvider(ctx, cfg)
		cancel()
		if err != nil {
			log.Fatalf("Failed to initialize OIDC provider: %v", err)
		}
		log.Printf("OIDC authentication enabled with issuer: %s", cfg.OIDCIssuer)
	} else {
		log.Println("OIDC authentication disabled (running in development mode)")
	}

	// Initialize middleware
	authMiddleware := auth.NewMiddleware(sessions, cfg.AuthEnabled())

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(oidcProvider, sessions, db)
	appsHandler := handlers.NewAppsHandler(db)
	favoritesHandler := handlers.NewFavoritesHandler(db)
	adminHandler := handlers.NewAdminHandler(db)

	// Create router
	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("GET /auth/login", authHandler.HandleLogin)
	mux.HandleFunc("GET /auth/callback", authHandler.HandleCallback)
	mux.HandleFunc("POST /auth/logout", authHandler.HandleLogout)

	// API routes (require authentication)
	mux.Handle("GET /api/me", authMiddleware.RequireAuth(http.HandlerFunc(appsHandler.HandleGetMe)))
	mux.Handle("GET /api/apps", authMiddleware.OptionalAuth(http.HandlerFunc(appsHandler.HandleGetApps)))

	// Favorites routes (require authentication)
	mux.Handle("GET /api/favorites", authMiddleware.RequireAuth(http.HandlerFunc(favoritesHandler.HandleGetFavorites)))
	mux.Handle("POST /api/favorites/{appId}", authMiddleware.RequireAuth(http.HandlerFunc(favoritesHandler.HandleAddFavorite)))
	mux.Handle("DELETE /api/favorites/{appId}", authMiddleware.RequireAuth(http.HandlerFunc(favoritesHandler.HandleRemoveFavorite)))

	// Admin routes (require admin role)
	adminAuth := func(h http.Handler) http.Handler {
		return authMiddleware.RequireAuth(authMiddleware.RequireAdmin(h))
	}
	mux.Handle("GET /api/admin/apps", adminAuth(http.HandlerFunc(adminHandler.HandleGetAllApps)))
	mux.Handle("POST /api/apps", adminAuth(http.HandlerFunc(adminHandler.HandleCreateApp)))
	mux.Handle("PUT /api/apps/{id}", adminAuth(http.HandlerFunc(adminHandler.HandleUpdateApp)))
	mux.Handle("DELETE /api/apps/{id}", adminAuth(http.HandlerFunc(adminHandler.HandleDeleteApp)))

	// Serve static files and SPA
	distFS, err := fs.Sub(embeddedFiles, "web/dist")
	if err != nil {
		log.Fatal("Failed to access embedded files:", err)
	}
	fileServer := http.FileServer(http.FS(distFS))

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

	// Create server
	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Launchpad server starting on http://localhost%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
