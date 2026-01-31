package models

import (
	"encoding/json"
	"time"
)

// User represents an authenticated user
type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	Roles       []string  `json:"roles"`
	IsAdmin     bool      `json:"isAdmin"`
	CreatedAt   time.Time `json:"createdAt"`
	LastLoginAt time.Time `json:"lastLoginAt"`
}

// RolesJSON returns the roles as a JSON string for database storage
func (u *User) RolesJSON() string {
	data, _ := json.Marshal(u.Roles)
	return string(data)
}

// ParseRoles parses a JSON string into roles slice
func ParseRoles(rolesJSON string) []string {
	var roles []string
	if rolesJSON == "" || rolesJSON == "null" {
		return []string{}
	}
	json.Unmarshal([]byte(rolesJSON), &roles)
	return roles
}

// Application represents a launchable application
type Application struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Icon        string   `json:"icon"`
	Category    string   `json:"category"`
	Roles       []string `json:"roles"`       // Empty = visible to all
	Enabled     bool     `json:"enabled"`     // Whether app is active
	SortOrder   int      `json:"sortOrder"`   // Display order
	IsFavorite  bool     `json:"isFavorite"`  // Populated per-user at runtime
}

// RolesJSON returns the roles as a JSON string for database storage
func (a *Application) RolesJSON() string {
	data, _ := json.Marshal(a.Roles)
	return string(data)
}

// Favorite represents a user's favorite application
type Favorite struct {
	UserID    string    `json:"userId"`
	AppID     string    `json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
}

// AppConfig is used for importing apps from JSON
type AppConfig struct {
	Applications []Application `json:"applications"`
}

// UserInfo contains minimal user info for API responses
type UserInfo struct {
	ID      string   `json:"id"`
	Email   string   `json:"email"`
	Name    string   `json:"name"`
	Roles   []string `json:"roles"`
	IsAdmin bool     `json:"isAdmin"`
}

// ToUserInfo converts a User to UserInfo
func (u *User) ToUserInfo() UserInfo {
	return UserInfo{
		ID:      u.ID,
		Email:   u.Email,
		Name:    u.Name,
		Roles:   u.Roles,
		IsAdmin: u.IsAdmin,
	}
}
