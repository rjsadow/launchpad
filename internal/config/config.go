package config

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	Port          int
	DBPath        string
	SessionSecret []byte

	// OIDC settings
	OIDCIssuer      string
	OIDCClientID    string
	OIDCClientSecret string
	OIDCRedirectURL string
	OIDCRolesClaim  string
	OIDCAdminRole   string
}

// Load reads configuration from environment variables with sensible defaults
func Load() *Config {
	cfg := &Config{
		Port:            getEnvInt("LAUNCHPAD_PORT", 8080),
		DBPath:          getEnv("LAUNCHPAD_DB_PATH", "./launchpad.db"),
		OIDCIssuer:      getEnv("LAUNCHPAD_OIDC_ISSUER", ""),
		OIDCClientID:    getEnv("LAUNCHPAD_OIDC_CLIENT_ID", ""),
		OIDCClientSecret: getEnv("LAUNCHPAD_OIDC_CLIENT_SECRET", ""),
		OIDCRedirectURL: getEnv("LAUNCHPAD_OIDC_REDIRECT_URL", ""),
		OIDCRolesClaim:  getEnv("LAUNCHPAD_OIDC_ROLES_CLAIM", "groups"),
		OIDCAdminRole:   getEnv("LAUNCHPAD_OIDC_ADMIN_ROLE", "launchpad-admin"),
	}

	// Session secret
	secretStr := getEnv("LAUNCHPAD_SESSION_SECRET", "")
	if secretStr != "" {
		cfg.SessionSecret = []byte(secretStr)
	} else {
		// Generate random secret for development (not persistent across restarts)
		cfg.SessionSecret = generateRandomSecret()
		log.Println("Warning: LAUNCHPAD_SESSION_SECRET not set, using random secret (sessions won't persist across restarts)")
	}

	return cfg
}

// AuthEnabled returns true if OIDC is configured
func (c *Config) AuthEnabled() bool {
	return c.OIDCIssuer != "" && c.OIDCClientID != "" && c.OIDCClientSecret != ""
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func generateRandomSecret() []byte {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a deterministic but unique secret
		return []byte("launchpad-dev-secret-change-me!")
	}
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(b)))
	base64.StdEncoding.Encode(encoded, b)
	return encoded[:32]
}
