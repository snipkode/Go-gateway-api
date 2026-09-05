package entra

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

var (
	ErrNotConfigured  = errors.New("entra oidc provider is not configured")
	ErrExchangeFailed = errors.New("failed to exchange authorization code")
	ErrEmailMissing   = errors.New("entra profile has no verified email")
)

// Config mirrors the environment-based OAuth settings. Keeping this as a tiny
// struct (rather than importing internal/config) keeps infrastructure
// decoupled from application config wiring.
type Config struct {
	ClientID     string
	ClientSecret string
	TenantID     string
	Issuer       string
	RedirectURL  string
	Timeout      time.Duration
}

// Profile is the normalized identity returned from the OIDC exchange. Entra
// guarantees email + unique_name are present when `openid profile email` are
// requested; `sub` is the stable external id.
type Profile struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	EmailVerified bool   `json:"email_verified"`
}

// Provider is an abstraction over Microsoft Entra (OIDC) so the HTTP layer
// depends on a small contract instead of on go-oidc/oauth2 directly.
type Provider interface {
	// AuthURL returns the browser redirect URL for the given CSRF state.
	AuthURL(ctx context.Context, state string) (string, error)
	// Exchange trades an authorization code for a verified user profile.
	Exchange(ctx context.Context, code string) (Profile, error)
}

// OIDCProvider is the production adapter built on top of
// https://github.com/coreos/go-oidc with the Entra v2.0 discovery endpoint:
//
//	https://login.microsoftonline.com/{tenant}/v2.0/.well-known/openid-configuration
type OIDCProvider struct {
	cfg    Config
	issuer string
	oauth  oauth2.Config
	driver *oidc.Provider
	token  *oidc.IDTokenVerifier
}

// Issuer returns the OIDC issuer the provider talks to (for logging/demo).
func (p *OIDCProvider) Issuer() string { return p.issuer }

// NewProvider runs OIDC discovery for the issuer and prepares the OAuth2
// client. It does not call home until a login actually happens, but building
// the verifier fetches the discovery document so it is best done at startup.
// Returns ErrNotConfigured when the client id / issuer are missing so the
// caller can leave the Entra flow disabled instead of failing the boot.
func NewProvider(ctx context.Context, cfg Config) (*OIDCProvider, error) {
	issuer := cfg.Issuer
	if issuer == "" {
		if cfg.TenantID == "" {
			return nil, ErrNotConfigured
		}
		issuer = fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", cfg.TenantID)
	}
	if cfg.ClientID == "" {
		return nil, ErrNotConfigured
	}

	driver, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}

	oauth := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     driver.Endpoint(),
		RedirectURL:  cfg.RedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeOfflineAccess, "profile", "email"},
	}
	verifier := driver.Verifier(&oidc.Config{ClientID: cfg.ClientID})

	return &OIDCProvider{
		cfg:    cfg,
		issuer: issuer,
		oauth:  oauth,
		driver: driver,
		token:  verifier,
	}, nil
}

// AuthURL returns the authorization endpoint URL the browser must be sent to.
func (p *OIDCProvider) AuthURL(_ context.Context, state string) (string, error) {
	if p == nil {
		return "", ErrNotConfigured
	}
	return p.oauth.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// Exchange verifies the authorization `code` with Azure AD, validates the ID
// token signature/audience and extracts the profile.
func (p *OIDCProvider) Exchange(ctx context.Context, code string) (Profile, error) {
	if p == nil {
		return Profile{}, ErrNotConfigured
	}

	ctx, cancel := context.WithTimeout(ctx, p.cfg.Timeout)
	defer cancel()

	tok, err := p.oauth.Exchange(ctx, code)
	if err != nil {
		return Profile{}, fmt.Errorf("%w: %v", ErrExchangeFailed, err)
	}

	raw, ok := tok.Extra("id_token").(string)
	if !ok || raw == "" {
		return Profile{}, fmt.Errorf("%w: missing id_token", ErrExchangeFailed)
	}

	idToken, err := p.token.Verify(ctx, raw)
	if err != nil {
		return Profile{}, fmt.Errorf("%w: id_token verification: %v", ErrExchangeFailed, err)
	}

	var claims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		PreferredName string `json:"preferred_username"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return Profile{}, fmt.Errorf("%w: claims: %v", ErrExchangeFailed, err)
	}

	name := claims.Name
	if name == "" {
		name = claims.PreferredName
	}
	if claims.Email == "" {
		return Profile{}, ErrEmailMissing
	}
	return Profile{
		ID:            claims.Sub,
		Email:         claims.Email,
		Name:          name,
		EmailVerified: claims.EmailVerified,
	}, nil
}

var _ Provider = (*OIDCProvider)(nil)
