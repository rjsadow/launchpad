package store

import (
	"database/sql"
	"encoding/json"

	"github.com/rjsadow/launchpad/internal/models"
)

// GetApps retrieves all enabled applications
func (db *DB) GetApps() ([]models.Application, error) {
	return db.getApps("SELECT id, name, description, url, icon, category, roles, enabled, sort_order FROM applications WHERE enabled = 1 ORDER BY sort_order, name")
}

// GetAllApps retrieves all applications (including disabled) for admin
func (db *DB) GetAllApps() ([]models.Application, error) {
	return db.getApps("SELECT id, name, description, url, icon, category, roles, enabled, sort_order FROM applications ORDER BY sort_order, name")
}

func (db *DB) getApps(query string) ([]models.Application, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []models.Application
	for rows.Next() {
		var app models.Application
		var rolesJSON string
		var enabled int

		if err := rows.Scan(&app.ID, &app.Name, &app.Description, &app.URL, &app.Icon, &app.Category, &rolesJSON, &enabled, &app.SortOrder); err != nil {
			return nil, err
		}

		app.Roles = models.ParseRoles(rolesJSON)
		app.Enabled = enabled == 1
		apps = append(apps, app)
	}

	return apps, rows.Err()
}

// GetApp retrieves a single application by ID
func (db *DB) GetApp(id string) (*models.Application, error) {
	var app models.Application
	var rolesJSON string
	var enabled int

	err := db.QueryRow(`
		SELECT id, name, description, url, icon, category, roles, enabled, sort_order
		FROM applications WHERE id = ?
	`, id).Scan(&app.ID, &app.Name, &app.Description, &app.URL, &app.Icon, &app.Category, &rolesJSON, &enabled, &app.SortOrder)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	app.Roles = models.ParseRoles(rolesJSON)
	app.Enabled = enabled == 1
	return &app, nil
}

// CreateApp creates a new application
func (db *DB) CreateApp(app *models.Application) error {
	if app.Roles == nil {
		app.Roles = []string{}
	}
	rolesJSON, _ := json.Marshal(app.Roles)

	enabled := 0
	if app.Enabled {
		enabled = 1
	}

	_, err := db.Exec(`
		INSERT INTO applications (id, name, description, url, icon, category, roles, enabled, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, app.ID, app.Name, app.Description, app.URL, app.Icon, app.Category, string(rolesJSON), enabled, app.SortOrder)

	return err
}

// UpdateApp updates an existing application
func (db *DB) UpdateApp(app *models.Application) error {
	if app.Roles == nil {
		app.Roles = []string{}
	}
	rolesJSON, _ := json.Marshal(app.Roles)

	enabled := 0
	if app.Enabled {
		enabled = 1
	}

	result, err := db.Exec(`
		UPDATE applications SET
			name = ?, description = ?, url = ?, icon = ?, category = ?,
			roles = ?, enabled = ?, sort_order = ?
		WHERE id = ?
	`, app.Name, app.Description, app.URL, app.Icon, app.Category, string(rolesJSON), enabled, app.SortOrder, app.ID)

	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteApp deletes an application
func (db *DB) DeleteApp(id string) error {
	result, err := db.Exec("DELETE FROM applications WHERE id = ?", id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// FilterAppsByRoles filters apps to only those visible to a user with given roles
func FilterAppsByRoles(apps []models.Application, userRoles []string, isAdmin bool) []models.Application {
	if isAdmin {
		return apps // Admin sees all
	}

	roleSet := make(map[string]bool)
	for _, r := range userRoles {
		roleSet[r] = true
	}

	var filtered []models.Application
	for _, app := range apps {
		if len(app.Roles) == 0 {
			// No role restrictions, visible to all
			filtered = append(filtered, app)
			continue
		}

		// Check if user has any required role
		for _, appRole := range app.Roles {
			if roleSet[appRole] {
				filtered = append(filtered, app)
				break
			}
		}
	}

	return filtered
}
