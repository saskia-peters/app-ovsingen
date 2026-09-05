package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/saskia-peters/gear/internal/platform/httpapi"
	"github.com/saskia-peters/gear/internal/user/core"
	"github.com/saskia-peters/gear/internal/user/ports"
)

type mockService struct {
	registerFunc func(ctx context.Context, input core.RegisterInput) (*ports.RegisterResult, error)
	loginFunc    func(ctx context.Context, input core.LoginInput) (*ports.LoginResult, error)
	logoutFunc   func(ctx context.Context, rawToken string) error
}

func (m *mockService) Register(ctx context.Context, input core.RegisterInput) (*ports.RegisterResult, error) {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, input)
	}
	return &ports.RegisterResult{
		Message: core.UniformSuccessMessage,
		Status:  "pending_approval",
	}, nil
}

func (m *mockService) Login(ctx context.Context, input core.LoginInput) (*ports.LoginResult, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, input)
	}
	return &ports.LoginResult{
		Token: "opaque-token",
		User: core.LoginUser{
			ID:          "u-1",
			Email:       "max@example.com",
			DisplayName: "Max Mustermann",
			FirstName:   "Max",
			LastName:    "Mustermann",
		},
	}, nil
}

func (m *mockService) Logout(ctx context.Context, rawToken string) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(ctx, rawToken)
	}
	return nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHandlerRegisterHappyPath(t *testing.T) {
	svc := &mockService{}
	h := NewHandler(svc, discardLogger())

	payload := map[string]string{
		"first_name":       "Max",
		"last_name":        "Mustermann",
		"email":            "max@example.com",
		"password":         "sicher123456",
		"password_confirm": "sicher123456",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var res ports.RegisterResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Message != core.UniformSuccessMessage {
		t.Errorf("got message %q, want %q", res.Message, core.UniformSuccessMessage)
	}
	if res.Status != "pending_approval" {
		t.Errorf("got status %q, want %q", res.Status, "pending_approval")
	}
}

func TestHandlerRegisterValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		serviceErr  error
		wantMessage string
	}{
		{
			name:        "short password",
			serviceErr:  core.ErrShortPassword,
			wantMessage: "Das Passwort muss mindestens 10 Zeichen lang sein.",
		},
		{
			name:        "password mismatch",
			serviceErr:  core.ErrPasswordMismatch,
			wantMessage: "Die Passwörter stimmen nicht überein.",
		},
		{
			name:        "invalid email",
			serviceErr:  core.ErrInvalidEmail,
			wantMessage: "Bitte gib eine gültige E-Mail-Adresse ein.",
		},
		{
			name:        "missing fields",
			serviceErr:  core.ErrMissingFields,
			wantMessage: "Alle Pflichtfelder müssen ausgefüllt sein.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockService{
				registerFunc: func(ctx context.Context, input core.RegisterInput) (*ports.RegisterResult, error) {
					return nil, tt.serviceErr
				},
			}
			h := NewHandler(svc, discardLogger())

			payload := map[string]string{
				"first_name":       "Max",
				"last_name":        "Mustermann",
				"email":            "max@example.com",
				"password":         "pass",
				"password_confirm": "pass",
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			h.Register(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}

			var env httpapi.ErrorEnvelope
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("failed to decode error envelope: %v", err)
			}

			if env.Error.Code != "invalid_request" {
				t.Errorf("code = %q, want %q", env.Error.Code, "invalid_request")
			}
			if env.Error.Message != tt.wantMessage {
				t.Errorf("message = %q, want %q", env.Error.Message, tt.wantMessage)
			}
		})
	}
}

func TestHandlerRegisterInvalidJSON(t *testing.T) {
	svc := &mockService{}
	h := NewHandler(svc, discardLogger())

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader([]byte("{invalid-json")))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var env httpapi.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to decode error envelope: %v", err)
	}
	if env.Error.Code != "invalid_request" {
		t.Errorf("code = %q, want %q", env.Error.Code, "invalid_request")
	}
}

func TestHandlerRegisterInternalError(t *testing.T) {
	svc := &mockService{
		registerFunc: func(ctx context.Context, input core.RegisterInput) (*ports.RegisterResult, error) {
			return nil, errors.New("db connection lost")
		},
	}
	h := NewHandler(svc, discardLogger())

	payload := map[string]string{
		"first_name":       "Max",
		"last_name":        "Mustermann",
		"email":            "max@example.com",
		"password":         "1234567890",
		"password_confirm": "1234567890",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	var env httpapi.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to decode error envelope: %v", err)
	}
	if env.Error.Code != "internal_error" {
		t.Errorf("code = %q, want %q", env.Error.Code, "internal_error")
	}
}

func TestHandlerLoginHappyPath(t *testing.T) {
	svc := &mockService{}
	h := NewHandler(svc, discardLogger())

	payload := map[string]string{"email": "max@example.com", "password": "geheim123456"}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var res ports.LoginResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.Token != "opaque-token" {
		t.Errorf("token = %q, want opaque-token", res.Token)
	}
	if res.User.Email != "max@example.com" {
		t.Errorf("user email = %q, want max@example.com", res.User.Email)
	}
}

