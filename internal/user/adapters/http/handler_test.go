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
	"strings"
	"testing"
	"time"

	"github.com/saskia-peters/gear/internal/platform/auth"
	"github.com/saskia-peters/gear/internal/platform/httpapi"
	"github.com/saskia-peters/gear/internal/user/core"
	"github.com/saskia-peters/gear/internal/user/ports"
)

type mockService struct {
	registerFunc func(ctx context.Context, input core.RegisterInput) (*ports.RegisterResult, error)
	loginFunc    func(ctx context.Context, input core.LoginInput) (*ports.LoginResult, error)
	logoutFunc   func(ctx context.Context, rawToken string) error
	enrollFunc   func(ctx context.Context, user *core.User) (*core.MFAEnrollResult, error)
	confirmFunc  func(ctx context.Context, user *core.User, secret, code string) error
	disableFunc  func(ctx context.Context, user *core.User, code string) error
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

func (m *mockService) EnrollMFARequest(ctx context.Context, user *core.User) (*core.MFAEnrollResult, error) {
	if m.enrollFunc != nil {
		return m.enrollFunc(ctx, user)
	}
	return &core.MFAEnrollResult{Secret: "SECRETBASE32", URI: "otpauth://totp/G.E.A.R.:max@example.com?secret=SECRETBASE32&issuer=G.E.A.R."}, nil
}

func (m *mockService) ConfirmMFAEnable(ctx context.Context, user *core.User, secret, code string) error {
	if m.confirmFunc != nil {
		return m.confirmFunc(ctx, user, secret, code)
	}
	return nil
}

func (m *mockService) DisableMFA(ctx context.Context, user *core.User, code string) error {
	if m.disableFunc != nil {
		return m.disableFunc(ctx, user, code)
	}
	return nil
}

func (m *mockService) MFAStatus(ctx context.Context, user *core.User) (bool, error) {
	if user != nil {
		return user.IsMFAEnabled, nil
	}
	return false, nil
}

func (m *mockService) RevokeOtherSessions(ctx context.Context, userID, rawToken string) error {
	return nil
}

func (m *mockService) RevokeAllSessions(ctx context.Context, userID string) error {
	return nil
}

// stubValidator always authenticates the caller as an active user. Used to
// exercise the MFA endpoints without a real session store.
type stubValidator struct {
	user *core.User
}

func (v *stubValidator) Validate(_ context.Context, _ string) (*core.Session, error) {
	if v == nil || v.user == nil {
		return &core.Session{User: &core.User{ID: "u-1", Email: "max@example.com", State: core.StateActive}}, nil
	}
	return &core.Session{User: v.user}, nil
}

// rejectingValidator always fails authentication.
type rejectingValidator struct{}

func (rejectingValidator) Validate(context.Context, string) (*core.Session, error) {
	return nil, core.ErrSessionNotFound
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestHandler(svc ports.Service, validator auth.SessionValidator) *Handler {
	return NewHandler(svc, discardLogger(), validator)
}

func TestHandlerRegisterHappyPath(t *testing.T) {
	svc := &mockService{}
	h := newTestHandler(svc, &stubValidator{})

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
			h := newTestHandler(svc, &stubValidator{})

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
	h := newTestHandler(svc, &stubValidator{})

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
	h := newTestHandler(svc, &stubValidator{})

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
	h := newTestHandler(svc, &stubValidator{})

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
	h := newTestHandler(svc, &stubValidator{})

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
			h := newTestHandler(svc, &stubValidator{})

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
	h := newTestHandler(svc, &stubValidator{})

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
	h := newTestHandler(svc, &stubValidator{})

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
	h := newTestHandler(svc, &stubValidator{})

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
	h := newTestHandler(svc, &stubValidator{})

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
	h := newTestHandler(svc, &stubValidator{})

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
	h := newTestHandler(svc, &stubValidator{})

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (idempotent)", rec.Code)
	}
}

func TestHandlerLoginMFAChallengeReturns200NoToken(t *testing.T) {
	svc := &mockService{
		loginFunc: func(ctx context.Context, input core.LoginInput) (*ports.LoginResult, error) {
			return &ports.LoginResult{MFARequired: true}, nil
		},
	}
	h := newTestHandler(svc, &stubValidator{})

	payload := map[string]string{"email": "max@example.com", "password": "geheim123456"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for mfa_required", rec.Code)
	}
	var wire map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire["mfa_required"] != true {
		t.Errorf("mfa_required = %v, want true", wire["mfa_required"])
	}
	if _, present := wire["token"]; present {
		t.Error("challenge response must not carry a token")
	}
}

func TestHandlerLoginMFAChallengeSuccessReturnsToken(t *testing.T) {
	svc := &mockService{
		loginFunc: func(ctx context.Context, input core.LoginInput) (*ports.LoginResult, error) {
			return &ports.LoginResult{
				Token: "opaque-token",
				User:  core.LoginUser{ID: "u-1", Email: "max@example.com", DisplayName: "Max", FirstName: "Max", LastName: "Mustermann"},
			}, nil
		},
	}
	h := newTestHandler(svc, &stubValidator{})

	payload := map[string]string{"email": "max@example.com", "password": "geheim123456", "totp_code": "123456"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var res ports.LoginResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Token != "opaque-token" {
		t.Errorf("token = %q, want opaque-token", res.Token)
	}
}

func TestHandlerLoginMFAChallengeFailureReturns401(t *testing.T) {
	svc := &mockService{
		loginFunc: func(ctx context.Context, input core.LoginInput) (*ports.LoginResult, error) {
			return nil, core.ErrInvalidCredentials
		},
	}
	h := newTestHandler(svc, &stubValidator{})

	payload := map[string]string{"email": "max@example.com", "password": "geheim123456", "totp_code": "000000"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	var env httpapi.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != "invalid_credentials" {
		t.Errorf("code = %q, want invalid_credentials", env.Error.Code)
	}
	// Anti-enumeration: identical microcopy as a wrong password (UX-DR7).
	if env.Error.Message != "E-Mail oder Passwort ist falsch." {
		t.Errorf("message = %q, want anti-enumeration microcopy", env.Error.Message)
	}
}

// authedMFARequest builds a request carrying an authenticated user in context,
// mirroring what the RequireAuth middleware injects.
func authedMFARequest(method, target string, body []byte) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	user := &core.User{ID: "u-1", Email: "max@example.com", State: core.StateActive}
	return req.WithContext(auth.WithUser(req.Context(), user))
}

func TestHandlerMFAEnrollRequest(t *testing.T) {
	svc := &mockService{}
	h := newTestHandler(svc, &stubValidator{})

	req := authedMFARequest(http.MethodPost, "/mfa/enroll", []byte("{}"))
	rec := httptest.NewRecorder()

	h.MFAEnroll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var res core.MFAEnrollResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Secret == "" {
		t.Error("expected a shared secret in the enroll response")
	}
	if res.URI == "" {
		t.Error("expected a provisioning URI in the enroll response")
	}
}

func TestHandlerMFAEnrollConfirmValid(t *testing.T) {
	var gotUser *core.User
	var gotSecret, gotCode string
	svc := &mockService{
		confirmFunc: func(ctx context.Context, user *core.User, secret, code string) error {
			gotUser = user
			gotSecret = secret
			gotCode = code
			return nil
		},
	}
	h := newTestHandler(svc, &stubValidator{})

	payload := map[string]string{"secret": "SECRETBASE32", "code": "123456"}
	body, _ := json.Marshal(payload)
	req := authedMFARequest(http.MethodPost, "/mfa/enroll", body)
	rec := httptest.NewRecorder()

	h.MFAEnroll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var wire map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire["enabled"] != true {
		t.Errorf("enabled = %v, want true", wire["enabled"])
	}
	if gotUser == nil || gotUser.Email != "max@example.com" {
		t.Errorf("user not passed through, got %+v", gotUser)
	}
	if gotSecret != "SECRETBASE32" || gotCode != "123456" {
		t.Errorf("secret/code not forwarded: %q / %q", gotSecret, gotCode)
	}
}

func TestHandlerMFAEnrollConfirmInvalidCodeReturns400(t *testing.T) {
	svc := &mockService{
		confirmFunc: func(ctx context.Context, user *core.User, secret, code string) error {
			return core.ErrTOTPInvalid
		},
	}
	h := newTestHandler(svc, &stubValidator{})

	payload := map[string]string{"secret": "SECRETBASE32", "code": "000000"}
	body, _ := json.Marshal(payload)
	req := authedMFARequest(http.MethodPost, "/mfa/enroll", body)
	rec := httptest.NewRecorder()

	h.MFAEnroll(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var env httpapi.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != "invalid_totp" {
		t.Errorf("code = %q, want invalid_totp", env.Error.Code)
	}
}

func TestHandlerMFAEnrollAlreadyEnabled(t *testing.T) {
	svc := &mockService{
		enrollFunc: func(ctx context.Context, user *core.User) (*core.MFAEnrollResult, error) {
			return nil, core.ErrMFAAlreadyEnabled
		},
	}
	h := newTestHandler(svc, &stubValidator{})

	req := authedMFARequest(http.MethodPost, "/mfa/enroll", []byte("{}"))
	rec := httptest.NewRecorder()

	h.MFAEnroll(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandlerMFADisableValid(t *testing.T) {
	var gotCode string
	svc := &mockService{
		disableFunc: func(ctx context.Context, user *core.User, code string) error {
			gotCode = code
			return nil
		},
	}
	h := newTestHandler(svc, &stubValidator{})

	payload := map[string]string{"code": "123456"}
	body, _ := json.Marshal(payload)
	req := authedMFARequest(http.MethodPost, "/mfa/disable", body)
	rec := httptest.NewRecorder()

	h.MFADisable(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var wire map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire["enabled"] != false {
		t.Errorf("enabled = %v, want false", wire["enabled"])
	}
	if gotCode != "123456" {
		t.Errorf("code = %q, want 123456", gotCode)
	}
}

func TestHandlerMFADisableInvalidCodeReturns400(t *testing.T) {
	svc := &mockService{
		disableFunc: func(ctx context.Context, user *core.User, code string) error {
			return core.ErrTOTPInvalid
		},
	}
	h := newTestHandler(svc, &stubValidator{})

	payload := map[string]string{"code": "000000"}
	body, _ := json.Marshal(payload)
	req := authedMFARequest(http.MethodPost, "/mfa/disable", body)
	rec := httptest.NewRecorder()

	h.MFADisable(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var env httpapi.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != "invalid_totp" {
		t.Errorf("code = %q, want invalid_totp", env.Error.Code)
	}
}

func TestHandlerMFADisableNotEnabled(t *testing.T) {
	svc := &mockService{
		disableFunc: func(ctx context.Context, user *core.User, code string) error {
			return core.ErrMFANotEnabled
		},
	}
	h := newTestHandler(svc, &stubValidator{})

	payload := map[string]string{"code": "123456"}
	body, _ := json.Marshal(payload)
	req := authedMFARequest(http.MethodPost, "/mfa/disable", body)
	rec := httptest.NewRecorder()

	h.MFADisable(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandlerMFAUnauthenticated(t *testing.T) {
	// The MFA routes are wrapped in the auth middleware: no valid bearer token
	// must yield a uniform 401.
	svc := &mockService{}
	h := NewHandler(svc, discardLogger(), &rejectingValidator{})

	req := httptest.NewRequest(http.MethodPost, "/mfa/enroll", bytes.NewReader([]byte("{}")))
	rec := httptest.NewRecorder()

	r := h.Routes()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 from the auth middleware", rec.Code)
	}
}

func TestHandlerMFAStatusContract(t *testing.T) {
	// Review finding 1.6-10: GET /api/v1/auth/mfa/status returns
	// {"enabled":true} for an MFA-enabled user and {"enabled":false} for a
	// disabled one, via the real RequireAuth middleware + Routes().
	tests := []struct {
		name    string
		user    *core.User
		want    bool
	}{
		{name: "enabled", user: &core.User{ID: "u-1", Email: "max@example.com", State: core.StateActive, IsMFAEnabled: true}, want: true},
		{name: "disabled", user: &core.User{ID: "u-1", Email: "max@example.com", State: core.StateActive, IsMFAEnabled: false}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(&mockService{}, discardLogger(), &stubValidator{user: tt.user})
			req := httptest.NewRequest(http.MethodGet, "/mfa/status", nil)
			req.Header.Set("Authorization", "Bearer valid-token")
			rec := httptest.NewRecorder()

			h.Routes().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			var wire map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
				t.Fatal(err)
			}
			if wire["enabled"] != tt.want {
				t.Errorf("enabled = %v, want %v", wire["enabled"], tt.want)
			}
		})
	}
}

func TestHandlerMFAStatusUnauthenticated(t *testing.T) {
	h := NewHandler(&mockService{}, discardLogger(), &rejectingValidator{})
	req := httptest.NewRequest(http.MethodGet, "/mfa/status", nil)
	rec := httptest.NewRecorder()

	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandlerMFAUnavailableMapsTo503(t *testing.T) {
	// Review finding 1.6-3: encryption-key misconfiguration surfaces as a clear
	// 503 "MFA ist derzeit nicht verfügbar.", not a generic 500.
	svc := &mockService{
		enrollFunc: func(ctx context.Context, user *core.User) (*core.MFAEnrollResult, error) {
			return nil, core.ErrMFAUnavailable
		},
	}
	h := newTestHandler(svc, &stubValidator{})

	req := authedMFARequest(http.MethodPost, "/mfa/enroll", []byte("{}"))
	rec := httptest.NewRecorder()
	h.MFAEnroll(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	var env httpapi.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != "mfa_unavailable" {
		t.Errorf("code = %q, want mfa_unavailable", env.Error.Code)
	}
	if env.Error.Message != "MFA ist derzeit nicht verfügbar." {
		t.Errorf("message = %q, want MFA-unavailable microcopy", env.Error.Message)
	}
}

func TestHandlerLoginMFAEnrollmentExpiredReturns400(t *testing.T) {
	svc := &mockService{
		confirmFunc: func(ctx context.Context, user *core.User, secret, code string) error {
			return core.ErrMFAEnrollmentExpired
		},
	}
	h := newTestHandler(svc, &stubValidator{})

	payload := map[string]string{"secret": "SECRETBASE32", "code": "123456"}
	body, _ := json.Marshal(payload)
	req := authedMFARequest(http.MethodPost, "/mfa/enroll", body)
	rec := httptest.NewRecorder()

	h.MFAEnroll(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var env httpapi.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != "invalid_totp" {
		t.Errorf("code = %q, want invalid_totp", env.Error.Code)
	}
}

func TestHandlerLoginChallengeSuccessLogRequiresMFA(t *testing.T) {
	// Review finding 1.6-11: the "mfa challenge success" log must only fire when
	// MFA was actually involved (user.IsMFAEnabled), not for a spurious
	// totp_code on an account without MFA.
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	// No-MFA account sending a spurious totp_code.
	svc := &mockService{
		loginFunc: func(ctx context.Context, input core.LoginInput) (*ports.LoginResult, error) {
			return &ports.LoginResult{
				Token: "opaque-token",
				User:  core.LoginUser{ID: "u-1", Email: "max@example.com", IsMFAEnabled: false},
			}, nil
		},
	}
	h := NewHandler(svc, logger, &stubValidator{})
	payload := map[string]string{"email": "max@example.com", "password": "geheim123456", "totp_code": "123456"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if strings.Contains(buf.String(), "mfa challenge success") {
		t.Errorf("spurious totp_code must not log mfa challenge success, got %q", buf.String())
	}

	// MFA-enabled account with a valid flow.
	buf.Reset()
	svc2 := &mockService{
		loginFunc: func(ctx context.Context, input core.LoginInput) (*ports.LoginResult, error) {
			return &ports.LoginResult{
				Token: "opaque-token",
				User:  core.LoginUser{ID: "u-1", Email: "max@example.com", IsMFAEnabled: true},
			}, nil
		},
	}
	h2 := NewHandler(svc2, logger, &stubValidator{})
	rec2 := httptest.NewRecorder()
	h2.Login(rec2, httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body)))
	if !strings.Contains(buf.String(), "mfa challenge success") {
		t.Errorf("MFA-enabled login must log mfa challenge success, got %q", buf.String())
	}
}

func TestHandlerLoginFailureLoggingIsUniform(t *testing.T) {
	// Review finding 1.6-4: a failed TOTP challenge logs the SAME uniform
	// "login failed" event as a wrong password — readers cannot distinguish the
	// failing stage.
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	svc := &mockService{
		loginFunc: func(ctx context.Context, input core.LoginInput) (*ports.LoginResult, error) {
			return nil, core.ErrInvalidCredentials
		},
	}
	h := NewHandler(svc, logger, &stubValidator{})

	payload := map[string]string{"email": "max@example.com", "password": "geheim123456", "totp_code": "000000"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	logs := buf.String()
	if !strings.Contains(logs, "login failed") {
		t.Errorf("expected a uniform 'login failed' log, got %q", logs)
	}
	if strings.Contains(logs, "mfa challenge failed") {
		t.Errorf("must not log a stage-specific failure, got %q", logs)
	}
}

func TestHandlerLoginMFAUnavailableMapsTo503(t *testing.T) {
	// Review finding 1.6-3: a rotated/missing encryption key during the TOTP
	// step surfaces as a clear 503, not a misleading generic 500.
	svc := &mockService{
		loginFunc: func(ctx context.Context, input core.LoginInput) (*ports.LoginResult, error) {
			return nil, core.ErrMFAUnavailable
		},
	}
	h := newTestHandler(svc, &stubValidator{})

	payload := map[string]string{"email": "max@example.com", "password": "geheim123456", "totp_code": "123456"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	var env httpapi.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != "mfa_unavailable" {
		t.Errorf("code = %q, want mfa_unavailable", env.Error.Code)
	}
	if env.Error.Message != "MFA ist derzeit nicht verfügbar." {
		t.Errorf("message = %q, want MFA-unavailable microcopy", env.Error.Message)
	}
}
