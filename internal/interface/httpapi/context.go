package httpapi

import (
	"context"

	"go-enterprise-api/internal/domain/session"
)

type ctxKey int

const (
	ctxRequestID ctxKey = iota
	ctxClaims
	ctxPermissionSet
)

// WithRequestID stores the request id for correlation across logs and audit.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxRequestID, id)
}

func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(ctxRequestID).(string)
	return v
}

// WithClaims stores the authenticated principal.
func WithClaims(ctx context.Context, c *session.Claims) context.Context {
	return context.WithValue(ctx, ctxClaims, c)
}

func Claims(ctx context.Context) (*session.Claims, bool) {
	c, ok := ctx.Value(ctxClaims).(*session.Claims)
	return c, ok
}

// WithPermissions caches the principal's permission slugs for one request.
func WithPermissions(ctx context.Context, perms map[string]bool) context.Context {
	return context.WithValue(ctx, ctxPermissionSet, perms)
}

func HasPermission(ctx context.Context, slug string) bool {
	set, _ := ctx.Value(ctxPermissionSet).(map[string]bool)
	return set[slug]
}
