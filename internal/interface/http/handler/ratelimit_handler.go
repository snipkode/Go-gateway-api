package handler

import (
	"net/http"

	rlapp "go-enterprise-api/internal/application/ratelimit"
	"go-enterprise-api/internal/domain/ratelimit"

	httpctx "go-enterprise-api/internal/interface/httpapi"
)

type RateLimitHandler struct {
	Base
	Rules rlapp.UseCase
}

type ruleRequest struct {
	Name          string `json:"name"`
	Scope         string `json:"scope"`
	Identifier    string `json:"identifier"`
	Requests      int64  `json:"requests"`
	WindowSeconds int64  `json:"window_seconds"`
	Enabled       *bool  `json:"enabled"`
	Priority      *int   `json:"priority"`
}

// @Summary List rate limit rules
// @Description Dynamic rules stored in PostgreSQL. Changes apply live within the Redis cache TTL (30s).
// @Tags rate-limits
// @Security BearerAuth
// @Produce json
// @Success 200 {array} ratelimit.Rule
// @Router /api/v1/rate-limits [get]
func (h *RateLimitHandler) List(w http.ResponseWriter, r *http.Request) {
	rules, err := h.Rules.List(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusOK, rules)
}

// @Summary Create rate limit rule
// @Tags rate-limits
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body ruleRequest true "Rule"
// @Success 201 {object} ratelimit.Rule
// @Router /api/v1/rate-limits [post]
func (h *RateLimitHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req ruleRequest
	if !httpctx.DecodeJSON(w, r, &req) {
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	priority := 0
	if req.Priority != nil {
		priority = *req.Priority
	}
	rule, err := h.Rules.Create(r.Context(), ratelimit.CreateParams{
		Name:          req.Name,
		Scope:         req.Scope,
		Identifier:    req.Identifier,
		Requests:      req.Requests,
		WindowSeconds: req.WindowSeconds,
		Enabled:       enabled,
		Priority:      priority,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusCreated, rule)
}

// @Summary Update rate limit rule
// @Description Example: raise the Viewer role from 500 to 1000 req/min without redeploying. Audited old → new.
// @Tags rate-limits
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path integer true "Rule ID"
// @Param body body ruleRequest true "Rule"
// @Success 200 {object} ratelimit.Rule
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/rate-limits/{id} [put]
func (h *RateLimitHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	var req ruleRequest
	if !httpctx.DecodeJSON(w, r, &req) {
		return
	}
	p := ratelimit.UpdateParams{
		Name:          &req.Name,
		Requests:      &req.Requests,
		WindowSeconds: &req.WindowSeconds,
		Priority:      req.Priority,
	}
	if req.Enabled != nil {
		p.Enabled = req.Enabled
	}
	rule, err := h.Rules.Update(r.Context(), id, p)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusOK, rule)
}

// @Summary Soft delete rate limit rule
// @Description Disables the rule.
// @Tags rate-limits
// @Security BearerAuth
// @Param id path integer true "Rule ID"
// @Success 204 "No Content"
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/rate-limits/{id} [delete]
func (h *RateLimitHandler) SoftDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	if err := h.Rules.SoftDelete(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteNoContent(w)
}

// @Summary Restore soft-deleted rate limit rule
// @Tags rate-limits
// @Security BearerAuth
// @Param id path integer true "Rule ID"
// @Success 204 "No Content"
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/rate-limits/{id}/restore [post]
func (h *RateLimitHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	if err := h.Rules.Restore(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteNoContent(w)
}
