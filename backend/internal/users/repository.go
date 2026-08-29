package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	userdb "repforge.local/backend/internal/users/db"
)

const (
	maxProfileUpdateAttempts     = 2
	serializationFailureSQLState = "40001"
)

type Repository interface {
	Get(context.Context, Identity) (Profile, error)
	Update(context.Context, Identity, Update) (Profile, error)
}

type PostgresRepository struct {
	queries      *userdb.Queries
	transactions transactionStarter
}

type transactionStarter interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{queries: userdb.New(pool), transactions: pool}
}

func (r *PostgresRepository) Get(ctx context.Context, identity Identity) (Profile, error) {
	row, err := r.queries.GetProfileByIdentity(ctx, userdb.GetProfileByIdentityParams{
		Provider: identity.Provider,
		Subject:  identity.Subject,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("get profile: %w", err)
	}
	consents, err := r.queries.ListConsentsByUser(ctx, row.UserID)
	if err != nil {
		return Profile{}, fmt.Errorf("list consents: %w", err)
	}
	result := Profile{
		UserID: row.UserID, DisplayName: row.DisplayName, Version: row.Version,
		CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
		Consents: make([]Consent, 0, len(consents)),
	}
	for _, consent := range consents {
		result.Consents = append(result.Consents, Consent{
			ID: consent.ID, DocumentKey: consent.DocumentKey, DocumentVersion: consent.DocumentVersion,
			Source: consent.Source, AcceptedAt: consent.AcceptedAt.Time, WithdrawnAt: nullableTime(consent.WithdrawnAt),
		})
	}
	return result, nil
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func (r *PostgresRepository) Update(ctx context.Context, identity Identity, update Update) (Profile, error) {
	for attempt := 0; attempt < maxProfileUpdateAttempts; attempt++ {
		profile, err := r.updateOnce(ctx, identity, update)
		if err == nil {
			return profile, nil
		}
		if !isSerializationFailure(err) {
			return Profile{}, err
		}
	}
	return Profile{}, ErrConflict
}

func (r *PostgresRepository) updateOnce(ctx context.Context, identity Identity, update Update) (Profile, error) {
	tx, err := r.transactions.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return Profile{}, fmt.Errorf("begin profile update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := userdb.New(tx)
	row, err := queries.UpdateProfileByIdentity(ctx, userdb.UpdateProfileByIdentityParams{
		DisplayName: update.DisplayName, Provider: identity.Provider, Subject: identity.Subject,
		ExpectedVersion: update.ExpectedVersion,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		_, getErr := queries.GetProfileByIdentity(ctx, userdb.GetProfileByIdentityParams{
			Provider: identity.Provider,
			Subject:  identity.Subject,
		})
		if errors.Is(getErr, pgx.ErrNoRows) {
			return Profile{}, ErrNotFound
		} else if getErr != nil {
			return Profile{}, fmt.Errorf("classify profile update: %w", getErr)
		}
		return Profile{}, ErrConflict
	}
	if err != nil {
		return Profile{}, fmt.Errorf("update profile: %w", err)
	}
	consents, err := queries.ListConsentsByUser(ctx, row.UserID)
	if err != nil {
		return Profile{}, fmt.Errorf("list consents for profile update: %w", err)
	}
	result := Profile{
		UserID: row.UserID, DisplayName: row.DisplayName, Version: row.Version,
		CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
		Consents: make([]Consent, 0, len(consents)),
	}
	for _, consent := range consents {
		result.Consents = append(result.Consents, Consent{
			ID: consent.ID, DocumentKey: consent.DocumentKey, DocumentVersion: consent.DocumentVersion,
			Source: consent.Source, AcceptedAt: consent.AcceptedAt.Time, WithdrawnAt: nullableTime(consent.WithdrawnAt),
		})
	}
	if err := tx.Commit(ctx); err != nil {
		return Profile{}, fmt.Errorf("commit profile update: %w", err)
	}
	return result, nil
}

func isSerializationFailure(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == serializationFailureSQLState
}
