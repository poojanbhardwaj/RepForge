package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"repforge.local/backend/internal/platform/config"
	"repforge.local/backend/internal/platform/database"
	userdb "repforge.local/backend/internal/users/db"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "repforge-devseed:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Environment != config.EnvironmentLocal && cfg.Environment != config.EnvironmentTest {
		return errors.New("synthetic seed is forbidden outside local/test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := userdb.New(tx)
	identity := userdb.GetUserIDByIdentityParams{Provider: cfg.DevAuthProvider, Subject: cfg.DevAuthSubject}
	userID, err := queries.GetUserIDByIdentity(ctx, identity)
	if errors.Is(err, pgx.ErrNoRows) {
		userID, err = uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate user ID: %w", err)
		}
		identityID, idErr := uuid.NewV7()
		if idErr != nil {
			return fmt.Errorf("generate identity ID: %w", idErr)
		}
		if err := queries.CreateUser(ctx, userID); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		if err := queries.CreateIdentityRef(ctx, userdb.CreateIdentityRefParams{
			ID: identityID, UserID: userID, Provider: cfg.DevAuthProvider, Subject: cfg.DevAuthSubject,
		}); err != nil {
			return fmt.Errorf("create identity: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("find identity: %w", err)
	}
	if err := queries.CreateProfile(ctx, userdb.CreateProfileParams{UserID: userID, DisplayName: "Local Athlete"}); err != nil {
		return fmt.Errorf("create profile: %w", err)
	}
	consentID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate consent ID: %w", err)
	}
	if err := queries.CreateConsent(ctx, userdb.CreateConsentParams{
		ID: consentID, UserID: userID, DocumentKey: "bootstrap_data_notice",
		DocumentVersion: "draft-local-1", Source: "synthetic_local",
		AcceptedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}); err != nil {
		return fmt.Errorf("create consent: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}
	fmt.Println("Synthetic local profile is ready.")
	return nil
}
