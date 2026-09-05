// Package http hosts the HTTP adapter for the User Directory & Auth hexagon.
package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

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
