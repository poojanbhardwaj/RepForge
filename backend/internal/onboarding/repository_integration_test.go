//go:build integration

package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"repforge.local/backend/internal/platform/config"
	"repforge.local/backend/internal/platform/database"
	"repforge.local/backend/internal/platform/migrations"
)

func TestPostgresRepositoryProvisionResumeIdempotencyOwnershipAndMigrationRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	databaseConfig, cleanup := createDisposableOnboardingDatabase(t, ctx)
	defer cleanup()
	_, currentFile, _, _ := runtime.Caller(0)
	migrationDirectory := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations"))
	if err := migrations.RunConfig(databaseConfig, migrationDirectory, "up"); err != nil {
		t.Fatalf("migration up: %v", err)
	}
	pool, err := database.OpenConfig(ctx, databaseConfig)
	if err != nil {
		t.Fatal(err)
	}
	repository := NewPostgresRepository(pool)
	fixedNow := time.Date(2026, time.August, 30, 1, 2, 3, 0, time.UTC)
	repository.now = func() time.Time { return fixedNow }
	service := NewService(repository, "terms-1", "privacy-1")

	t.Run("concurrent first login provisions exactly one owned graph", func(t *testing.T) {
		identity := Identity{Provider: "https://issuer.example/", Subject: "concurrent-user"}
		const callers = 8
		results := make(chan State, callers)
		errorsChannel := make(chan error, callers)
		start := make(chan struct{})
		var wait sync.WaitGroup
		for range callers {
			wait.Add(1)
			go func() {
				defer wait.Done()
				<-start
				state, getErr := service.Get(ctx, identity)
				results <- state
				errorsChannel <- getErr
			}()
		}
		close(start)
		wait.Wait()
		close(results)
		close(errorsChannel)
		for getErr := range errorsChannel {
			if getErr != nil {
				t.Fatalf("concurrent get: %v", getErr)
			}
		}
		var userID uuid.UUID
		for state := range results {
			if userID == uuid.Nil {
				userID = state.UserID
			}
			if state.UserID != userID || state.State != "in_progress" || state.Version != 1 {
				t.Fatalf("incoherent provisioned state: %+v", state)
			}
		}
		for table, query := range map[string]string{
			"user":       "SELECT count(*) FROM users u JOIN identity_refs i ON i.user_id = u.id WHERE i.provider = $1 AND i.subject = $2",
			"identity":   "SELECT count(*) FROM identity_refs WHERE provider = $1 AND subject = $2",
			"profile":    "SELECT count(*) FROM user_profiles p JOIN identity_refs i ON i.user_id = p.user_id WHERE i.provider = $1 AND i.subject = $2",
			"onboarding": "SELECT count(*) FROM user_onboarding o JOIN identity_refs i ON i.user_id = o.user_id WHERE i.provider = $1 AND i.subject = $2",
		} {
			var count int
			if err := pool.QueryRow(ctx, query, identity.Provider, identity.Subject).Scan(&count); err != nil || count != 1 {
				t.Fatalf("%s count=%d error=%v", table, count, err)
			}
		}
	})

	t.Run("concurrent same-key saves mutate once and replay an authoritative response", func(t *testing.T) {
		identity := Identity{Provider: "https://issuer.example/", Subject: "idempotent-user"}
		if _, err := service.Get(ctx, identity); err != nil {
			t.Fatal(err)
		}
		const callers = 6
		results := make(chan State, callers)
		errorsChannel := make(chan error, callers)
		start := make(chan struct{})
		var wait sync.WaitGroup
		for range callers {
			wait.Add(1)
			go func() {
				defer wait.Done()
				<-start
				step := int16(2)
				state, updateErr := service.Update(ctx, identity, "same-key-1234567890123456", Update{Source: "web", CurrentStep: &step})
				results <- state
				errorsChannel <- updateErr
			}()
		}
		close(start)
		wait.Wait()
		close(results)
		close(errorsChannel)
		for updateErr := range errorsChannel {
			if updateErr != nil {
				t.Fatalf("concurrent update: %v", updateErr)
			}
		}
		var canonical []byte
		for state := range results {
			encoded, _ := json.Marshal(state)
			if canonical == nil {
				canonical = encoded
			}
			if string(encoded) != string(canonical) || state.Version != 2 {
				t.Fatalf("non-authoritative replay: %s want %s", encoded, canonical)
			}
		}
		var version int64
		var records int
		if err := pool.QueryRow(ctx, "SELECT o.version FROM user_onboarding o JOIN identity_refs i ON i.user_id = o.user_id WHERE i.provider = $1 AND i.subject = $2", identity.Provider, identity.Subject).Scan(&version); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM onboarding_idempotency k JOIN identity_refs i ON i.user_id = k.user_id WHERE i.provider = $1 AND i.subject = $2", identity.Provider, identity.Subject).Scan(&records); err != nil {
			t.Fatal(err)
		}
		if version != 2 || records != 1 {
			t.Fatalf("version=%d records=%d", version, records)
		}
		step := int16(3)
		if _, err := service.Update(ctx, identity, "same-key-1234567890123456", Update{Source: "web", CurrentStep: &step}); !errors.Is(err, ErrIdempotencyConflict) {
			t.Fatalf("mismatched replay error=%v", err)
		}
	})

	t.Run("completion records versioned consents and legal changes re-gate access", func(t *testing.T) {
		identity := Identity{Provider: "https://issuer.example/", Subject: "complete-user"}
		if _, err := service.Get(ctx, identity); err != nil {
			t.Fatal(err)
		}
		completed, err := service.Update(ctx, identity, "complete-key-123456789012", completeOnboardingUpdate("terms-1", "privacy-1"))
		if err != nil || completed.State != "complete" || completed.CompletedAt == nil {
			t.Fatalf("complete state=%+v error=%v", completed, err)
		}
		var consentCount int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM consents c JOIN identity_refs i ON i.user_id = c.user_id WHERE i.provider = $1 AND i.subject = $2 AND c.source = 'web'", identity.Provider, identity.Subject).Scan(&consentCount); err != nil || consentCount != 2 {
			t.Fatalf("consent count=%d error=%v", consentCount, err)
		}

		updatedTermsService := NewService(repository, "terms-2", "privacy-1")
		regated, err := updatedTermsService.Get(ctx, identity)
		if err != nil || regated.State != "in_progress" || regated.CompletedAt != nil || regated.TermsVersion == nil || *regated.TermsVersion != "terms-1" {
			t.Fatalf("re-gated state=%+v error=%v", regated, err)
		}
		terms2 := "terms-2"
		if _, err := updatedTermsService.Update(ctx, identity, "complete-key-123456789012", Update{Source: "web", TermsVersion: &terms2}); !errors.Is(err, ErrIdempotencyConflict) {
			t.Fatalf("old key under new legal version error=%v", err)
		}
		reaccepted, err := updatedTermsService.Update(ctx, identity, "terms-two-key-1234567890", Update{Source: "web", TermsVersion: &terms2})
		if err != nil || reaccepted.State != "complete" || reaccepted.TermsVersion == nil || *reaccepted.TermsVersion != "terms-2" {
			t.Fatalf("reaccepted state=%+v error=%v", reaccepted, err)
		}
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM consents c JOIN identity_refs i ON i.user_id = c.user_id WHERE i.provider = $1 AND i.subject = $2", identity.Provider, identity.Subject).Scan(&consentCount); err != nil || consentCount != 3 {
			t.Fatalf("versioned consent count=%d error=%v", consentCount, err)
		}

		originalAdultAttestation := reaccepted.AdultAttestedAt
		originalSafetyAcknowledgement := reaccepted.SafetyAcknowledgedAt
		repository.now = func() time.Time { return fixedNow.Add(time.Hour) }
		cleared, err := updatedTermsService.Update(ctx, identity, "clear-diet-key-1234567890", Update{
			Source: "web", ClearDietPreference: true, AdultAttested: boolPointer(true), SafetyAcknowledged: boolPointer(true),
		})
		if err != nil || cleared.State != "complete" || cleared.DietPreference != nil {
			t.Fatalf("cleared optional diet state=%+v error=%v", cleared, err)
		}
		if !timesEqual(cleared.AdultAttestedAt, originalAdultAttestation) || !timesEqual(cleared.SafetyAcknowledgedAt, originalSafetyAcknowledgement) {
			t.Fatalf("repeat saves changed evidence timestamps: before=%v/%v after=%v/%v", originalAdultAttestation, originalSafetyAcknowledgement, cleared.AdultAttestedAt, cleared.SafetyAcknowledgedAt)
		}
		repository.now = func() time.Time { return fixedNow }
	})

	t.Run("identities cannot mutate another owner and disabled identities fail closed", func(t *testing.T) {
		owner := Identity{Provider: "https://issuer.example/", Subject: "owner"}
		other := Identity{Provider: "https://issuer.example/", Subject: "other"}
		ownerBefore, err := service.Get(ctx, owner)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.Get(ctx, other); err != nil {
			t.Fatal(err)
		}
		step := int16(4)
		otherState, err := service.Update(ctx, other, "other-owner-key-12345678", Update{Source: "mobile", CurrentStep: &step})
		if err != nil || otherState.UserID == ownerBefore.UserID {
			t.Fatalf("other update state=%+v error=%v", otherState, err)
		}
		ownerAfter, err := service.Get(ctx, owner)
		if err != nil || ownerAfter.UserID != ownerBefore.UserID || ownerAfter.Version != ownerBefore.Version {
			t.Fatalf("owner changed: before=%+v after=%+v error=%v", ownerBefore, ownerAfter, err)
		}

		disabledUser, _ := uuid.NewV7()
		disabledIdentity, _ := uuid.NewV7()
		if _, err := pool.Exec(ctx, "INSERT INTO users (id, status) VALUES ($1, 'disabled')", disabledUser); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, "INSERT INTO identity_refs (id, user_id, provider, subject) VALUES ($1, $2, $3, $4)", disabledIdentity, disabledUser, "https://issuer.example/", "disabled"); err != nil {
			t.Fatal(err)
		}
		if _, err := service.Get(ctx, Identity{Provider: "https://issuer.example/", Subject: "disabled"}); !errors.Is(err, ErrNotFound) {
			t.Fatalf("disabled get error=%v", err)
		}
		var onboardingCount int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM user_onboarding WHERE user_id = $1", disabledUser).Scan(&onboardingCount); err != nil || onboardingCount != 0 {
			t.Fatalf("disabled onboarding count=%d error=%v", onboardingCount, err)
		}
	})

	pool.Close()
	if err := migrations.RunConfig(databaseConfig, migrationDirectory, "down"); err != nil {
		t.Fatalf("migration down: %v", err)
	}
	if err := migrations.RunConfig(databaseConfig, migrationDirectory, "up"); err != nil {
		t.Fatalf("second migration up: %v", err)
	}
}

