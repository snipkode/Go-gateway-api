package handler

import (
	"net/http"

	userapp "go-enterprise-api/internal/application/user"
	"go-enterprise-api/internal/domain/user"

	httpctx "go-enterprise-api/internal/interface/httpapi"
)

type UserHandler struct {
	Base
	Users userapp.UseCase
}

type createUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type updateUserRequest struct {
	Name   *string `json:"name"`
	Status *string `json:"status"`
}

// @Summary List users
// @Description Returns all active (non soft-deleted) users. Audited resource; RBAC protected.
// @Tags users
// @Security BearerAuth
// @Produce json
// @Success 200 {array} user.User
// @Failure 401 {object} httpctx.ErrorResponse
// @Failure 403 {object} httpctx.ErrorResponse
// @Router /api/v1/users [get]
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.Users.List(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusOK, users)
}

// @Summary Create user
// @Description Creates a local-provider user. The password is bcrypt-hashed in the application layer. Will not accept unknown JSON fields.
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body createUserRequest true "User to create"
// @Success 201 {object} user.User
// @Failure 400 {object} httpctx.ErrorResponse
// @Failure 401 {object} httpctx.ErrorResponse
// @Failure 403 {object} httpctx.ErrorResponse
// @Failure 409 {object} httpctx.ErrorResponse "email already in use"
// @Router /api/v1/users [post]
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if !httpctx.DecodeJSON(w, r, &req) {
		return
	}
	u, err := h.Users.Create(r.Context(), user.CreateParams{
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: req.Password,
		Provider:     "local",
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusCreated, u)
}

// @Summary Get user
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param id path integer true "User ID"
// @Success 200 {object} user.User
// @Failure 401 {object} httpctx.ErrorResponse
// @Failure 403 {object} httpctx.ErrorResponse
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	u, err := h.Users.Get(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusOK, u)
}

// @Summary Update user
// @Description Updates name and/or status. Audited (old → new data) inside the same transaction.
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path integer true "User ID"
// @Param body body updateUserRequest true "Fields to update"
// @Success 200 {object} user.User
// @Failure 400 {object} httpctx.ErrorResponse
// @Failure 401 {object} httpctx.ErrorResponse
// @Failure 403 {object} httpctx.ErrorResponse
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	var req updateUserRequest
	if !httpctx.DecodeJSON(w, r, &req) {
		return
	}
	if req.Name == nil && req.Status == nil {
		httpctx.WriteError(w, http.StatusBadRequest, "empty_update", "nothing to update")
		return
	}
	u, err := h.Users.Update(r.Context(), id, user.UpdateParams{Name: req.Name, Status: req.Status})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusOK, u)
}

// @Summary Soft delete user
// @Description Soft delete (sets deleted_at). The email becomes reusable thanks to the partial unique index.
// @Tags users
// @Security BearerAuth
// @Param id path integer true "User ID"
// @Success 204 "No Content"
// @Failure 401 {object} httpctx.ErrorResponse
// @Failure 403 {object} httpctx.ErrorResponse
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) SoftDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	if err := h.Users.SoftDelete(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteNoContent(w)
}

// @Summary Restore soft-deleted user
// @Description Clears deleted_at. Audited.
// @Tags users
// @Security BearerAuth
// @Param id path integer true "User ID"
// @Success 204 "No Content"
// @Failure 401 {object} httpctx.ErrorResponse
// @Failure 403 {object} httpctx.ErrorResponse
// @Failure 404 {object} httpctx.ErrorResponse
// @Router /api/v1/users/{id}/restore [post]
func (h *UserHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	if err := h.Users.Restore(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteNoContent(w)
}
