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
	"testing"

	"github.com/saskia-peters/gear/internal/platform/httpapi"
	"github.com/saskia-peters/gear/internal/user/core"
	"github.com/saskia-peters/gear/internal/user/ports"
)

type mockService struct {
	registerFunc func(ctx context.Context, input core.RegisterInput) (*ports.RegisterResult, error)
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
