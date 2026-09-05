package config

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func mapEnv(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadDefaults(t *testing.T) {
	cfg := Load(mapEnv(nil))

	if cfg.DatabaseURL != DefaultDatabaseURL {
		t.Errorf("DatabaseURL = %q, want default %q", cfg.DatabaseURL, DefaultDatabaseURL)
	}
	if cfg.HTTPAddr != DefaultHTTPAddr {
		t.Errorf("HTTPAddr = %q, want default %q", cfg.HTTPAddr, DefaultHTTPAddr)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want info", cfg.LogLevel)
	}
	if cfg.SessionIdle != 8*time.Hour {
		t.Errorf("SessionIdle = %v, want 8h default", cfg.SessionIdle)
	}
}

func TestLoadOverrides(t *testing.T) {
	env := map[string]string{
		"GEAR_DATABASE_URL": "postgres://u:p@db.example:5432/gear?sslmode=require",
		"GEAR_HTTP_ADDR":    "127.0.0.1:9999",
		"GEAR_LOG_LEVEL":    "debug",
		"GEAR_SESSION_IDLE": "30m",
	}
	cfg := Load(mapEnv(env))

	if cfg.DatabaseURL != env["GEAR_DATABASE_URL"] {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, env["GEAR_DATABASE_URL"])
	}
	if cfg.HTTPAddr != env["GEAR_HTTP_ADDR"] {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, env["GEAR_HTTP_ADDR"])
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want debug", cfg.LogLevel)
	}
	if cfg.SessionIdle != 30*time.Minute {
		t.Errorf("SessionIdle = %v, want 30m", cfg.SessionIdle)
	}
}

func TestLoadSessionIdleInvalidFallsBack(t *testing.T) {
	cfg := Load(mapEnv(map[string]string{"GEAR_SESSION_IDLE": "not-a-duration"}))
	if cfg.SessionIdle != 8*time.Hour {
		t.Errorf("SessionIdle = %v, want 8h fallback", cfg.SessionIdle)
	}
}

func TestLoadSessionIdleInvalidWarns(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.MultiWriter(&buf), &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(prev)

	_ = Load(mapEnv(map[string]string{"GEAR_SESSION_IDLE": "not-a-duration"}))

	if !strings.Contains(buf.String(), "invalid duration configured") {
		t.Errorf("expected a warning for invalid GEAR_SESSION_IDLE, got output %q", buf.String())
	}
}
