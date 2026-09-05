package handler

import (
	"net/http"

	permapp "go-enterprise-api/internal/application/permission"
	"go-enterprise-api/internal/domain/permission"

	httpctx "go-enterprise-api/internal/interface/httpapi"
)

type PermissionHandler struct {
	Base
	Permissions permapp.UseCase
}

type permissionRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// @Summary List permissions
// @Tags permissions
// @Security BearerAuth
// @Produce json
// @Success 200 {array} permission.Permission
// @Router /api/v1/permissions [get]
func (h *PermissionHandler) List(w http.ResponseWriter, r *http.Request) {
	perms, err := h.Permissions.List(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusOK, perms)
}

// @Summary Create permission
// @Tags permissions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body permissionRequest true "Permission"
// @Success 201 {object} permission.Permission
// @Failure 409 {object} httpctx.ErrorResponse
// @Router /api/v1/permissions [post]
func (h *PermissionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req permissionRequest
	if !httpctx.DecodeJSON(w, r, &req) {
		return
	}
	created, err := h.Permissions.Create(r.Context(), permission.CreateParams{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusCreated, created)
}

// @Summary Update permission
// @Tags permissions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path integer true "Permission ID"
// @Param body body permissionRequest true "Permission"
// @Success 200 {object} permission.Permission
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/permissions/{id} [put]
func (h *PermissionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	var req permissionRequest
	if !httpctx.DecodeJSON(w, r, &req) {
		return
	}
	updated, err := h.Permissions.Update(r.Context(), id, permission.CreateParams{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusOK, updated)
}

// @Summary Soft delete permission
// @Tags permissions
// @Security BearerAuth
// @Param id path integer true "Permission ID"
// @Success 204 "No Content"
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/permissions/{id} [delete]
func (h *PermissionHandler) SoftDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	if err := h.Permissions.SoftDelete(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteNoContent(w)
}

// @Summary Restore soft-deleted permission
// @Tags permissions
// @Security BearerAuth
// @Param id path integer true "Permission ID"
// @Success 204 "No Content"
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/permissions/{id}/restore [post]
func (h *PermissionHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	if err := h.Permissions.Restore(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteNoContent(w)
}
