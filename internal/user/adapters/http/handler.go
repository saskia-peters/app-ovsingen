// Package http hosts the HTTP adapter for the User Directory & Auth hexagon.
package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/saskia-peters/gear/internal/platform/auth"
	"github.com/saskia-peters/gear/internal/platform/httpapi"
	"github.com/saskia-peters/gear/internal/user/core"
	"github.com/saskia-peters/gear/internal/user/ports"
)

// Handler serves HTTP requests for authentication and user registration.
type Handler struct {
	service ports.Service
	logger  *slog.Logger
}

// NewHandler constructs a User HTTP handler.
func NewHandler(service ports.Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// Routes returns a chi.Router with all auth routes mounted.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/logout", h.Logout)
	return r
}

// Register handles POST /api/v1/auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	var input core.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Ungültiges JSON-Format.")
		return
	}

	res, err := h.service.Register(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, core.ErrMissingFields):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", core.MsgMissingFields)
		case errors.Is(err, core.ErrInvalidEmail):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", core.MsgInvalidEmail)
		case errors.Is(err, core.ErrShortPassword):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", core.MsgShortPassword)
		case errors.Is(err, core.ErrPasswordMismatch):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", core.MsgPasswordMismatch)
		default:
			h.logger.Error("registration failed unexpectedly", "error", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Ein interner Fehler ist aufgetreten.")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, res)
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	var input core.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Ungültiges JSON-Format.")
		return
	}

	res, err := h.service.Login(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, core.ErrLockedOut):
			// Progressive lockout (FR-3): the email is temporarily blocked. The
			// message stays generic (never reveal why), Retry-After lets the
			// client count down, and the trigger is emitted to structured
			// logging (NFR-O1). A bare ErrLockedOut without a *LockoutError
			// falls back to a sane 30s window.
			retryAfter, ok := core.LockoutRetryAfter(err)
			seconds := core.LockoutDefaultRetrySeconds
			if ok && retryAfter > 0 {
				seconds = int(retryAfter.Seconds())
				if seconds < 1 {
					seconds = 1
				}
			} else {
				h.logger.Warn("login lockout triggered without retry details", "error", err)
			}
			var lockoutErr *core.LockoutError
			email := ""
			if errors.As(err, &lockoutErr) {
				email = lockoutErr.Email
			}
			h.logger.Warn("login lockout triggered", "email", email, "retry_after", seconds)
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			httpapi.WriteError(w, http.StatusTooManyRequests, "too_many_attempts",
				fmt.Sprintf("Zu viele Fehlversuche. Bitte warte %d Sekunden.", seconds))
		case errors.Is(err, core.ErrInvalidCredentials):
			// Anti-enumeration: identical response for every failure (UX-DR7).
			httpapi.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "E-Mail oder Passwort ist falsch.")
		case errors.Is(err, core.ErrInvalidLoginInput):
			// Oversized email/password rejected before the Argon2id verify.
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", core.MsgInvalidLoginInput)
		default:
			h.logger.Error("login failed unexpectedly", "error", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Ein interner Fehler ist aufgetreten.")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, res)
}

// Logout handles POST /api/v1/auth/logout. It invalidates the caller's
// session token server-side (NFR-S2). Logout is idempotent: it always returns
// 204, even for unknown/absent tokens.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := auth.BearerToken(r)
	if err := h.service.Logout(r.Context(), token); err != nil {
		h.logger.Error("logout failed unexpectedly", "error", err)
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Ein interner Fehler ist aufgetreten.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
