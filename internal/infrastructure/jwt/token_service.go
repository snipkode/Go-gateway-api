package jwt

import (
	"context"
	"errors"
	"time"

	"go-enterprise-api/internal/domain/session"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type TokenService struct {
	secret []byte
}

func NewTokenService(secret string) *TokenService {
	return &TokenService{secret: []byte(secret)}
}

type tokenClaims struct {
	SessionID string   `json:"session_id"`
	UserID    int64    `json:"user_id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Roles     []string `json:"roles"`
	jwt.RegisteredClaims
}

func (s *TokenService) Issue(ctx context.Context, c session.Claims, ttl time.Duration) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(ttl)
	tc := tokenClaims{
		SessionID: c.SessionID,
		UserID:    c.UserID,
		Email:     c.Email,
		Name:      c.Name,
		Roles:     c.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tc)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

func (s *TokenService) Parse(ctx context.Context, tokenString string) (*session.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &tokenClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	tc, ok := token.Claims.(*tokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return &session.Claims{
		SessionID: tc.SessionID,
		UserID:    tc.UserID,
		Email:     tc.Email,
		Name:      tc.Name,
		Roles:     tc.Roles,
	}, nil
}

var _ session.TokenService = (*TokenService)(nil)
