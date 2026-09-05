// Package router assembles the HTTP routing surface shared by the server
// composition root and its tests: the middleware chain, the JSON 404/405
// handlers, and the /healthz mount. The chi router emits plain-text bodies for
// unmatched routes and panics by default, so this package overrides them with
// the uniform JSON error envelope.
package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/saskia-peters/gear/internal/platform/health"
	"github.com/saskia-peters/gear/internal/platform/httpapi"
	appmw "github.com/saskia-peters/gear/internal/platform/middleware"
)

// New returns the mounted chi router. The panic-recovery middleware and the
// 404/405 handlers answer with the uniform JSON envelope (not plain text), and
// /healthz is wired to the given Pinger with the structured request logger.
func New(p health.Pinger, log *slog.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(appmw.Recovery(log))
	r.Use(appmw.RequestLogger(log))
	r.NotFound(httpapi.NotFoundHandler())
	r.MethodNotAllowed(httpapi.MethodNotAllowedHandler())
	r.Get("/healthz", health.New(p, log))
	return r
}

// compile-time check: *chi.Mux satisfies http.Handler.
var _ http.Handler = (*chi.Mux)(nil)