package handler

import (
	"net/http"

	gwapp "go-enterprise-api/internal/application/gateway"
	"go-enterprise-api/internal/domain/gatewayapi"
	"go-enterprise-api/internal/infrastructure/gatewaymonitor"

	httpctx "go-enterprise-api/internal/interface/httpapi"
)

type GatewayHandler struct {
	Base
	Gateway gwapp.UseCase
	Monitor *gatewaymonitor.Aggregator
}

type createGatewayAPIRequest struct {
	Name         string   `json:"name"`
	BasePath     string   `json:"base_path"`
	Upstream     string   `json:"upstream"`
	Methods      []string `json:"methods"`
	RequiresAuth *bool    `json:"requires_auth"`
	RateLimitRPM *int     `json:"rate_limit_rpm"`
	IsActive     *bool    `json:"is_active"`
	Note         string   `json:"note"`
}

type updateGatewayAPIRequest struct {
	Name         *string   `json:"name"`
	BasePath     *string   `json:"base_path"`
	Upstream     *string   `json:"upstream"`
	Methods      *[]string `json:"methods"`
	RequiresAuth *bool     `json:"requires_auth"`
	RateLimitRPM *int      `json:"rate_limit_rpm"`
	IsActive     *bool     `json:"is_active"`
	Note         *string   `json:"note"`
}

func (r createGatewayAPIRequest) toParams() gatewayapi.CreateParams {
	p := gatewayapi.CreateParams{
		Name: r.Name, BasePath: r.BasePath, Upstream: r.Upstream,
		Methods: r.Methods, Note: r.Note,
	}
	if r.RequiresAuth != nil {
		p.RequiresAuth = *r.RequiresAuth
	}
	if r.RateLimitRPM != nil {
		p.RateLimitRPM = *r.RateLimitRPM
	}
	if r.IsActive != nil {
		p.IsActive = *r.IsActive
	}
	return p
}

// @Summary List registered gateway APIs
// @Description Returns all registered APIs managed by the gateway console, with live health status.
// @Tags gateway
// @Security BearerAuth
// @Produce json
// @Success 200 {array} gatewayapi.GatewayAPI
// @Failure 401 {object} httpctx.ErrorResponse
// @Failure 403 {object} httpctx.ErrorResponse
// @Router /api/v1/gateway/apis [get]
func (h *GatewayHandler) List(w http.ResponseWriter, r *http.Request) {
	apis, err := h.Gateway.List(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusOK, apis)
}

// @Summary Register a gateway API
// @Description Registers an upstream API and publishes it to the gateway (nginx include + hot reload) immediately. base_path must be a single segment like /orders; upstream is the origin url. Methods default to GET, rate_limit_rpm to 60, requires_auth to true.
// @Tags gateway
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body createGatewayAPIRequest true "API to register"
// @Success 201 {object} gatewayapi.GatewayAPI
// @Failure 400 {object} httpctx.ErrorResponse
// @Failure 401 {object} httpctx.ErrorResponse
// @Failure 403 {object} httpctx.ErrorResponse
// @Failure 409 {object} httpctx.ErrorResponse "base path already in use"
// @Router /api/v1/gateway/apis [post]
func (h *GatewayHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createGatewayAPIRequest
	if !httpctx.DecodeJSON(w, r, &req) {
		return
	}
	api, err := h.Gateway.Create(r.Context(), req.toParams())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusCreated, api)
}

// @Summary Get one registered gateway API
// @Tags gateway
// @Security BearerAuth
// @Produce json
// @Param id path integer true "Gateway API ID"
// @Success 200 {object} gatewayapi.GatewayAPI
// @Failure 401 {object} httpctx.ErrorResponse
// @Failure 403 {object} httpctx.ErrorResponse
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/gateway/apis/{id} [get]
func (h *GatewayHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	api, err := h.Gateway.Get(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusOK, api)
}

