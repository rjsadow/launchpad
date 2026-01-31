package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/rjsadow/launchpad/internal/config"
	"github.com/rjsadow/launchpad/internal/models"
	"golang.org/x/oauth2"
)

// OIDCProvider handles OIDC authentication
type OIDCProvider struct {
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config oauth2.Config
	rolesClaim   string
	adminRole    string
}

// NewOIDCProvider creates a new OIDC provider from config
func NewOIDCProvider(ctx context.Context, cfg *config.Config) (*OIDCProvider, error) {
	if !cfg.AuthEnabled() {
		return nil, nil
	}

	provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuer)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	return &OIDCProvider{
		provider: provider,
		verifier: provider.Verifier(&oidc.Config{ClientID: cfg.OIDCClientID}),
		oauth2Config: oauth2.Config{
			ClientID:     cfg.OIDCClientID,
			ClientSecret: cfg.OIDCClientSecret,
			RedirectURL:  cfg.OIDCRedirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		rolesClaim: cfg.OIDCRolesClaim,
		adminRole:  cfg.OIDCAdminRole,
	}, nil
}

// GetAuthURL returns the OAuth2 authorization URL
func (p *OIDCProvider) GetAuthURL(state string) string {
	return p.oauth2Config.AuthCodeURL(state)
}

// GenerateState generates a random state parameter for CSRF protection
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ExchangeCode exchanges the authorization code for tokens and extracts user info
func (p *OIDCProvider) ExchangeCode(ctx context.Context, code string) (*models.User, error) {
	token, err := p.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no id_token in response")
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Extract claims
	var claims struct {
		Subject string                 `json:"sub"`
		Email   string                 `json:"email"`
		Name    string                 `json:"name"`
		Extra   map[string]interface{} `json:"-"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	// Extract all claims for role lookup
	var allClaims map[string]interface{}
	if err := idToken.Claims(&allClaims); err != nil {
		return nil, fmt.Errorf("failed to parse all claims: %w", err)
	}

	// Extract roles from configured claim
	roles := p.extractRoles(allClaims)

	// Check if user has admin role
	isAdmin := false
	for _, role := range roles {
		if role == p.adminRole {
			isAdmin = true
			break
		}
	}

	return &models.User{
		ID:      claims.Subject,
		Email:   claims.Email,
		Name:    claims.Name,
		Roles:   roles,
		IsAdmin: isAdmin,
	}, nil
}

// extractRoles extracts roles from the claims based on the configured claim name
func (p *OIDCProvider) extractRoles(claims map[string]interface{}) []string {
	var roles []string

	rolesClaim := claims[p.rolesClaim]
	if rolesClaim == nil {
		return roles
	}

	switch v := rolesClaim.(type) {
	case []interface{}:
		for _, r := range v {
			if s, ok := r.(string); ok {
				roles = append(roles, s)
			}
		}
	case []string:
		roles = v
	case string:
		roles = []string{v}
	}

	return roles
}

// stateCookie manages the state parameter cookie
const stateCookieName = "launchpad_oauth_state"

// SetStateCookie sets the state cookie for CSRF protection
func SetStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300, // 5 minutes
	})
}

// GetStateCookie retrieves and clears the state cookie
func GetStateCookie(w http.ResponseWriter, r *http.Request) (string, error) {
	cookie, err := r.Cookie(stateCookieName)
	if err != nil {
		return "", err
	}

	// Clear the cookie
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	return cookie.Value, nil
}
