// Command api is the nutrition tracker's HTTP service.
//
// This file is the only place that knows how the pieces connect. Everything else
// receives its dependencies as parameters — see docs/01-architecture.md.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nfriesen1/nutrition-tracker/api/internal/config"
	"github.com/nfriesen1/nutrition-tracker/api/internal/httpx"
)

// version is set at build time with -ldflags "-X main.version=$(git rev-parse --short HEAD)".
// It is reported by /healthz, so it is possible to tell what is actually running.
var version = "dev"

func main() {
	// Cancelled on SIGINT or SIGTERM. Fly sends SIGTERM when it replaces a machine.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("startup failed", "error", err)
		os.Exit(1)
	}
}

// run serves until ctx is cancelled, then drains and returns.
//
// It takes the cancellation source as a parameter rather than installing the signal
// handler itself, for two reasons: every path returns an error instead of calling
// os.Exit, which would skip deferred cleanup; and the shutdown path becomes testable
// without delivering a real signal — which matters because Windows only delivers
// os.Interrupt through console events, so a signal-based test cannot run on a dev
// machine here even though production is Linux.
func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	setupLogging(cfg)

	srv := &http.Server{
		// 0.0.0.0, not 127.0.0.1. Fly routes to the container's external interface, and
		// in development a phone on the LAN needs to reach this too.
		Addr:    net.JoinHostPort("0.0.0.0", cfg.Port),
		Handler: httpx.NewServer(cfg, version).Routes(),

		// Without these a single stalled client holds a connection open forever.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("listening",
			"addr", srv.Addr, "env", cfg.Env, "version", version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err

	case <-ctx.Done():
		slog.Info("shutdown signal received, draining")

		// Stop listening, let in-flight requests finish, then exit. Without this the
		// process dies mid-request and the client sees a connection reset.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			// Deadline hit with requests still running. Close is the blunt fallback.
			slog.Error("graceful shutdown failed, closing", "error", err)
			return srv.Close()
		}

		slog.Info("shutdown complete")
		return nil
	}
}

func setupLogging(cfg config.Config) {
	var handler slog.Handler
	if cfg.IsDevelopment() {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		// JSON in production so Fly's log search can filter on fields.
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	slog.SetDefault(slog.New(handler))
}
