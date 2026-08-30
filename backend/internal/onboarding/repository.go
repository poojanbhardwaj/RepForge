package onboarding

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	onboardingdb "repforge.local/backend/internal/onboarding/db"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool, now: time.Now}
}

func (repository *PostgresRepository) Get(ctx context.Context, identity Identity, termsVersion, privacyVersion string) (State, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return State{}, fmt.Errorf("begin onboarding read: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := onboardingdb.New(tx)
	if _, err := repository.ensureIdentity(ctx, queries, identity); err != nil {
		return State{}, err
	}
	row, err := queries.GetOnboardingByIdentity(ctx, onboardingdb.GetOnboardingByIdentityParams{
		Provider: identity.Provider, Subject: identity.Subject,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return State{}, ErrNotFound
	}
	if err != nil {
		return State{}, fmt.Errorf("get onboarding: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return State{}, fmt.Errorf("commit onboarding read: %w", err)
	}
	return present(row, termsVersion, privacyVersion), nil
}

func (repository *PostgresRepository) Update(ctx context.Context, identity Identity, idempotencyKey string, update Update, termsVersion, privacyVersion string) (State, error) {
	requestJSON, err := json.Marshal(update)
	if err != nil {
		return State{}, fmt.Errorf("encode onboarding request: %w", err)
	}
	hashInput := append(requestJSON, []byte("\x1f"+update.Source+"\x1f"+termsVersion+"\x1f"+privacyVersion)...)
	requestHash := sha256.Sum256(hashInput)
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return State{}, fmt.Errorf("begin onboarding update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := onboardingdb.New(tx)
	userID, err := repository.ensureIdentity(ctx, queries, identity)
	if err != nil {
		return State{}, err
	}
	row, err := queries.LockOnboarding(ctx, userID)
	if err != nil {
		return State{}, fmt.Errorf("lock onboarding: %w", err)
	}
	stored, err := queries.GetIdempotencyRecord(ctx, onboardingdb.GetIdempotencyRecordParams{
		UserID: userID, IdempotencyKey: idempotencyKey,
	})
	if err == nil {
		if !bytes.Equal(stored.RequestHash, requestHash[:]) {
			return State{}, ErrIdempotencyConflict
		}
		var state State
		if err := json.Unmarshal(stored.ResponseSnapshot, &state); err != nil {
			return State{}, fmt.Errorf("decode onboarding idempotency response: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return State{}, fmt.Errorf("commit onboarding replay: %w", err)
		}
		return state, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return State{}, fmt.Errorf("read onboarding idempotency key: %w", err)
	}

	now := repository.now().UTC()
	if update.AdultAttested != nil && !row.AdultAttestedAt.Valid {
		row.AdultAttestedAt = timestamp(now)
	}
	if update.TermsVersion != nil && (row.TermsVersion == nil || *row.TermsVersion != termsVersion) {
		row.TermsVersion = stringPointer(termsVersion)
		row.TermsAcceptedAt = timestamp(now)
		if err := createConsent(ctx, queries, userID, "terms", termsVersion, update.Source, now); err != nil {
			return State{}, err
		}
	}
	if update.PrivacyVersion != nil && (row.PrivacyVersion == nil || *row.PrivacyVersion != privacyVersion) {
		row.PrivacyVersion = stringPointer(privacyVersion)
		row.PrivacyAcceptedAt = timestamp(now)
		if err := createConsent(ctx, queries, userID, "privacy", privacyVersion, update.Source, now); err != nil {
			return State{}, err
		}
	}
	if update.Timezone != nil {
		row.Timezone = update.Timezone
	}
	if update.Units != nil {
		row.Units = update.Units
	}
	if update.PrimaryGoal != nil {
		row.PrimaryGoal = update.PrimaryGoal
	}
	if update.ExperienceLevel != nil {
		row.ExperienceLevel = update.ExperienceLevel
	}
	if update.WeeklyAvailability != nil {
		row.WeeklyAvailability = update.WeeklyAvailability
	}
	if update.SessionDurationMinutes != nil {
		row.SessionDurationMinutes = update.SessionDurationMinutes
	}
	if update.EquipmentAccess != nil {
		row.EquipmentAccess = update.EquipmentAccess
	}
	if update.ClearDietPreference {
		row.DietPreference = nil
	} else if update.DietPreference != nil {
		row.DietPreference = update.DietPreference
	}
	if update.SafetyAcknowledged != nil && !row.SafetyAcknowledgedAt.Valid {
		row.SafetyAcknowledgedAt = timestamp(now)
	}
	if update.CurrentStep != nil && *update.CurrentStep > row.CurrentStep {
		row.CurrentStep = *update.CurrentStep
	}
	completed := isComplete(row, termsVersion, privacyVersion)
	stateValue := "in_progress"
	completedAt := pgtype.Timestamptz{}
	if completed {
		stateValue = "complete"
		if row.CompletedAt.Valid {
			completedAt = row.CompletedAt
		} else {
			completedAt = timestamp(now)
		}
	}
	updated, err := queries.UpdateOnboarding(ctx, onboardingdb.UpdateOnboardingParams{
		AdultAttestedAt: row.AdultAttestedAt, TermsVersion: row.TermsVersion, TermsAcceptedAt: row.TermsAcceptedAt,
		PrivacyVersion: row.PrivacyVersion, PrivacyAcceptedAt: row.PrivacyAcceptedAt, Timezone: row.Timezone,
		Units: row.Units, PrimaryGoal: row.PrimaryGoal, ExperienceLevel: row.ExperienceLevel,
		WeeklyAvailability: row.WeeklyAvailability, SessionDurationMinutes: row.SessionDurationMinutes,
		EquipmentAccess: row.EquipmentAccess, DietPreference: row.DietPreference,
		SafetyAcknowledgedAt: row.SafetyAcknowledgedAt, CurrentStep: row.CurrentStep, State: stateValue,
		CompletedAt: completedAt, UpdatedAt: timestamp(now), UserID: userID,
	})
	if err != nil {
		return State{}, fmt.Errorf("update onboarding: %w", err)
	}
	state := present(updated, termsVersion, privacyVersion)
	responseJSON, err := json.Marshal(state)
	if err != nil {
		return State{}, fmt.Errorf("encode onboarding response: %w", err)
	}
	if err := queries.CreateIdempotencyRecord(ctx, onboardingdb.CreateIdempotencyRecordParams{
		UserID: userID, IdempotencyKey: idempotencyKey, RequestHash: requestHash[:], ResponseSnapshot: responseJSON,
	}); err != nil {
		return State{}, fmt.Errorf("store onboarding idempotency response: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return State{}, fmt.Errorf("commit onboarding update: %w", err)
	}
	return state, nil
}

func (repository *PostgresRepository) ensureIdentity(ctx context.Context, queries *onboardingdb.Queries, identity Identity) (uuid.UUID, error) {
	provider := identity.Provider
	subject := identity.Subject
	if err := queries.LockOnboardingIdentity(ctx, onboardingdb.LockOnboardingIdentityParams{Provider: &provider, Subject: &subject}); err != nil {
		return uuid.Nil, fmt.Errorf("lock onboarding identity: %w", err)
	}
	identityUser, err := queries.GetIdentityUser(ctx, onboardingdb.GetIdentityUserParams{Provider: identity.Provider, Subject: identity.Subject})
	if errors.Is(err, pgx.ErrNoRows) {
		userID, idErr := uuid.NewV7()
		err = idErr
		if err != nil {
			return uuid.Nil, fmt.Errorf("generate onboarding user ID: %w", err)
		}
		identityID, idErr := uuid.NewV7()
		if idErr != nil {
			return uuid.Nil, fmt.Errorf("generate identity ID: %w", idErr)
		}
		if err := queries.CreateUser(ctx, userID); err != nil {
			return uuid.Nil, fmt.Errorf("create onboarding user: %w", err)
		}
		if err := queries.CreateIdentity(ctx, onboardingdb.CreateIdentityParams{ID: identityID, UserID: userID, Provider: identity.Provider, Subject: identity.Subject}); err != nil {
			return uuid.Nil, fmt.Errorf("create onboarding identity: %w", err)
		}
		identityUser.UserID = userID
		identityUser.Status = "active"
	} else if err != nil {
		return uuid.Nil, fmt.Errorf("find onboarding identity: %w", err)
	}
	if identityUser.Status != "active" {
		return uuid.Nil, ErrNotFound
	}
	userID := identityUser.UserID
	if err := queries.CreateDefaultProfile(ctx, userID); err != nil {
		return uuid.Nil, fmt.Errorf("create default profile: %w", err)
	}
	if err := queries.CreateOnboarding(ctx, userID); err != nil {
		return uuid.Nil, fmt.Errorf("create onboarding state: %w", err)
	}
	return userID, nil
}

func createConsent(ctx context.Context, queries *onboardingdb.Queries, userID uuid.UUID, documentKey, version, source string, acceptedAt time.Time) error {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate consent ID: %w", err)
	}
	if err := queries.CreateConsent(ctx, onboardingdb.CreateConsentParams{
		ID: id, UserID: userID, DocumentKey: documentKey, DocumentVersion: version,
		Source: source, AcceptedAt: timestamp(acceptedAt),
	}); err != nil {
		return fmt.Errorf("create onboarding consent: %w", err)
	}
	return nil
}

func isComplete(row onboardingdb.UserOnboarding, termsVersion, privacyVersion string) bool {
	return row.AdultAttestedAt.Valid && row.TermsAcceptedAt.Valid && row.PrivacyAcceptedAt.Valid &&
		row.TermsVersion != nil && *row.TermsVersion == termsVersion &&
		row.PrivacyVersion != nil && *row.PrivacyVersion == privacyVersion &&
		row.Timezone != nil && row.Units != nil && row.PrimaryGoal != nil && row.ExperienceLevel != nil &&
		row.WeeklyAvailability != nil && row.SessionDurationMinutes != nil && len(row.EquipmentAccess) > 0 &&
		row.SafetyAcknowledgedAt.Valid
}

func present(row onboardingdb.UserOnboarding, termsVersion, privacyVersion string) State {
	state := State{
		UserID: row.UserID, State: row.State, CurrentStep: row.CurrentStep, Version: row.Version,
		AdultAttestedAt: nullableTimestamp(row.AdultAttestedAt), TermsVersion: row.TermsVersion,
		TermsAcceptedAt: nullableTimestamp(row.TermsAcceptedAt), PrivacyVersion: row.PrivacyVersion,
		PrivacyAcceptedAt: nullableTimestamp(row.PrivacyAcceptedAt), Timezone: row.Timezone, Units: row.Units,
		PrimaryGoal: row.PrimaryGoal, ExperienceLevel: row.ExperienceLevel,
		WeeklyAvailability: row.WeeklyAvailability, SessionDurationMinutes: row.SessionDurationMinutes,
		EquipmentAccess: append([]string(nil), row.EquipmentAccess...), DietPreference: row.DietPreference,
		SafetyAcknowledgedAt: nullableTimestamp(row.SafetyAcknowledgedAt), CompletedAt: nullableTimestamp(row.CompletedAt),
		CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
		RequiredTermsVersion: termsVersion, RequiredPrivacyVersion: privacyVersion,
	}
	if !isComplete(row, termsVersion, privacyVersion) {
		state.State = "in_progress"
		state.CompletedAt = nil
	}
	return state
}

func timestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func nullableTimestamp(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time.UTC()
	return &result
}

func stringPointer(value string) *string { return &value }
