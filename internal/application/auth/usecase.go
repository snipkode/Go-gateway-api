package auth

import (
	"context"
	"time"

	"go-enterprise-api/internal/domain/session"
)

type LoginParams struct {
	Email     string
	Password  string
	IP        string
	UserAgent string
}

// EntraLoginParams carries a verified OIDC identity into the use case.
type EntraLoginParams struct {
	ProfileID string
	Email     string
	Name      string
	IP        string
	UserAgent string
}

type LoginResult struct {
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time      `json:"expires_at"`
	User         map[string]any `json:"user"`
}

type UseCase interface {
	Login(ctx context.Context, p LoginParams) (LoginResult, error)
	// EntraLogin signs in (and JIT-provisions) a verified Microsoft Entra identity.
	EntraLogin(ctx context.Context, p EntraLoginParams) (LoginResult, error)
	Logout(ctx context.Context, sessionID string) error
	Profile(ctx context.Context, c *session.Claims) (map[string]any, error)
}
