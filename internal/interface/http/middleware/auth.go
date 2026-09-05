package middleware

import (
	"net/http"
	"strings"

	"go-enterprise-api/internal/domain/audit"
	"go-enterprise-api/internal/domain/role"
	"go-enterprise-api/internal/domain/session"

	httpctx "go-enterprise-api/internal/interface/httpapi"
)

// AuthOptions wires the pieces needed to authenticate a request:
//
//   - Tokens:   JWT parse/validation
//   - Sessions: Redis-backed session store (a valid session must exist)
//   - Roles:    RBAC repository used to (re)load permission set if needed
//   - Audit:    best-effort login/logout trace (optional, may be nil)
type Authenticator struct {
	Tokens   session.TokenService
	Sessions session.Repository
	Roles    role.Repository
	Audit    audit.Logger
}

func (a *Authenticator) bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(h, "Bearer ")), true
}

// Authorize authenticates the bearer token and binds the session. It runs
// after rate limiting and before permission checks:
//
//	Request ID → Recovery → Rate Limit → Auth (Entra/JWT + Redis session)
//	                                      → Permission → Handler
func (a *Authenticator) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := a.bearerToken(r)
		if !ok {
			httpctx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "missing bearer token")
			return
		}

		claims, err := a.Tokens.Parse(r.Context(), token)
		if err != nil {
			httpctx.WriteError(w, http.StatusUnauthorized, "invalid_token", "token is invalid or expired")
			return
		}

		// Sessions live in Redis; a revoked/expired session denies access even
		// when the JWT is still cryptographically valid.
		if a.Sessions != nil {
			if _, err := a.Sessions.UserID(r.Context(), claims.SessionID); err != nil {
				httpctx.WriteError(w, http.StatusUnauthorized, "invalid_session", "session is invalid or expired")
				return
			}
		}

		ctx := httpctx.WithClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
