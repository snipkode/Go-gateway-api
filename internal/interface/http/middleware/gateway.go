package middleware

import (
	"net/http"
	"strings"

	httpctx "go-enterprise-api/internal/interface/httpapi"
)

// GatewayToken is the outer shared-secret check between the Nginx gateway and
// the API. When a token is configured, every request that does not carry
// `X-Gateway-Token` is rejected — so the API only trusts traffic that passed
// through the secured gateway. Health endpoints are exempt (container
// healthchecks and load-balancer probes hit them without the header).
func GatewayToken(token string) func(http.Handler) http.Handler {
	if token == "" {
		// No token configured → accept everything (dev/direct mode).
		return func(h http.Handler) http.Handler { return h }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/healthz") &&
				!strings.HasPrefix(r.URL.Path, "/readyz") &&
				r.Header.Get("X-Gateway-Token") != token {
				httpctx.WriteError(w, http.StatusForbidden, "gateway_required", "request must go through gateway")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
