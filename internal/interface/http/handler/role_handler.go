package handler

import (
	"net/http"

	roleapp "go-enterprise-api/internal/application/role"
	"go-enterprise-api/internal/domain/role"

	httpctx "go-enterprise-api/internal/interface/httpapi"
)

type RoleHandler struct {
	Base
	Roles roleapp.UseCase
}

type roleRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type assignPermissionRequest struct {
	PermissionID int64 `json:"permission_id"`
}

// @Summary List roles
// @Tags roles
// @Security BearerAuth
// @Produce json
// @Success 200 {array} role.Role
// @Router /api/v1/roles [get]
func (h *RoleHandler) List(w http.ResponseWriter, r *http.Request) {
	roles, err := h.Roles.List(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusOK, roles)
}

// @Summary Create role
// @Tags roles
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body roleRequest true "Role"
// @Success 201 {object} role.Role
// @Failure 409 {object} httpctx.ErrorResponse "slug already in use"
// @Router /api/v1/roles [post]
func (h *RoleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req roleRequest
	if !httpctx.DecodeJSON(w, r, &req) {
		return
	}
	created, err := h.Roles.Create(r.Context(), role.CreateParams{
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

// @Summary Get role
// @Tags roles
// @Security BearerAuth
// @Produce json
// @Param id path integer true "Role ID"
// @Success 200 {object} role.Role
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/roles/{id} [get]
func (h *RoleHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	rl, err := h.Roles.Get(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusOK, rl)
}

// @Summary Update role
// @Tags roles
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path integer true "Role ID"
// @Param body body roleRequest true "Role"
// @Success 200 {object} role.Role
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/roles/{id} [put]
func (h *RoleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	var req roleRequest
	if !httpctx.DecodeJSON(w, r, &req) {
		return
	}
	rl, err := h.Roles.Update(r.Context(), id, role.UpdateParams{
		Name:        &req.Name,
		Description: &req.Description,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusOK, rl)
}

// @Summary Soft delete role
// @Tags roles
// @Security BearerAuth
// @Param id path integer true "Role ID"
// @Success 204 "No Content"
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/roles/{id} [delete]
func (h *RoleHandler) SoftDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	if err := h.Roles.SoftDelete(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteNoContent(w)
}

// @Summary Restore soft-deleted role
// @Tags roles
// @Security BearerAuth
// @Param id path integer true "Role ID"
// @Success 204 "No Content"
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/roles/{id}/restore [post]
func (h *RoleHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	if err := h.Roles.Restore(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteNoContent(w)
}

// @Summary Grant permission to role
// @Description Links a permission to a role. Audited as PERMISSION_GRANTED.
// @Tags roles
// @Security BearerAuth
// @Accept json
// @Param id path integer true "Role ID"
// @Param body body assignPermissionRequest true "Permission to grant"
// @Success 204 "No Content"
// @Failure 400 {object} httpctx.ErrorResponse
// @Router /api/v1/roles/{id}/permissions [post]
func (h *RoleHandler) AssignPermission(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	var req assignPermissionRequest
	if !httpctx.DecodeJSON(w, r, &req) {
		return
	}
	if err := h.Roles.AssignPermission(r.Context(), id, req.PermissionID); err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteNoContent(w)
}

// @Summary Revoke permission from role
// @Tags roles
// @Security BearerAuth
// @Param id path integer true "Role ID"
// @Param permissionId path integer true "Permission ID"
// @Success 204 "No Content"
// @Failure 400 {object} httpctx.ErrorResponse
// @Router /api/v1/roles/{id}/permissions/{permissionId} [delete]
func (h *RoleHandler) RemovePermission(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	pid, err := pathID(r, "permissionId")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid permission id")
		return
	}
	if err := h.Roles.RemovePermission(r.Context(), id, pid); err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteNoContent(w)
}
