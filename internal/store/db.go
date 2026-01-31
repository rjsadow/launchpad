package store

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"

	"github.com/rjsadow/launchpad/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the database connection
type DB struct {
	*sql.DB
}

// New creates a new database connection and runs migrations
func New(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, err
	}

	store := &DB{db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// migrate runs database migrations
func (db *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			name TEXT,
			roles TEXT DEFAULT '[]',
			is_admin INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_login_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS applications (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			url TEXT NOT NULL,
			icon TEXT,
			category TEXT NOT NULL,
			roles TEXT DEFAULT '[]',
			enabled INTEGER DEFAULT 1,
			sort_order INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS favorites (
			user_id TEXT NOT NULL,
			app_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, app_id),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (app_id) REFERENCES applications(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_user ON favorites(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_applications_category ON applications(category)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return err
		}
	}

	return nil
}

// SeedFromJSON imports applications from a JSON file if the database is empty
func (db *DB) SeedFromJSON(jsonPath string) error {
	// Check if apps already exist
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM applications").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		log.Printf("Database already has %d applications, skipping seed", count)
		return nil
	}

	// Read JSON file
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("No seed file found at", jsonPath)
			return nil
		}
		return err
	}

	var config models.AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Import applications
	for i, app := range config.Applications {
		if app.Roles == nil {
			app.Roles = []string{}
		}
		rolesJSON, _ := json.Marshal(app.Roles)

		_, err := db.Exec(`
			INSERT INTO applications (id, name, description, url, icon, category, roles, enabled, sort_order)
			VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?)
		`, app.ID, app.Name, app.Description, app.URL, app.Icon, app.Category, string(rolesJSON), i)
		if err != nil {
			return err
		}
	}

	log.Printf("Seeded %d applications from %s", len(config.Applications), jsonPath)
	return nil
}
