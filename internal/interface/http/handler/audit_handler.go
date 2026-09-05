package handler

import (
	"net/http"
	"strconv"

	auditapp "go-enterprise-api/internal/application/audit"
	"go-enterprise-api/internal/domain/audit"

	httpctx "go-enterprise-api/internal/interface/httpapi"
)

type AuditHandler struct {
	Base
	Audit auditapp.UseCase
}

// @Summary Search audit logs
// @Description Paginated, filterable trail of WHO/WHAT/DATA/WHEN/FROM. Filters: user_id, action, resource, resource_id, request_id.
// @Tags audit
// @Security BearerAuth
// @Produce json
// @Param user_id query integer false "Filter by user"
// @Param action query string false "Filter by action (LOGIN, USER_CREATED, …)"
// @Param resource query string false "Filter by resource (users, roles, …)"
// @Param resource_id query string false "Filter by resource id"
// @Param request_id query string false "Filter by request id"
// @Param page query integer false "Page number (default 1)"
// @Param page_size query integer false "Page size (default 20)"
// @Success 200 {array} audit.Entry
// @Router /api/v1/audit-logs [get]
func (h *AuditHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	entries, total, err := h.Audit.Search(r.Context(), audit.SearchFilter{
		UserID:     intQuery(r, "user_id"),
		Action:     q.Get("action"),
		Resource:   q.Get("resource"),
		ResourceID: q.Get("resource_id"),
		RequestID:  q.Get("request_id"),
		Page:       intOr(q.Get("page"), 1),
		PageSize:   intOr(q.Get("page_size"), 20),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteMetaJSON(w, http.StatusOK, entries,
		map[string]any{
			"total":     total,
			"page":      intOr(q.Get("page"), 1),
			"page_size": intOr(q.Get("page_size"), 20),
		})
}

// @Summary Write a manual audit entry
// @Description Records a raw audit entry. The authenticated principal, request id, method and path are attached automatically.
// @Tags audit
// @Security BearerAuth
// @Accept json
// @Param body body audit.Entry true "Audit entry"
// @Success 204 "No Content"
// @Router /api/v1/audit-logs [post]
func (h *AuditHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req audit.Entry
	if !httpctx.DecodeJSON(w, r, &req) {
		return
	}
	if c, ok := httpctx.Claims(r.Context()); ok && c != nil {
		req.UserID = c.UserID
	}
	req.RequestID = httpctx.RequestID(r.Context())
	req.Method = r.Method
	req.Path = r.URL.Path
	if err := h.Audit.Log(r.Context(), req); err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteNoContent(w)
}

func intQuery(r *http.Request, key string) int64 {
	n, _ := strconv.ParseInt(r.URL.Query().Get(key), 10, 64)
	return n
}

func intOr(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}
