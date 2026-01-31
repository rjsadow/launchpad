// Package auth provides authentication and authorization functionality.
package auth

import (
	"golang.org/x/crypto/bcrypt"
)

const (
	// bcryptCost is the cost parameter for bcrypt hashing.
	// A higher cost means more secure but slower hashing.
	bcryptCost = 12
)

// HashPassword hashes a plaintext password using bcrypt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compares a plaintext password with a bcrypt hash.
// Returns nil if they match, or an error if they don't.
func CheckPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
