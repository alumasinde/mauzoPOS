package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/audit"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
)

// Handler exposes auth endpoints for both login contexts.
type Handler struct {
	svc   *Service
	audit *audit.Writer
}

// NewHandler builds a Handler.
func NewHandler(svc *Service, audit *audit.Writer) *Handler {
	return &Handler{svc: svc, audit: audit}
}

// LoginPlatform handles POST /platform/auth/login.
func (h *Handler) LoginPlatform(w http.ResponseWriter, r *http.Request) {
	var req PlatformLoginRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	res, err := h.svc.LoginPlatform(r.Context(), req.Email, req.Password, metaFromRequest(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, loginResponse(res))
}

// LoginTenant handles POST /t/auth/login.
func (h *Handler) LoginTenant(w http.ResponseWriter, r *http.Request) {
	var req TenantLoginRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	res, err := h.svc.LoginTenant(r.Context(), req.Tenant, req.Identity, req.Password, metaFromRequest(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, loginResponse(res))
}

// VerifyMFA handles POST */auth/mfa/verify.
func (h *Handler) VerifyMFA(w http.ResponseWriter, r *http.Request) {
	var req MFAVerifyRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	res, err := h.svc.VerifyMFA(r.Context(), req.Ticket, req.Code, metaFromRequest(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, loginResponse(res))
}

// ForgotPasswordPlatform handles POST /platform/auth/password/forgot.
func (h *Handler) ForgotPasswordPlatform(w http.ResponseWriter, r *http.Request) {
	h.forgot(w, r, identity.ActorPlatformUser)
}

// ForgotPasswordTenant handles POST /t/auth/password/forgot.
func (h *Handler) ForgotPasswordTenant(w http.ResponseWriter, r *http.Request) {
	h.forgot(w, r, identity.ActorUser)
}

func (h *Handler) forgot(w http.ResponseWriter, r *http.Request, actorType identity.ActorType) {
	var req ForgotPasswordRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.svc.ForgotPassword(r.Context(), actorType, req.Tenant, req.Identity); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// ResetPassword handles POST */auth/password/reset.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.svc.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// Logout handles POST */auth/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	actor, _ := identity.ActorFrom(r.Context())
	if err := h.svc.Logout(r.Context(), actor); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// Me handles GET */auth/me.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	actor, _ := identity.ActorFrom(r.Context())
	acct, t, err := h.svc.Profile(r.Context(), actor)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, profileResponse(acct, t))
}

// MFASetup handles POST */auth/mfa/setup.
func (h *Handler) MFASetup(w http.ResponseWriter, r *http.Request) {
	actor, _ := identity.ActorFrom(r.Context())
	secret, url, err := h.svc.MFASetup(r.Context(), actor)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, MFASetupResponse{Secret: secret, OTPAuthURL: url})
}

// MFAEnable handles POST */auth/mfa/enable.
func (h *Handler) MFAEnable(w http.ResponseWriter, r *http.Request) {
	var req MFACodeRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, _ := identity.ActorFrom(r.Context())
	codes, err := h.svc.MFAEnable(r.Context(), actor, req.Code)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.audit.Log(r.Context(), audit.FromRequest(r, audit.Entry{Action: "auth.mfa.enable"}))
	httpx.OK(w, MFAEnableResponse{RecoveryCodes: codes})
}

// MFADisable handles POST */auth/mfa/disable.
func (h *Handler) MFADisable(w http.ResponseWriter, r *http.Request) {
	var req MFADisableRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, _ := identity.ActorFrom(r.Context())
	if err := h.svc.MFADisable(r.Context(), actor, req.Password); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.audit.Log(r.Context(), audit.FromRequest(r, audit.Entry{Action: "auth.mfa.disable"}))
	httpx.NoContent(w)
}

// Sessions handles GET */auth/sessions.
func (h *Handler) Sessions(w http.ResponseWriter, r *http.Request) {
	actor, _ := identity.ActorFrom(r.Context())
	ss, err := h.svc.Sessions(r.Context(), actor)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]SessionResponse, len(ss))
	for i, s := range ss {
		out[i] = SessionResponse{
			ID:         s.PublicID,
			IP:         s.IPAddress,
			UserAgent:  s.UserAgent,
			Current:    s.ID == actor.SessionID,
			CreatedAt:  s.CreatedAt,
			LastUsedAt: s.LastUsedAt,
		}
	}
	httpx.OK(w, out)
}

// RevokeSession handles DELETE */auth/sessions/{id}.
func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	actor, _ := identity.ActorFrom(r.Context())
	if err := h.svc.RevokeSession(r.Context(), actor, chi.URLParam(r, "id")); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.audit.Log(r.Context(), audit.FromRequest(r, audit.Entry{
		Action: "auth.session.revoke", EntityType: "session", EntityID: chi.URLParam(r, "id"),
	}))
	httpx.NoContent(w)
}

// History handles GET */auth/history.
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	actor, _ := identity.ActorFrom(r.Context())
	rows, err := h.svc.LoginHistory(r.Context(), actor)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]HistoryResponse, len(rows))
	for i, a := range rows {
		var reason string
		if a.FailureReason != nil {
			reason = *a.FailureReason
		}
		out[i] = HistoryResponse{
			Identity: a.Identity, Success: a.Success, FailureReason: reason,
			IP: a.IPAddress, UserAgent: a.UserAgent, CreatedAt: a.CreatedAt,
		}
	}
	httpx.OK(w, out)
}
