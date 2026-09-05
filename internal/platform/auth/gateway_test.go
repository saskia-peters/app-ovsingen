package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saskia-peters/gear/internal/platform/httpapi"
	"github.com/saskia-peters/gear/internal/user/core"
)

const protectedCode = "admin.recovery.approve"

type mockValidator struct {
	session *core.Session
	err     error
}

func (m *mockValidator) Validate(_ context.Context, _ string) (*core.Session, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.session, nil
}

type mockResolver struct {
	perms []string
	err   error
}

func (m *mockResolver) ListPermissionsByUser(_ context.Context, _ string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.perms, nil
}

func newProtectedRouter(v SessionValidator, r PermissionResolver) http.Handler {
	router := chi.NewRouter()
	router.Use(RequirePermission(v, r, protectedCode))
	router.Get("/demo", func(w http.ResponseWriter, _ *http.Request) {
		httpapi.WriteJSON(w, http.StatusOK, map[string]string{"ok": "true"})
	})
	return router
}

func doRequest(h http.Handler, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/demo", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func activeUser() *core.User {
	return &core.User{ID: "u-1", Email: "a@example.com", State: core.StateActive}
}

func TestGatewayNoToken(t *testing.T) {
	v := &mockValidator{}
	r := &mockResolver{}
	h := newProtectedRouter(v, r)

	rec := doRequest(h, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	var env httpapi.ErrorEnvelope
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env.Error.Code != "unauthorized" {
		t.Errorf("code = %q, want unauthorized", env.Error.Code)
	}
}

func TestGatewayInvalidToken(t *testing.T) {
	v := &mockValidator{err: core.ErrSessionNotFound}
	r := &mockResolver{}
	h := newProtectedRouter(v, r)

	rec := doRequest(h, "garbage-token")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	var env httpapi.ErrorEnvelope
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env.Error.Code != "unauthorized" {
		t.Errorf("code = %q, want unauthorized", env.Error.Code)
	}
}

func TestGatewayForbidden(t *testing.T) {
	v := &mockValidator{session: &core.Session{User: activeUser()}}
	r := &mockResolver{perms: []string{"some.other.perm"}}
	h := newProtectedRouter(v, r)

	rec := doRequest(h, "valid-token")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	var env httpapi.ErrorEnvelope
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env.Error.Code != "forbidden" {
		t.Errorf("code = %q, want forbidden", env.Error.Code)
	}
}

func TestGatewayAllowed(t *testing.T) {
	user := activeUser()
	v := &mockValidator{session: &core.Session{User: user}}
	r := &mockResolver{perms: []string{protectedCode}}
	h := newProtectedRouter(v, r)

	rec := doRequest(h, "valid-token")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestGatewayUserInContext(t *testing.T) {
	user := activeUser()
	v := &mockValidator{session: &core.Session{User: user}}
	r := &mockResolver{perms: []string{protectedCode}}

	router := chi.NewRouter()
	router.Use(RequirePermission(v, r, protectedCode))
	router.Get("/demo", func(w http.ResponseWriter, req *http.Request) {
		u := UserFrom(req.Context())
		if u == nil || u.ID != "u-1" {
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "no user")
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]string{"user": u.ID})
	})

	rec := doRequest(router, "valid-token")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestGatewayResolverError(t *testing.T) {
	v := &mockValidator{session: &core.Session{User: activeUser()}}
	r := &mockResolver{err: errors.New("db down")}
	h := newProtectedRouter(v, r)

	rec := doRequest(h, "valid-token")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestGatewayBearerTokenParsing(t *testing.T) {
	if got := BearerToken(httptest.NewRequest(http.MethodGet, "/", nil)); got != "" {
		t.Errorf("missing header should yield empty token, got %q", got)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer abc123")
	if got := BearerToken(req); got != "abc123" {
		t.Errorf("bearer token = %q, want abc123", got)
	}
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Authorization", "bearer abc123")
	if got := BearerToken(req2); got != "abc123" {
		t.Errorf("case-insensitive scheme = %q, want abc123", got)
	}
	// An exactly-empty bearer header yields no token.
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.Header.Set("Authorization", "Bearer ")
	if got := BearerToken(req3); got != "" {
		t.Errorf("empty bearer token should yield empty, got %q", got)
	}
}

// inMemorySessionStore is a minimal SessionStore used to drive the real
// SessionManager (and thus the real expiry check) through the gateway.
type inMemorySessionStore struct {
	sessions map[string]*core.Session
	users    map[string]*core.User
	nextID   int
}

func newInMemorySessionStore() *inMemorySessionStore {
	return &inMemorySessionStore{sessions: make(map[string]*core.Session)}
}

func (m *inMemorySessionStore) CreateSession(_ context.Context, userID, tokenHash string, expiresAt time.Time) (*core.Session, error) {
	m.nextID++
	s := &core.Session{
		ID:        fmt.Sprintf("sess-%d", m.nextID),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
		User:      m.users[userID],
	}
	m.sessions[tokenHash] = s
	return s, nil
}

func (m *inMemorySessionStore) GetSessionByTokenHash(_ context.Context, tokenHash string) (*core.Session, error) {
	s, ok := m.sessions[tokenHash]
	if !ok {
		return nil, core.ErrSessionNotFound
	}
	return s, nil
}

func (m *inMemorySessionStore) DeleteSessionByTokenHash(_ context.Context, tokenHash string) error {
	delete(m.sessions, tokenHash)
	return nil
}

// TestGatewayExpiredWithRealSessionManager exercises the actual expiry logic
// (SessionManager.Validate re-checks the stored expires_at) through the
// gateway, rather than faking it with a mock error.
func TestGatewayExpiredWithRealSessionManager(t *testing.T) {
	store := newInMemorySessionStore()
	user := activeUser()
	store.users = map[string]*core.User{user.ID: user}
	sm := core.NewSessionManager(store, time.Hour)

	raw, err := sm.Issue(context.Background(), user)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	// Backdate the stored session so it is past idle expiry.
	for _, s := range store.sessions {
		s.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	}

	r := &mockResolver{perms: []string{protectedCode}}
	h := newProtectedRouter(sm, r)

	rec := doRequest(h, raw)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for expired session", rec.Code)
	}
	var env httpapi.ErrorEnvelope
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env.Error.Code != "unauthorized" {
		t.Errorf("code = %q, want unauthorized", env.Error.Code)
	}
}

// TestGatewayValidWithRealSessionManager confirms the real SessionManager
// accepts a fresh session through the gateway (200).
func TestGatewayValidWithRealSessionManager(t *testing.T) {
	store := newInMemorySessionStore()
	user := activeUser()
	store.users = map[string]*core.User{user.ID: user}
	sm := core.NewSessionManager(store, time.Hour)

	raw, err := sm.Issue(context.Background(), user)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	r := &mockResolver{perms: []string{protectedCode}}
	h := newProtectedRouter(sm, r)

	if rec := doRequest(h, raw); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for a fresh session", rec.Code)
	}
}
