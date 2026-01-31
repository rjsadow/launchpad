package store

import (
	"database/sql"
	"time"

	"github.com/rjsadow/launchpad/internal/models"
)

// GetUser retrieves a user by ID
func (db *DB) GetUser(id string) (*models.User, error) {
	var user models.User
	var rolesJSON string
	var isAdmin int
	var lastLogin sql.NullTime

	err := db.QueryRow(`
		SELECT id, email, name, roles, is_admin, created_at, last_login_at
		FROM users WHERE id = ?
	`, id).Scan(&user.ID, &user.Email, &user.Name, &rolesJSON, &isAdmin, &user.CreatedAt, &lastLogin)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	user.Roles = models.ParseRoles(rolesJSON)
	user.IsAdmin = isAdmin == 1
	if lastLogin.Valid {
		user.LastLoginAt = lastLogin.Time
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (db *DB) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	var rolesJSON string
	var isAdmin int
	var lastLogin sql.NullTime

	err := db.QueryRow(`
		SELECT id, email, name, roles, is_admin, created_at, last_login_at
		FROM users WHERE email = ?
	`, email).Scan(&user.ID, &user.Email, &user.Name, &rolesJSON, &isAdmin, &user.CreatedAt, &lastLogin)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	user.Roles = models.ParseRoles(rolesJSON)
	user.IsAdmin = isAdmin == 1
	if lastLogin.Valid {
		user.LastLoginAt = lastLogin.Time
	}

	return &user, nil
}

// UpsertUser creates or updates a user (called on login)
func (db *DB) UpsertUser(user *models.User) error {
	isAdmin := 0
	if user.IsAdmin {
		isAdmin = 1
	}

	_, err := db.Exec(`
		INSERT INTO users (id, email, name, roles, is_admin, created_at, last_login_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			email = excluded.email,
			name = excluded.name,
			roles = excluded.roles,
			is_admin = excluded.is_admin,
			last_login_at = CURRENT_TIMESTAMP
	`, user.ID, user.Email, user.Name, user.RolesJSON(), isAdmin)

	return err
}

// UpdateUserLastLogin updates the last login timestamp
func (db *DB) UpdateUserLastLogin(id string) error {
	_, err := db.Exec(`UPDATE users SET last_login_at = ? WHERE id = ?`, time.Now(), id)
	return err
}
