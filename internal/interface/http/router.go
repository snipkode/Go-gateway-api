package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/swaggo/http-swagger"

	"go-enterprise-api/internal/domain/ratelimit"
	"go-enterprise-api/internal/domain/role"
	"go-enterprise-api/internal/domain/session"
	"go-enterprise-api/internal/interface/http/handler"
	"go-enterprise-api/internal/interface/http/middleware"
)

// Dependencies groups every service the router needs. main.go wires concrete
// implementations into this struct; the router only cares about HTTP.
type Dependencies struct {
	Logger   *slog.Logger
	TokenSvc session.TokenService
	Sessions session.Repository
	Roles    role.Repository

	// GatewayToken is the shared secret between the Nginx gateway and the API;
	// when set, direct traffic is rejected (middleware/gateway.go).
	GatewayToken string
	// SwaggerEnabled mounts the OpenAPI spec + swagger-ui when true.
	SwaggerEnabled bool

	AuthHandler       *handler.AuthHandler
	UserHandler       *handler.UserHandler
	RoleHandler       *handler.RoleHandler
	PermissionHandler *handler.PermissionHandler
	RateLimitHandler  *handler.RateLimitHandler
	AuditHandler      *handler.AuditHandler
	GatewayHandler    *handler.GatewayHandler
	HealthHandler     *handler.HealthHandler

	RateLimitEvaluator ratelimit.Evaluator
}

