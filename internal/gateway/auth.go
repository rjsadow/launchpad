package gateway

import (
	"net/http"
	"strings"
)

// NoopAuthenticator allows all requests (for development/testing)
type NoopAuthenticator struct{}

// NewNoopAuthenticator creates a new NoopAuthenticator
func NewNoopAuthenticator() *NoopAuthenticator {
	return &NoopAuthenticator{}
}

// Authenticate always returns "anonymous" user
func (a *NoopAuthenticator) Authenticate(r *http.Request) (string, error) {
	return "anonymous", nil
}

// HeaderAuthenticator extracts user ID from a request header
type HeaderAuthenticator struct {
	headerName string
	required   bool
}

// NewHeaderAuthenticator creates a new HeaderAuthenticator
func NewHeaderAuthenticator(headerName string, required bool) *HeaderAuthenticator {
	return &HeaderAuthenticator{
		headerName: headerName,
		required:   required,
	}
}

// Authenticate extracts user ID from the configured header
func (a *HeaderAuthenticator) Authenticate(r *http.Request) (string, error) {
	userID := strings.TrimSpace(r.Header.Get(a.headerName))
	if userID == "" {
		if a.required {
			return "", ErrUnauthorized
		}
		return "anonymous", nil
	}
	return userID, nil
}

// TokenAuthenticator validates bearer tokens
type TokenAuthenticator struct {
	validateFunc func(token string) (userID string, err error)
}

// NewTokenAuthenticator creates a new TokenAuthenticator with a validation function
func NewTokenAuthenticator(validateFunc func(token string) (string, error)) *TokenAuthenticator {
	return &TokenAuthenticator{
		validateFunc: validateFunc,
	}
}

// Authenticate validates the bearer token and returns the user ID
func (a *TokenAuthenticator) Authenticate(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", ErrUnauthorized
	}

	// Extract bearer token
	const prefix = "Bearer "
	if len(authHeader) < len(prefix) || authHeader[:len(prefix)] != prefix {
		return "", ErrUnauthorized
	}

	token := strings.TrimSpace(authHeader[len(prefix):])
	if token == "" {
		return "", ErrUnauthorized
	}

	return a.validateFunc(token)
}

// ChainAuthenticator tries multiple authenticators in order
type ChainAuthenticator struct {
	authenticators []Authenticator
}

// NewChainAuthenticator creates a new ChainAuthenticator
func NewChainAuthenticator(authenticators ...Authenticator) *ChainAuthenticator {
	return &ChainAuthenticator{
		authenticators: authenticators,
	}
}

// Authenticate tries each authenticator until one succeeds
func (a *ChainAuthenticator) Authenticate(r *http.Request) (string, error) {
	var lastErr error
	for _, auth := range a.authenticators {
		userID, err := auth.Authenticate(r)
		if err == nil {
			return userID, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", ErrUnauthorized
}
