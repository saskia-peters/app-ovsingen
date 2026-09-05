// Command server is the G.E.A.R. composition root (AD-1): the only place that
// wires module hexagons and their adapters together and mounts the HTTP
// surface. No business logic lives here — handlers, adapters and repositories
// delegate to the modules.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saskia-peters/gear/internal/platform/config"
	"github.com/saskia-peters/gear/internal/platform/logger"
	"github.com/saskia-peters/gear/internal/platform/router"
	userpostgres "github.com/saskia-peters/gear/internal/user/adapters/postgres"
)

func main() {
	cfg := config.Load(os.Getenv)
	log := logger.New(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("invalid database configuration", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// AD-1: adapters are constructed here and handed to the hexagons. The
	// User repository adapter is wired now; its service and handlers land
	// with stories 1.3-1.10.
	userStore := userpostgres.New(pool)
	log.Info("wired user repository adapter", "store", fmt.Sprintf("%T", userStore))

	r := router.New(pool, log)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("graceful shutdown failed", "error", err)
		}
	}()

	log.Info("server listening", "addr", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server failed", "error", err)
		os.Exit(1)
	}
	log.Info("server stopped")
}
