package session

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("session not found")

type Claims struct {
	SessionID string   `json:"session_id"`
	UserID    int64    `json:"user_id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Roles     []string `json:"roles"`
	Expiry    int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
}

type Session struct {
	SessionID     string
	UserID        int64
	AccessToken   string
	RefreshToken  string
	IDTokenClaims *Claims
	ExpiresAt     time.Time
	UserAgent     string
	IPAddress     string
}

type Repository interface {
	Create(ctx context.Context, s Session) error
	Get(ctx context.Context, sessionID string) (Session, error)
	UserID(ctx context.Context, sessionID string) (int64, error)
	Delete(ctx context.Context, sessionID string) error
	DeleteByUser(ctx context.Context, userID int64) error
	Touch(ctx context.Context, sessionID string) error
}

// TokenService issues and validates JWTs.
type TokenService interface {
	Issue(ctx context.Context, c Claims, ttl time.Duration) (string, time.Time, error)
	Parse(ctx context.Context, token string) (*Claims, error)
}
