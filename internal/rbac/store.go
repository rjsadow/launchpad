package rbac

import (
	"database/sql"
	"fmt"
)

// DBStore implements UserStore using a SQL database
type DBStore struct {
	db          *sql.DB
	defaultRole Role
}

// NewDBStore creates a new database-backed user store
func NewDBStore(db *sql.DB) (*DBStore, error) {
	store := &DBStore{
		db:          db,
		defaultRole: RoleUser,
	}

	// Ensure the user_roles table exists
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate rbac tables: %w", err)
	}

	return store, nil
}

// migrate creates the user_roles table
func (s *DBStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS user_roles (
		user_id TEXT PRIMARY KEY,
		role TEXT NOT NULL DEFAULT 'user',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(role);
	`
	_, err := s.db.Exec(schema)
	return err
}

// GetUserRole returns the role for a user ID
// Returns the default role if user not found
func (s *DBStore) GetUserRole(userID string) (Role, error) {
	var roleStr string
	err := s.db.QueryRow(
		"SELECT role FROM user_roles WHERE user_id = ?",
		userID,
	).Scan(&roleStr)

	if err == sql.ErrNoRows {
		return s.defaultRole, nil
	}
	if err != nil {
		return s.defaultRole, err
	}

	role := Role(roleStr)
	if !IsValidRole(string(role)) {
		return s.defaultRole, nil
	}

	return role, nil
}

// SetUserRole sets or updates the role for a user
func (s *DBStore) SetUserRole(userID string, role Role) error {
	if !IsValidRole(string(role)) {
		return fmt.Errorf("invalid role: %s", role)
	}

	_, err := s.db.Exec(`
		INSERT INTO user_roles (user_id, role, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			role = excluded.role,
			updated_at = CURRENT_TIMESTAMP
	`, userID, string(role))

	return err
}

// DeleteUserRole removes a user's role assignment
func (s *DBStore) DeleteUserRole(userID string) error {
	_, err := s.db.Exec("DELETE FROM user_roles WHERE user_id = ?", userID)
	return err
}

// ListUserRoles returns all user role assignments
func (s *DBStore) ListUserRoles() ([]UserRole, error) {
	rows, err := s.db.Query(
		"SELECT user_id, role, created_at, updated_at FROM user_roles ORDER BY user_id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []UserRole
	for rows.Next() {
		var ur UserRole
		var roleStr string
		if err := rows.Scan(&ur.UserID, &roleStr, &ur.CreatedAt, &ur.UpdatedAt); err != nil {
			return nil, err
		}
		ur.Role = Role(roleStr)
		roles = append(roles, ur)
	}

	return roles, rows.Err()
}

// ListUsersByRole returns all users with a specific role
func (s *DBStore) ListUsersByRole(role Role) ([]string, error) {
	rows, err := s.db.Query(
		"SELECT user_id FROM user_roles WHERE role = ? ORDER BY user_id",
		string(role),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, rows.Err()
}

// CountUsersByRole returns the count of users with each role
func (s *DBStore) CountUsersByRole() (map[Role]int, error) {
	rows, err := s.db.Query(
		"SELECT role, COUNT(*) FROM user_roles GROUP BY role",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[Role]int)
	for rows.Next() {
		var roleStr string
		var count int
		if err := rows.Scan(&roleStr, &count); err != nil {
			return nil, err
		}
		counts[Role(roleStr)] = count
	}

	return counts, rows.Err()
}

// UserRole represents a user-role assignment
type UserRole struct {
	UserID    string `json:"user_id"`
	Role      Role   `json:"role"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// InMemoryStore implements UserStore using an in-memory map
// Useful for testing
type InMemoryStore struct {
	roles       map[string]Role
	defaultRole Role
}

// NewInMemoryStore creates a new in-memory user store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		roles:       make(map[string]Role),
		defaultRole: RoleUser,
	}
}

// GetUserRole returns the role for a user ID
func (s *InMemoryStore) GetUserRole(userID string) (Role, error) {
	role, ok := s.roles[userID]
	if !ok {
		return s.defaultRole, nil
	}
	return role, nil
}

// SetUserRole sets the role for a user
func (s *InMemoryStore) SetUserRole(userID string, role Role) {
	s.roles[userID] = role
}

// SetDefaultRole sets the default role for unknown users
func (s *InMemoryStore) SetDefaultRole(role Role) {
	s.defaultRole = role
}