// NewRouter builds the HTTP mux with the enterprise middleware chain.
//
//	Request ID → Gateway token → Recovery → CORS → Rate Limit ─> mux
//	                                                                ├─ public handlers (health, login)
//	                                                                ├─ Auth ─> Handler           (me, logout)
//	                                                                └─ Auth → Permission → Handler
//
// Recovery wraps everything so panics are logged with the request id and a 500
// is returned without killing the process.
func NewRouter(deps Dependencies, allowedOrigins []string) *http.Server {
	mux := http.NewServeMux()

	rateLimiter := middleware.NewRateLimiter(deps.RateLimitEvaluator, deps.TokenSvc)
	authenticator := &middleware.Authenticator{
		Tokens:   deps.TokenSvc,
		Sessions: deps.Sessions,
	}
	permissions := middleware.NewPermissionAuthorizer(deps.Roles)

	// protected wraps a handler with auth, and optionally a permission check.
	protected := func(required string, next http.HandlerFunc) http.Handler {
		h := http.Handler(next)
		if required != "" {
			h = permissions.Require(required)(h)
		}
		return authenticator.Authorize(h)
	}
	// open registers a handler with no authentication, but still behind the
	// global middleware (rate limit + recovery + request id).
	open := func(h http.HandlerFunc) http.Handler { return http.Handler(h) }

	// ── Public ─────────────────────────────────────────────────────────
	mux.Handle("GET /healthz", open(deps.HealthHandler.Liveness))
	mux.Handle("GET /readyz", open(deps.HealthHandler.Readiness))
	mux.Handle("POST /api/v1/auth/login", open(deps.AuthHandler.Login))

	// Microsoft Entra (OIDC). Only reachable through the gateway; guarded by
	// zone_auth at Nginx and by the per-route app rate limiter as well.
	mux.Handle("GET /api/v1/auth/entra/login", open(deps.AuthHandler.EntraLogin))
	mux.Handle("GET /api/v1/auth/entra/callback", open(deps.AuthHandler.EntraCallback))

	// ── Authenticated ──────────────────────────────────────────────────
	mux.Handle("POST /api/v1/auth/logout", protected("", deps.AuthHandler.Logout))
	mux.Handle("GET /api/v1/auth/me", protected("", deps.AuthHandler.Me))

	// ── Users ────────────────────────────────────────────────────────────
	mux.Handle("GET /api/v1/users", protected("user:read", deps.UserHandler.List))
	mux.Handle("POST /api/v1/users", protected("user:create", deps.UserHandler.Create))
	mux.Handle("GET /api/v1/users/{id}", protected("user:read", deps.UserHandler.Get))
	mux.Handle("PUT /api/v1/users/{id}", protected("user:update", deps.UserHandler.Update))
	mux.Handle("DELETE /api/v1/users/{id}", protected("user:delete", deps.UserHandler.SoftDelete))
	mux.Handle("POST /api/v1/users/{id}/restore", protected("user:restore", deps.UserHandler.Restore))

	// ── Roles ────────────────────────────────────────────────────────────
	mux.Handle("GET /api/v1/roles", protected("role:read", deps.RoleHandler.List))
	mux.Handle("POST /api/v1/roles", protected("role:write", deps.RoleHandler.Create))
	mux.Handle("GET /api/v1/roles/{id}", protected("role:read", deps.RoleHandler.Get))
	mux.Handle("PUT /api/v1/roles/{id}", protected("role:write", deps.RoleHandler.Update))
	mux.Handle("DELETE /api/v1/roles/{id}", protected("role:write", deps.RoleHandler.SoftDelete))
	mux.Handle("POST /api/v1/roles/{id}/restore", protected("role:write", deps.RoleHandler.Restore))
	mux.Handle("POST /api/v1/roles/{id}/permissions", protected("permission:assign", deps.RoleHandler.AssignPermission))
	mux.Handle("DELETE /api/v1/roles/{id}/permissions/{permissionId}", protected("permission:assign", deps.RoleHandler.RemovePermission))

	// ── Permissions ─────────────────────────────────────────────────────
	mux.Handle("GET /api/v1/permissions", protected("role:read", deps.PermissionHandler.List))
	mux.Handle("POST /api/v1/permissions", protected("permission:assign", deps.PermissionHandler.Create))
	mux.Handle("PUT /api/v1/permissions/{id}", protected("permission:assign", deps.PermissionHandler.Update))
	mux.Handle("DELETE /api/v1/permissions/{id}", protected("permission:assign", deps.PermissionHandler.SoftDelete))
	mux.Handle("POST /api/v1/permissions/{id}/restore", protected("permission:assign", deps.PermissionHandler.Restore))

	// ── Rate limits (dynamic admin) ─────────────────────────────────────
	mux.Handle("GET /api/v1/rate-limits", protected("ratelimit:read", deps.RateLimitHandler.List))
	mux.Handle("POST /api/v1/rate-limits", protected("ratelimit:write", deps.RateLimitHandler.Create))
	mux.Handle("PUT /api/v1/rate-limits/{id}", protected("ratelimit:write", deps.RateLimitHandler.Update))
	mux.Handle("DELETE /api/v1/rate-limits/{id}", protected("ratelimit:write", deps.RateLimitHandler.SoftDelete))
	mux.Handle("POST /api/v1/rate-limits/{id}/restore", protected("ratelimit:write", deps.RateLimitHandler.Restore))

	// ── Audit logs ──────────────────────────────────────────────────────
	mux.Handle("GET /api/v1/audit-logs", protected("audit:read", deps.AuditHandler.Search))
	mux.Handle("POST /api/v1/audit-logs", protected("audit:read", deps.AuditHandler.Create))

	// ── API Gateway management console ─────────────────────────────────
	mux.Handle("GET /api/v1/gateway/apis", protected("apigateway:read", deps.GatewayHandler.List))
	mux.Handle("POST /api/v1/gateway/apis", protected("apigateway:manage", deps.GatewayHandler.Create))
	mux.Handle("GET /api/v1/gateway/apis/{id}", protected("apigateway:read", deps.GatewayHandler.Get))
	mux.Handle("PUT /api/v1/gateway/apis/{id}", protected("apigateway:manage", deps.GatewayHandler.Update))
	mux.Handle("DELETE /api/v1/gateway/apis/{id}", protected("apigateway:manage", deps.GatewayHandler.SoftDelete))
	mux.Handle("POST /api/v1/gateway/apis/{id}/restore", protected("apigateway:manage", deps.GatewayHandler.Restore))
	mux.Handle("GET /api/v1/gateway/apis/{id}/preview", protected("apigateway:read", deps.GatewayHandler.Preview))
	mux.Handle("GET /api/v1/gateway/apis/{id}/stats", protected("apigateway:read", deps.GatewayHandler.Stats))
	mux.Handle("POST /api/v1/gateway/publish", protected("apigateway:manage", deps.GatewayHandler.Publish))

	// Internal endpoint used by nginx auth_request subrequests for
	// registered APIs that require auth (returns 200/401 only).
	mux.Handle("GET /internal/auth", protected("", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// ── OpenAPI / Swagger UI ────────────────────────────────────────────
	if deps.SwaggerEnabled {
		mux.Handle("GET /swagger/{any}", open(httpSwagger.WrapHandler))
		mux.Handle("GET /swagger/", open(httpSwagger.WrapHandler))
	}

	// Global cross-cutting chain (single pass over every request).
	// Request ID → Gateway token → Recovery → CORS → Rate Limit → mux
	top := middleware.Chain(
		middleware.RequestID,
		middleware.GatewayToken(deps.GatewayToken),
		func(h http.Handler) http.Handler { return middleware.Recovery(deps.Logger, h) },
		middleware.CORS(allowedOrigins),
		rateLimiter.Limit,
	)(mux)

	return &http.Server{
		Handler:           top,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
