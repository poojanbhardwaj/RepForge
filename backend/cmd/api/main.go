package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"repforge.local/backend/internal/auth"
	"repforge.local/backend/internal/onboarding"
	apiserver "repforge.local/backend/internal/platform/api"
	"repforge.local/backend/internal/platform/config"
	"repforge.local/backend/internal/platform/database"
	"repforge.local/backend/internal/platform/httpx"
	"repforge.local/backend/internal/platform/observability"
	"repforge.local/backend/internal/users"
)

var (
	version = "dev"
	commit  = "uncommitted"
	builtAt = "1970-01-01T00:00:00Z"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "repforge-api:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration: %w", err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	repository := users.NewPostgresRepository(pool)
	userHandler := users.NewHandler(users.NewService(repository))
	authenticator, clientSources, err := buildAuthenticator(cfg)
	if err != nil {
		return err
	}
	onboardingRepository := onboarding.NewPostgresRepository(pool)
	onboardingHandler := onboarding.NewHandler(
		onboarding.NewService(onboardingRepository, cfg.TermsVersion, cfg.PrivacyVersion),
		clientSources,
	)
	mux := apiserver.NewRouter(apiserver.Options{
		Readiness: pool, ReadinessTimeout: cfg.ReadinessTimeout, Users: userHandler,
		Onboarding: onboardingHandler, Authenticator: authenticator,
		CORSAllowedOrigin: cfg.CORSAllowedOrigin,
		Build:             apiserver.BuildInfo{Version: version, Commit: commit, BuiltAt: builtAt},
	})

	handler := newHTTPHandler(logger, mux)
	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	serverError := make(chan error, 1)
	go func() {
		logger.Info("api_started", "address", cfg.HTTPAddress, "version", version)
		serverError <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		logger.Info("api_stopped")
		return nil
	case err := <-serverError:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	}
}

func buildAuthenticator(cfg config.Config) (auth.Authenticator, map[string]string, error) {
	if cfg.AuthMode == "dev" {
		return auth.NewDevAuthenticator(cfg.DevAuthToken, cfg.DevAuthProvider, cfg.DevAuthSubject),
			map[string]string{"repforge-dev-client": "synthetic_local"}, nil
	}
	authenticator, err := auth.NewOIDCAuthenticator(auth.OIDCOptions{
		Issuer: cfg.OIDCIssuer, Audience: cfg.OIDCAudience, AllowedClientIDs: cfg.OIDCClientIDs,
		ClockSkew: 30 * time.Second,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("OIDC authenticator: %w", err)
	}
	return authenticator, map[string]string{
		cfg.OIDCClientIDs[0]: "mobile",
		cfg.OIDCClientIDs[1]: "web",
	}, nil
}

func newHTTPHandler(logger *slog.Logger, mux http.Handler) http.Handler {
	return httpx.RequestIDMiddleware(httpx.Recovery(logger,
		httpx.SecurityHeaders(observability.HTTP(httpx.AccessLog(logger, mux)))))
}