func completeOnboardingUpdate(termsVersion, privacyVersion string) Update {
	adult, safety := true, true
	timezone, units, goal := "Asia/Kolkata", "metric", "strength"
	experience, diet := "beginner", "vegetarian"
	weekly, duration, step := int16(4), int16(60), int16(6)
	return Update{
		Source: "web", AdultAttested: &adult, TermsVersion: &termsVersion, PrivacyVersion: &privacyVersion,
		Timezone: &timezone, Units: &units, PrimaryGoal: &goal, ExperienceLevel: &experience,
		WeeklyAvailability: &weekly, SessionDurationMinutes: &duration,
		EquipmentAccess: []string{"bodyweight", "dumbbells"}, DietPreference: &diet,
		SafetyAcknowledged: &safety, CurrentStep: &step,
	}
}

func timesEqual(left, right *time.Time) bool {
	return left != nil && right != nil && left.Equal(*right)
}

func createDisposableOnboardingDatabase(t *testing.T, ctx context.Context) (*pgx.ConnConfig, func()) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("refusing integration database operation: %v", err)
	}
	baseConfig, err := config.ParseDatabaseConfig(cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("refusing integration database operation: %v", err)
	}
	databaseName := "repforge_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	adminConfig := baseConfig.Copy()
	adminConfig.Database = "postgres"
	if err := config.RevalidateDatabaseConfig(adminConfig); err != nil {
		t.Fatalf("refusing integration admin database operation: %v", err)
	}
	admin, err := pgx.ConnectConfig(ctx, adminConfig)
	if err != nil {
		t.Fatalf("connect admin database: %v", err)
	}
	identifier := pgx.Identifier{databaseName}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+identifier); err != nil {
		_ = admin.Close(ctx)
		t.Fatalf("create disposable database: %v", err)
	}
	_ = admin.Close(ctx)
	testConfig := baseConfig.Copy()
	testConfig.Database = databaseName
	if err := config.RevalidateDatabaseConfig(testConfig); err != nil {
		t.Fatalf("refusing disposable integration database operation: %v", err)
	}
	cleanup := func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		connection, connectionErr := pgx.ConnectConfig(cleanupCtx, adminConfig)
		if connectionErr != nil {
			t.Errorf("cleanup connect: %v", connectionErr)
			return
		}
		defer connection.Close(cleanupCtx)
		_, _ = connection.Exec(cleanupCtx, "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1", databaseName)
		if _, dropErr := connection.Exec(cleanupCtx, "DROP DATABASE "+identifier); dropErr != nil {
			t.Errorf("drop disposable database %s: %v", fmt.Sprint(databaseName), dropErr)
		}
	}
	return testConfig, cleanup
}
