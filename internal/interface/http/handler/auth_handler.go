package handler

import (
	"log/slog"
	"net/http"
	"time"

	authapp "go-enterprise-api/internal/application/auth"
	"go-enterprise-api/internal/infrastructure/entra"

	httpctx "go-enterprise-api/internal/interface/httpapi"
)

type AuthHandler struct {
	Base
	Auth authapp.UseCase

	// Entra is the OIDC provider adapter; nil means the flow is disabled.
	Entra  entra.Provider
	States *entra.StateStore
	// FrontendURL, when set, makes the OIDC callback redirect the browser to
	// <FrontendURL>#access_token=... (SPA flow) instead of returning JSON.
	FrontendURL string
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// @Summary Login with email/password
// @Description Authenticates with the local provider and returns a JWT access token. Route-limited by the dynamic rule POST:/api/v1/auth/login (10 req/min by default).
// @Tags auth
// @Accept json
// @Produce json
// @Param body body loginRequest true "Credentials"
// @Success 200 {object} authapp.LoginResult
// @Failure 401 {object} httpctx.ErrorResponse
// @Failure 429 {object} httpctx.ErrorResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !httpctx.DecodeJSON(w, r, &req) {
		return
	}

	result, err := h.Auth.Login(r.Context(), authapp.LoginParams{
		Email:     req.Email,
		Password:  req.Password,
		IP:        clientIPFromRequest(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpctx.WriteJSON(w, http.StatusOK, result)
}

// @Summary Logout / revoke session
// @Description Revokes the Redis session, invalidating the JWT immediately.
// @Tags auth
// @Security BearerAuth
// @Success 204 "No Content"
// @Failure 401 {object} httpctx.ErrorResponse
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	c, ok := httpctx.Claims(r.Context())
	if !ok || c.SessionID == "" {
		httpctx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
		return
	}
	if err := h.Auth.Logout(r.Context(), c.SessionID); err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteNoContent(w)
}

// @Summary Current user profile
// @Description Returns the authenticated user with their effective roles.
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]any
// @Failure 401 {object} httpctx.ErrorResponse
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	c, ok := httpctx.Claims(r.Context())
	if !ok {
		httpctx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
		return
	}
	profile, err := h.Auth.Profile(r.Context(), c)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpctx.WriteJSON(w, http.StatusOK, profile)
}

func clientIPFromRequest(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	return r.RemoteAddr
}

// @Summary Start Microsoft Entra (OIDC) login
// @Description Issues a CSRF state and 302-redirects the browser to Microsoft. Unconfigured when OAUTH_CLIENT_ID/OAUTH_TENANT_ID are empty → 503 so the SPA can fall back to the local form.
// @Tags auth
// @Success 302 "Redirect to login.microsoftonline.com"
// @Failure 503 {object} httpctx.ErrorResponse
// @Router /api/v1/auth/entra/login [get]
func (h *AuthHandler) EntraLogin(w http.ResponseWriter, r *http.Request) {
	log := h.log(r)
	if h.Entra == nil || h.States == nil {
		httpctx.WriteError(w, http.StatusServiceUnavailable, "sso_not_configured", "entra oidc is not configured")
		return
	}
	state := h.States.New()
	authURL, err := h.Entra.AuthURL(r.Context(), state)
	if err != nil {
		log.Warn("entra auth url failed", slog.String("error", err.Error()))
		httpctx.WriteError(w, http.StatusServiceUnavailable, "sso_not_configured", "entra oidc is not configured")
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

// @Summary Entra OIDC callback
// @Description Exchanges the authorization code for a verified profile, JIT-provisions the user with the default role and issues a session. Returns the standard LoginResult JSON (or a fragment redirect when OAUTH_FRONTEND_URL is set).
// @Tags auth
// @Produce json
// @Param code query string true "Authorization code from Microsoft"
// @Param state query string true "CSRF state returned from /entra/login"
// @Success 200 {object} authapp.LoginResult
// @Failure 400 {object} httpctx.ErrorResponse "missing/invalid code or state"
// @Failure 502 {object} httpctx.ErrorResponse "exchange with Microsoft failed"
// @Failure 503 {object} httpctx.ErrorResponse
// @Router /api/v1/auth/entra/callback [get]
func (h *AuthHandler) EntraCallback(w http.ResponseWriter, r *http.Request) {
	log := h.log(r)
	if h.Entra == nil || h.States == nil {
		httpctx.WriteError(w, http.StatusServiceUnavailable, "sso_not_configured", "entra oidc is not configured")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_request", "missing authorization code")
		return
	}
	if !h.States.Verify(r.URL.Query().Get("state")) {
		httpctx.WriteError(w, http.StatusBadRequest, "invalid_state", "state is missing, expired or already used")
		return
	}

	profile, err := h.Entra.Exchange(r.Context(), code)
	if err != nil {
		log.Warn("entra exchange failed", slog.String("error", err.Error()))
		httpctx.WriteError(w, http.StatusBadGateway, "exchange_failed", "failed to exchange authorization code")
		return
	}

	result, err := h.Auth.EntraLogin(r.Context(), authapp.EntraLoginParams{
		ProfileID: profile.ID,
		Email:     profile.Email,
		Name:      profile.Name,
		IP:        clientIPFromRequest(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	if h.FrontendURL != "" {
		http.Redirect(w, r, h.FrontendURL+"#access_token="+result.AccessToken+"&expires_at="+result.ExpiresAt.Format(time.RFC3339), http.StatusFound)
		return
	}
	httpctx.WriteJSON(w, http.StatusOK, result)
}

func (h *AuthHandler) log(r *http.Request) *slog.Logger {
	if h.Logger != nil {
		return h.Logger
	}
	return slog.Default()
}
