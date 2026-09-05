package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	authapp "go-enterprise-api/internal/application/auth"
	userapp "go-enterprise-api/internal/application/user"
	"go-enterprise-api/internal/domain/ratelimit"
	"go-enterprise-api/internal/domain/role"
	"go-enterprise-api/internal/domain/user"

	httpctx "go-enterprise-api/internal/interface/httpapi"
)

// Base carries cross-cutting dependencies used by every handler.
type Base struct {
	Logger *slog.Logger
}

// pathID parses a positive integer path parameter.
func pathID(r *http.Request, name string) (int64, error) {
	raw := r.PathValue(name)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

// mapError translates sentinel domain errors into HTTP responses.
func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrNotFound),
		errors.Is(err, role.ErrNotFound),
		errors.Is(err, ratelimit.ErrNotFound):
		httpctx.WriteError(w, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, user.ErrEmailTaken), errors.Is(err, userapp.ErrEmailTaken):
		httpctx.WriteError(w, http.StatusConflict, "conflict", "email already in use")
	case errors.Is(err, user.ErrInvalidCred),
		errors.Is(err, authapp.ErrInvalidCredentials),
		errors.Is(err, authapp.ErrSessionNotFound):
		httpctx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
	case errors.Is(err, authapp.ErrSSOAccountConflict),
		errors.Is(err, authapp.ErrSSOIdentityConflict):
		httpctx.WriteError(w, http.StatusConflict, "sso_conflict", "sso login conflicts with existing account")
	case errors.Is(err, authapp.ErrSSODefaultRole):
		httpctx.WriteError(w, http.StatusInternalServerError, "sso_not_configured", "sso default role is misconfigured")
	default:
		httpctx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
