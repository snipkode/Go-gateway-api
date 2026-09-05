package handler

import (
	"net/http"

	httpctx "go-enterprise-api/internal/interface/httpapi"
)

// HealthHandler exposes liveness/readiness for orchestrators and load balancers.
type HealthHandler struct {
	// Ready is invoked by /readyz; return an error when a dependency is down.
	Ready func() error
}

// @Summary Liveness probe
// @Description Returns 200 while the process is up. Used by the container healthcheck and load balancers.
// @Tags system
// @Produce json
// @Success 200 {object} map[string]string
// @Router /healthz [get]
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	httpctx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// @Summary Readiness probe
// @Description Returns 200 when Postgres and Redis are reachable, otherwise 503.
// @Tags system
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} httpctx.ErrorResponse
// @Router /readyz [get]
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	if h.Ready != nil {
		if err := h.Ready(); err != nil {
			httpctx.WriteError(w, http.StatusServiceUnavailable, "not_ready", err.Error())
			return
		}
	}
	httpctx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
