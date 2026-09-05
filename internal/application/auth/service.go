package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"go-enterprise-api/internal/domain/audit"
	"go-enterprise-api/internal/domain/role"
	"go-enterprise-api/internal/domain/session"
	"go-enterprise-api/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrSessionNotFound     = errors.New("session not found")
	ErrSSOAccountConflict  = errors.New("sso login conflicts with an existing local account")
	ErrSSOIdentityConflict = errors.New("sso account is already linked to a different identity")
	ErrSSODefaultRole      = errors.New("sso default role not found")
)

type Service struct {
	Users           user.Repository
	Roles           role.Repository
	Sessions        session.Repository
	Tokens          session.TokenService
	Audit           audit.Logger
	JWTTTL          time.Duration
	DefaultRoleSlug string
}

func NewService(
	users user.Repository,
	roles role.Repository,
	sessions session.Repository,
	tokens session.TokenService,
	audit audit.Logger,
	jwtTTL time.Duration,
	defaultRoleSlug string,
) *Service {
	return &Service{
		Users:           users,
		Roles:           roles,
		Sessions:        sessions,
		Tokens:          tokens,
		Audit:           audit,
		JWTTTL:          jwtTTL,
		DefaultRoleSlug: defaultRoleSlug,
	}
}

func (s *Service) Login(ctx context.Context, p LoginParams) (LoginResult, error) {
	u, err := s.Users.GetByEmail(ctx, p.Email)
	if err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}
	if !u.IsActive() {
		return LoginResult{}, ErrInvalidCredentials
	}
	if u.PasswordHash == "" {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(p.Password)); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	return s.issueSession(ctx, &u, p.IP, p.UserAgent, audit.ActionLogin)
}

// EntraLogin authenticates an OIDC identity. On first sign-in it provisions
// the user (JIT) with the configured default role; afterwards the user is
// recognized by email and reused. A local account (provider='local') with the
// same email is rejected rather than silently taken over — an admin must link
// the SSO identity on the local account first.
func (s *Service) EntraLogin(ctx context.Context, p EntraLoginParams) (LoginResult, error) {
	u, err := s.Users.GetByEmail(ctx, p.Email)
	switch {
	case err == nil:
		if u.Provider == "local" || u.Provider == "" {
			return LoginResult{}, ErrSSOAccountConflict
		}
		if u.ProviderID != "" && u.ProviderID != p.ProfileID {
			return LoginResult{}, ErrSSOIdentityConflict
		}
		if !u.IsActive() {
			return LoginResult{}, ErrInvalidCredentials
		}
	case errors.Is(err, user.ErrNotFound):
		created, err := s.Users.Create(ctx, user.CreateParams{
			Email:        p.Email,
			Name:         p.Name,
			PasswordHash: "",
			Provider:     "entra",
			ProviderID:   p.ProfileID,
		})
		if err != nil {
			return LoginResult{}, err
		}
		u = created

		rl, err := s.Roles.GetBySlug(ctx, s.DefaultRoleSlug)
		if err != nil {
			if errors.Is(err, role.ErrNotFound) {
				return LoginResult{}, ErrSSODefaultRole
			}
			return LoginResult{}, err
		}
		if err := s.Roles.AssignRoleToUser(ctx, u.ID, rl.ID); err != nil {
			return LoginResult{}, err
		}
	default:
		return LoginResult{}, err
	}

	return s.issueSession(ctx, &u, p.IP, p.UserAgent, audit.ActionLogin)
}

func (s *Service) issueSession(ctx context.Context, u *user.User, ip, userAgent, action string) (LoginResult, error) {
	roles, err := s.Roles.RoleSlugsByUser(ctx, u.ID)
	if err != nil {
		return LoginResult{}, err
	}

	sessionID := "sess-" + randomID()
	claims := session.Claims{
		SessionID: sessionID,
		UserID:    u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Roles:     roles,
	}
	accessToken, expiresAt, err := s.Tokens.Issue(ctx, claims, s.JWTTTL)
	if err != nil {
		return LoginResult{}, err
	}

	sess := session.Session{
		SessionID:     sessionID,
		UserID:        u.ID,
		AccessToken:   accessToken,
		IDTokenClaims: &claims,
		ExpiresAt:     expiresAt,
		UserAgent:     userAgent,
		IPAddress:     ip,
	}
	if err := s.Sessions.Create(ctx, sess); err != nil {
		return LoginResult{}, err
	}

	_ = s.Audit.Log(ctx, audit.Entry{
		UserID:    u.ID,
		Action:    action,
		Resource:  "auth",
		IPAddress: ip,
		UserAgent: userAgent,
	})

	return LoginResult{
		AccessToken: accessToken,
		ExpiresAt:   expiresAt,
		User: map[string]any{
			"id":    u.ID,
			"email": u.Email,
			"name":  u.Name,
			"roles": roles,
		},
	}, nil
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	uID, err := s.Sessions.UserID(ctx, sessionID)
	if err != nil {
		return ErrSessionNotFound
	}
	if err := s.Sessions.Delete(ctx, sessionID); err != nil {
		return err
	}
	_ = s.Audit.Log(ctx, audit.Entry{
		UserID:   uID,
		Action:   audit.ActionLogout,
		Resource: "auth",
	})
	return nil
}

func (s *Service) Profile(ctx context.Context, c *session.Claims) (map[string]any, error) {
	u, err := s.Users.GetByID(ctx, c.UserID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":    u.ID,
		"email": u.Email,
		"name":  u.Name,
		"roles": c.Roles,
	}, nil
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
