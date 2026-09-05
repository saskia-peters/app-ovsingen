package config

import (
	"log/slog"
	"testing"
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
}

func TestLoadOverrides(t *testing.T) {
	env := map[string]string{
		"GEAR_DATABASE_URL": "postgres://u:p@db.example:5432/gear?sslmode=require",
		"GEAR_HTTP_ADDR":    "127.0.0.1:9999",
		"GEAR_LOG_LEVEL":    "debug",
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
}