// Package auth provides the server-side authorization gateway (AD-2/AD-6):
// a chi middleware that validates a caller's bearer session token, resolves
// the caller's live permission set (AD-12) on every request — never a cached
// snapshot — and enforces a required permission code, answering with the
// uniform JSON error envelope (401 unauthorized / 403 forbidden).
package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/saskia-peters/gear/internal/platform/httpapi"
	"github.com/saskia-peters/gear/internal/user/core"
)

// SessionValidator validates a raw session token and returns the owning user.
type SessionValidator interface {
	Validate(ctx context.Context, rawToken string) (*core.Session, error)
}

// PermissionResolver resolves a user's live permission set (AD-12).
type PermissionResolver interface {
	ListPermissionsByUser(ctx context.Context, userID string) ([]string, error)
}

type userContextKey struct{}

// WithUser returns a context carrying the authenticated user.
func WithUser(ctx context.Context, user *core.User) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

// UserFrom returns the authenticated user stored in the request context, or
// nil when absent.
func UserFrom(ctx context.Context) *core.User {
	u, _ := ctx.Value(userContextKey{}).(*core.User)
	return u
}

// RequirePermission is the auth-gateway middleware (AD-2/AD-6). It validates
// the bearer token, re-derives the caller's live permission set and requires
// the given permission code. Missing/invalid/expired tokens return 401;
// authenticated callers lacking the permission return 403.
func RequirePermission(validator SessionValidator, resolver PermissionResolver, required string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := BearerToken(r)
			if token == "" {
				httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "Authentifizierung erforderlich.")
				return
			}

			sess, err := validator.Validate(r.Context(), token)
			if err != nil || sess.User == nil {
				httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "Authentifizierung erforderlich.")
				return
			}

			perms, err := resolver.ListPermissionsByUser(r.Context(), sess.User.ID)
			if err != nil {
				httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Ein interner Fehler ist aufgetreten.")
				return
			}

			if !hasPermission(perms, required) {
				// FR-19 existence-hiding: no disclosure of what the caller lacks.
				httpapi.WriteError(w, http.StatusForbidden, "forbidden", "Keine Berechtigung.")
				return
			}

			next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), sess.User)))
		})
	}
}

// BearerToken extracts the bearer token from the Authorization header. It is
// the single shared parser for the auth gateway and the user HTTP handlers.
func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) <= len(bearerPrefix) || !strings.EqualFold(h[:len(bearerPrefix)], bearerPrefix) {
		return ""
	}
	return strings.TrimSpace(h[len(bearerPrefix):])
}

const bearerPrefix = "Bearer "

func hasPermission(perms []string, required string) bool {
	for _, p := range perms {
		if p == required {
			return true
		}
	}
	return false
}

// Route returns a small chi router used to demonstrate the gateway on a
// protected demo endpoint. It is mounted by the composition root.
func Route(validator SessionValidator, resolver PermissionResolver, required string) http.Handler {
	r := chi.NewRouter()
	r.Use(RequirePermission(validator, resolver, required))
	r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
		u := UserFrom(r.Context())
		if u == nil {
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Ein interner Fehler ist aufgetreten.")
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{
			"id":           u.ID,
			"email":        u.Email,
			"display_name": u.DisplayName,
			"first_name":   u.FirstName,
			"last_name":    u.LastName,
			"state":        string(u.State),
		})
	})
	return r
}