func TestHandlerLoginInvalidCredentials(t *testing.T) {
	svc := &mockService{
		loginFunc: func(ctx context.Context, input core.LoginInput) (*ports.LoginResult, error) {
			return nil, core.ErrInvalidCredentials
		},
	}
	h := NewHandler(svc, discardLogger())

	payload := map[string]string{"email": "nobody@example.com", "password": "falsch"}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}

	var env httpapi.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to decode error envelope: %v", err)
	}
	if env.Error.Code != "invalid_credentials" {
		t.Errorf("code = %q, want invalid_credentials", env.Error.Code)
	}
	if env.Error.Message != "E-Mail oder Passwort ist falsch." {
		t.Errorf("message = %q, want anti-enumeration microcopy", env.Error.Message)
	}
}

func TestHandlerLoginLockedOutReturns429WithRetryAfter(t *testing.T) {
	tests := []struct {
		name         string
		retryAfter   time.Duration
		wantSeconds  string
		wantMessage  string
	}{
		{name: "30 second lockout", retryAfter: 30 * time.Second, wantSeconds: "30", wantMessage: "Zu viele Fehlversuche. Bitte warte 30 Sekunden."},
		{name: "60 second lockout", retryAfter: 60 * time.Second, wantSeconds: "60", wantMessage: "Zu viele Fehlversuche. Bitte warte 60 Sekunden."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockService{
				loginFunc: func(ctx context.Context, input core.LoginInput) (*ports.LoginResult, error) {
					return nil, core.NewLockoutError(tt.retryAfter, "active@example.com")
				},
			}
			h := NewHandler(svc, discardLogger())

			payload := map[string]string{"email": "active@example.com", "password": "geheim123456"}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			h.Login(rec, req)

			if rec.Code != http.StatusTooManyRequests {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
			}
			if got := rec.Header().Get("Retry-After"); got != tt.wantSeconds {
				t.Errorf("Retry-After = %q, want %q", got, tt.wantSeconds)
			}

			var env httpapi.ErrorEnvelope
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("failed to decode error envelope: %v", err)
			}
			if env.Error.Code != "too_many_attempts" {
				t.Errorf("code = %q, want too_many_attempts", env.Error.Code)
			}
			if env.Error.Message != tt.wantMessage {
				t.Errorf("message = %q, want %q", env.Error.Message, tt.wantMessage)
			}
			// The Retry-After header value must parse to the advertised seconds.
			parsed, err := strconv.Atoi(rec.Header().Get("Retry-After"))
			if err != nil || parsed <= 0 {
				t.Errorf("Retry-After = %q is not a positive integer", rec.Header().Get("Retry-After"))
			}
		})
	}
}

func TestHandlerLoginBareLockedOutFallsBackToSaneRetry(t *testing.T) {
	// A lockout error that is NOT a *LockoutError (bare sentinel) must still
	// produce a valid 429 with a sane Retry-After default, not seconds=1.
	svc := &mockService{
		loginFunc: func(ctx context.Context, input core.LoginInput) (*ports.LoginResult, error) {
			return nil, core.ErrLockedOut
		},
	}
	h := NewHandler(svc, discardLogger())

	payload := map[string]string{"email": "active@example.com", "password": "geheim123456"}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
	if got := rec.Header().Get("Retry-After"); got != "30" {
		t.Errorf("Retry-After = %q, want default 30", got)
	}

	var env httpapi.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to decode error envelope: %v", err)
	}
	if env.Error.Code != "too_many_attempts" {
		t.Errorf("code = %q, want too_many_attempts", env.Error.Code)
	}
	if env.Error.Message != "Zu viele Fehlversuche. Bitte warte 30 Sekunden." {
		t.Errorf("message = %q, want 30s fallback microcopy", env.Error.Message)
	}
}

func TestHandlerLoginInvalidJSON(t *testing.T) {
	svc := &mockService{}
	h := NewHandler(svc, discardLogger())

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte("{bad")))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandlerLoginInternalError(t *testing.T) {
	svc := &mockService{
		loginFunc: func(ctx context.Context, input core.LoginInput) (*ports.LoginResult, error) {
			return nil, errors.New("db down")
		},
	}
	h := NewHandler(svc, discardLogger())

	payload := map[string]string{"email": "a@example.com", "password": "x"}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestHandlerLoginInvalidInput(t *testing.T) {
	svc := &mockService{
		loginFunc: func(ctx context.Context, input core.LoginInput) (*ports.LoginResult, error) {
			return nil, core.ErrInvalidLoginInput
		},
	}
	h := NewHandler(svc, discardLogger())

	payload := map[string]string{"email": "a@example.com", "password": "x"}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var env httpapi.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to decode error envelope: %v", err)
	}
	if env.Error.Code != "invalid_request" {
		t.Errorf("code = %q, want invalid_request", env.Error.Code)
	}
	if env.Error.Message != core.MsgInvalidLoginInput {
		t.Errorf("message = %q, want %q", env.Error.Message, core.MsgInvalidLoginInput)
	}
}

func TestHandlerLogoutHappyPath(t *testing.T) {
	var gotToken string
	svc := &mockService{
		logoutFunc: func(ctx context.Context, rawToken string) error {
			gotToken = rawToken
			return nil
		},
	}
	h := NewHandler(svc, discardLogger())

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.Header.Set("Authorization", "Bearer sesstoken123")
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if gotToken != "sesstoken123" {
		t.Errorf("logout token = %q, want sesstoken123", gotToken)
	}
}

func TestHandlerLogoutWithoutToken(t *testing.T) {
	svc := &mockService{}
	h := NewHandler(svc, discardLogger())

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (idempotent)", rec.Code)
	}
}
