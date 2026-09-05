package middleware

import (
	"context"
	"net/http"

	"go-enterprise-api/internal/domain/role"

	httpctx "go-enterprise-api/internal/interface/httpapi"
)

// PermissionAuthorizer enforces RBAC. Permissions are dynamic: they live in
// PostgreSQL and are resolved per user from their roles on each request
// (cached per-request in context). Requires Authorize to have run first.
type PermissionAuthorizer struct {
	Roles role.Repository
}

func NewPermissionAuthorizer(roles role.Repository) *PermissionAuthorizer {
	return &PermissionAuthorizer{Roles: roles}
}

// Require returns middleware that denies the request unless the principal
// holds the given permission slug (e.g. "user:update").
func (p *PermissionAuthorizer) Require(slug string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := httpctx.Claims(r.Context())
			if !ok {
				httpctx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
				return
			}

			set, err := p.permissionSet(r.Context(), claims.UserID)
			if err != nil {
				httpctx.WriteError(w, http.StatusInternalServerError, "rbac_error", "failed to resolve permissions")
				return
			}
			ctx := httpctx.WithPermissions(r.Context(), set)
			if !httpctx.HasPermission(ctx, slug) {
				httpctx.WriteError(w, http.StatusForbidden, "forbidden", "insufficient permission")
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// permissionSet loads the user's permission slugs from their roles.
func (p *PermissionAuthorizer) permissionSet(ctx context.Context, userID int64) (map[string]bool, error) {
	slugs, err := p.Roles.PermissionsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(slugs))
	for _, s := range slugs {
		set[s] = true
	}
	return set, nil
}
