// Package config loads runtime configuration from the environment using the
// GEAR_-prefixed variables:
//
//	GEAR_DATABASE_URL  PostgreSQL DSN for the pgx pool
//	GEAR_HTTP_ADDR     listen address for the HTTP server
//	GEAR_LOG_LEVEL     debug | info | warn | error
//
// Local-development defaults match the root compose.yaml and justfile so a
// fresh checkout needs no configuration. Secrets (admin bootstrap credentials,
// DB passwords in production) are supplied at runtime via the environment and
// never committed (NFR-S4 / AD-13).
package config

import (
	"log/slog"

	"github.com/saskia-peters/gear/internal/platform/logger"
)

// Defaults for local development; they mirror compose.yaml and the justfile.
const (
	DefaultDatabaseURL = "postgres://gear:gear@localhost:5432/gear?sslmode=disable"
	DefaultHTTPAddr    = ":8080"
	DefaultLogLevel    = "info"
)

// Config holds the runtime configuration of the server.
type Config struct {
	DatabaseURL string
	HTTPAddr    string
	LogLevel    slog.Level
}

// Load reads the configuration through getenv (os.Getenv in production).
func Load(getenv func(string) string) Config {
	return Config{
		DatabaseURL: envOr(getenv, "GEAR_DATABASE_URL", DefaultDatabaseURL),
		HTTPAddr:    envOr(getenv, "GEAR_HTTP_ADDR", DefaultHTTPAddr),
		LogLevel:    logger.ParseLevel(envOr(getenv, "GEAR_LOG_LEVEL", DefaultLogLevel)),
	}
}

func envOr(getenv func(string) string, key, fallback string) string {
	if v := getenv(key); v != "" {
		return v
	}
	return fallback
}