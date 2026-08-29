package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"repforge.local/backend/internal/platform/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "repforge-worker: configuration:", err)
		os.Exit(1)
	}
	if !cfg.WorkerIdleAllowed {
		fmt.Fprintln(os.Stderr, "repforge-worker: no job handlers are registered; idle mode is allowed only when explicitly configured for local/test")
		os.Exit(1)
	}
	if cfg.Environment != config.EnvironmentLocal && cfg.Environment != config.EnvironmentTest {
		fmt.Fprintln(os.Stderr, "repforge-worker: idle worker is forbidden outside local/test")
		os.Exit(1)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger.Info("worker_started", "mode", "idle_local_test")
	<-ctx.Done()
	if !errors.Is(ctx.Err(), context.Canceled) {
		logger.Error("worker_stopped", "reason", "unexpected_context_error")
		os.Exit(1)
	}
	logger.Info("worker_stopped")
}