// @Summary Update a registered gateway API
// @Description Updates fields and re-publishes the gateway config. Audited (old → new data).
// @Tags gateway
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path integer true "Gateway API ID"
// @Param body body updateGatewayAPIRequest true "Fields to update"
// @Success 200 {object} gatewayapi.GatewayAPI
// @Failure 400 {object} httpctx.ErrorResponse
// @Failure 401 {object} httpctx.ErrorResponse
// @Failure 403 {object} httpctx.ErrorResponse
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/gateway/apis/{id} [put]
func (h *GatewayHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	var req updateGatewayAPIRequest
	if !httpctx.DecodeJSON(w, r, &req) {
		return
	}
	api, err := h.Gateway.Update(r.Context(), id, gatewayapi.UpdateParams{
		Name: req.Name, BasePath: req.BasePath, Upstream: req.Upstream,
		Methods: req.Methods, RequiresAuth: req.RequiresAuth,
		RateLimitRPM: req.RateLimitRPM, IsActive: req.IsActive, Note: req.Note,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusOK, api)
}

// @Summary Soft delete a registered gateway API
// @Description Removes the API from the gateway (generated nginx config is dropped).
// @Tags gateway
// @Security BearerAuth
// @Param id path integer true "Gateway API ID"
// @Success 204 "No Content"
// @Failure 401 {object} httpctx.ErrorResponse
// @Failure 403 {object} httpctx.ErrorResponse
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/gateway/apis/{id} [delete]
func (h *GatewayHandler) SoftDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	if err := h.Gateway.SoftDelete(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteNoContent(w)
}

// @Summary Restore a soft-deleted gateway API
// @Description Re-registers the API and re-publishes the gateway config.
// @Tags gateway
// @Security BearerAuth
// @Param id path integer true "Gateway API ID"
// @Success 204 "No Content"
// @Failure 401 {object} httpctx.ErrorResponse
// @Failure 403 {object} httpctx.ErrorResponse
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/gateway/apis/{id}/restore [post]
func (h *GatewayHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	if err := h.Gateway.Restore(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteNoContent(w)
}

// @Summary Preview generated nginx config (no publish)
// @Description Simulates the exact nginx location block the gateway would apply.
// @Tags gateway
// @Security BearerAuth
// @Produce plain
// @Param id path integer true "Gateway API ID"
// @Success 200 {string} string "generated nginx snippet"
// @Failure 401 {object} httpctx.ErrorResponse
// @Failure 403 {object} httpctx.ErrorResponse
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/gateway/apis/{id}/preview [get]
func (h *GatewayHandler) Preview(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	snippet, err := h.Gateway.Preview(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(snippet))
}

// @Summary Re-publish the full registry to the gateway
// @Description Regenerates every nginx include and triggers the gateway hot-reload.
// @Tags gateway
// @Security BearerAuth
// @Success 204 "No Content"
// @Failure 401 {object} httpctx.ErrorResponse
// @Failure 403 {object} httpctx.ErrorResponse
// @Router /api/v1/gateway/publish [post]
func (h *GatewayHandler) Publish(w http.ResponseWriter, r *http.Request) {
	if err := h.Gateway.Publish(r.Context()); err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteNoContent(w)
}

// @Summary Gateway API statistics
// @Description Today's totals (Redis) plus the most recent proxied requests (in-memory ring) for one registered API.
// @Tags gateway
// @Security BearerAuth
// @Produce json
// @Param id path integer true "Gateway API ID"
// @Success 200 {object} gatewayStatsResponse
// @Failure 401 {object} httpctx.ErrorResponse
// @Failure 403 {object} httpctx.ErrorResponse
// @Router /api/v1/gateway/apis/{id}/stats [get]
func (h *GatewayHandler) Stats(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	if _, err := h.Gateway.Get(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	if h.Monitor == nil {
		httpctx.WriteError(w, http.StatusServiceUnavailable, "monitor_disabled", "gateway monitoring is disabled")
		return
	}
	today, recent := h.Monitor.Stats(r.Context(), id)
	httpctx.WriteJSON(w, http.StatusOK, gatewayStatsResponse{Today: today, Recent: recent})
}

type gatewayStatsResponse struct {
	Today  gatewaymonitor.TodayStats `json:"today"`
	Recent []gatewaymonitor.Entry    `json:"recent"`
}
