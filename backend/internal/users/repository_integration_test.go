//go:build integration

package users

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"repforge.local/backend/internal/platform/config"
	"repforge.local/backend/internal/platform/database"
	"repforge.local/backend/internal/platform/httpx"
	"repforge.local/backend/internal/platform/migrations"
	userdb "repforge.local/backend/internal/users/db"
)

func TestPostgresRepositoryOwnershipConflictAndMigrationRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	databaseConfig, cleanup := createDisposableDatabase(t, ctx)
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
	queries := userdb.New(pool)
	userID, _ := uuid.NewV7()
	identityID, _ := uuid.NewV7()
	consentID, _ := uuid.NewV7()
	if err := queries.CreateUser(ctx, userID); err != nil {
		t.Fatal(err)
	}
	if err := queries.CreateIdentityRef(ctx, userdb.CreateIdentityRefParams{ID: identityID, UserID: userID, Provider: "test", Subject: "owner"}); err != nil {
		t.Fatal(err)
	}
	if err := queries.CreateProfile(ctx, userdb.CreateProfileParams{UserID: userID, DisplayName: "Owner"}); err != nil {
		t.Fatal(err)
	}
	if err := queries.CreateConsent(ctx, userdb.CreateConsentParams{
		ID: consentID, UserID: userID, DocumentKey: "test_notice", DocumentVersion: "1",
		Source: "synthetic_local", AcceptedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	otherUserID, _ := uuid.NewV7()
	otherIdentityID, _ := uuid.NewV7()
	if err := queries.CreateUser(ctx, otherUserID); err != nil {
		t.Fatal(err)
	}
	if err := queries.CreateIdentityRef(ctx, userdb.CreateIdentityRefParams{
		ID: otherIdentityID, UserID: otherUserID, Provider: "test", Subject: "other",
	}); err != nil {
		t.Fatal(err)
	}
	disabledUserID, _ := uuid.NewV7()
	disabledIdentityID, _ := uuid.NewV7()
	if err := queries.CreateUser(ctx, disabledUserID); err != nil {
		t.Fatal(err)
	}
	if err := queries.CreateIdentityRef(ctx, userdb.CreateIdentityRefParams{
		ID: disabledIdentityID, UserID: disabledUserID, Provider: "test", Subject: "disabled",
	}); err != nil {
		t.Fatal(err)
	}
	if err := queries.CreateProfile(ctx, userdb.CreateProfileParams{UserID: disabledUserID, DisplayName: "Disabled"}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET status = $1 WHERE id = $2", "disabled", disabledUserID); err != nil {
		t.Fatal(err)
	}

	profile, err := repository.Get(ctx, Identity{Provider: "test", Subject: "owner"})
	if err != nil || profile.DisplayName != "Owner" || len(profile.Consents) != 1 {
		t.Fatalf("owner read: profile=%+v error=%v", profile, err)
	}
	if _, err := repository.Get(ctx, Identity{Provider: "test", Subject: "other"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-owner read error = %v, want not found", err)
	}
	for testName, identity := range map[string]Identity{
		"another identity without a profile": {Provider: "test", Subject: "other"},
		"missing identity":                   {Provider: "test", Subject: "missing"},
		"wrong provider":                     {Provider: "other-provider", Subject: "owner"},
		"disabled user":                      {Provider: "test", Subject: "disabled"},
	} {
		t.Run(testName, func(t *testing.T) {
			before, err := repository.Get(ctx, Identity{Provider: "test", Subject: "owner"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := repository.Update(ctx, identity, Update{DisplayName: "Unauthorized change", ExpectedVersion: 1}); !errors.Is(err, ErrNotFound) {
				t.Fatalf("update error = %v, want not found", err)
			}
			after, err := repository.Get(ctx, Identity{Provider: "test", Subject: "owner"})
			if err != nil {
				t.Fatal(err)
			}
			if after.DisplayName != before.DisplayName || after.Version != before.Version || after.UpdatedAt != before.UpdatedAt {
				t.Fatalf("owner changed after denied update: before=%+v after=%+v", before, after)
			}
		})
	}
	var disabledName string
	var disabledVersion int64
	if err := pool.QueryRow(ctx, "SELECT display_name, version FROM user_profiles WHERE user_id = $1", disabledUserID).Scan(&disabledName, &disabledVersion); err != nil {
		t.Fatal(err)
	}
	if disabledName != "Disabled" || disabledVersion != 1 {
		t.Fatalf("disabled profile changed: display_name=%q version=%d", disabledName, disabledVersion)
	}
	updated, err := repository.Update(ctx, Identity{Provider: "test", Subject: "owner"}, Update{DisplayName: "Updated", ExpectedVersion: 1})
	if err != nil || updated.Version != 2 {
		t.Fatalf("update: profile=%+v error=%v", updated, err)
	}
	if _, err := repository.Update(ctx, Identity{Provider: "test", Subject: "owner"}, Update{DisplayName: "Stale", ExpectedVersion: 1}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale update error = %v, want conflict", err)
	}
	pool.Close()
	if err := migrations.RunConfig(databaseConfig, migrationDirectory, "down"); err != nil {
		t.Fatalf("migration down: %v", err)
	}
	if err := migrations.RunConfig(databaseConfig, migrationDirectory, "up"); err != nil {
		t.Fatalf("second migration up: %v", err)
	}
}

type commitBarrierStarter struct {
	delegate  transactionStarter
	committed chan struct{}
	release   chan struct{}
}

func (starter *commitBarrierStarter) BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	tx, err := starter.delegate.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	return &commitBarrierTx{Tx: tx, committed: starter.committed, release: starter.release}, nil
}

type commitBarrierTx struct {
	pgx.Tx
	committed chan struct{}
	release   chan struct{}
}

type snapshotBarrierStarter struct {
	delegate transactionStarter
	ready    chan struct{}
	release  chan struct{}

	mu                  sync.Mutex
	attempts            int
	initialTransactions int
	snapshots           int
}

func (starter *snapshotBarrierStarter) BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	tx, err := starter.delegate.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	starter.mu.Lock()
	starter.attempts++
	participatesInBarrier := starter.initialTransactions < 2
	if participatesInBarrier {
		starter.initialTransactions++
	}
	starter.mu.Unlock()
	if !participatesInBarrier {
		return tx, nil
	}

	var profileCount int64
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM user_profiles").Scan(&profileCount); err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("establish profile update snapshot: %w", err)
	}
	starter.mu.Lock()
	starter.snapshots++
	if starter.snapshots == 2 {
		close(starter.ready)
	}
	starter.mu.Unlock()

	select {
	case <-starter.release:
		return tx, nil
	case <-ctx.Done():
		_ = tx.Rollback(ctx)
		return nil, ctx.Err()
	}
}

func (starter *snapshotBarrierStarter) attemptCount() int {
	starter.mu.Lock()
	defer starter.mu.Unlock()
	return starter.attempts
}

func (tx *commitBarrierTx) Commit(ctx context.Context) error {
	if err := tx.Tx.Commit(ctx); err != nil {
		return err
	}
	close(tx.committed)
	select {
	case <-tx.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestPostgresRepositoryUpdateResponseRemainsAuthoritativeAfterLaterMutation(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		mutate func(*testing.T, context.Context, *pgxpool.Pool, *PostgresRepository, uuid.UUID)
	}{
		{
			name: "later update",
			mutate: func(t *testing.T, ctx context.Context, _ *pgxpool.Pool, repository *PostgresRepository, _ uuid.UUID) {
				later, err := repository.Update(ctx, Identity{Provider: "test", Subject: "owner"}, Update{
					DisplayName: "Later update", ExpectedVersion: 2,
				})
				if err != nil || later.DisplayName != "Later update" || later.Version != 3 {
					t.Fatalf("later update: profile=%+v error=%v", later, err)
				}
			},
		},
		{
			name: "later disable",
			mutate: func(t *testing.T, ctx context.Context, pool *pgxpool.Pool, _ *PostgresRepository, userID uuid.UUID) {
				if _, err := pool.Exec(ctx, "UPDATE users SET status = $1 WHERE id = $2", "disabled", userID); err != nil {
					t.Fatalf("disable user: %v", err)
				}
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			databaseConfig, cleanup := createDisposableDatabase(t, ctx)
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
			defer pool.Close()
			queries := userdb.New(pool)
			userID, _ := uuid.NewV7()
			identityID, _ := uuid.NewV7()
			consentID, _ := uuid.NewV7()
			if err := queries.CreateUser(ctx, userID); err != nil {
				t.Fatal(err)
			}
			if err := queries.CreateIdentityRef(ctx, userdb.CreateIdentityRefParams{
				ID: identityID, UserID: userID, Provider: "test", Subject: "owner",
			}); err != nil {
				t.Fatal(err)
			}
			if err := queries.CreateProfile(ctx, userdb.CreateProfileParams{UserID: userID, DisplayName: "Original"}); err != nil {
				t.Fatal(err)
			}
			if err := queries.CreateConsent(ctx, userdb.CreateConsentParams{
				ID: consentID, UserID: userID, DocumentKey: "test_notice", DocumentVersion: "1",
				Source: "synthetic_local", AcceptedAt: pgtype.Timestamptz{
					Time: time.Date(2026, time.August, 29, 0, 0, 0, 0, time.UTC), Valid: true,
				},
			}); err != nil {
				t.Fatal(err)
			}

			committed := make(chan struct{})
			release := make(chan struct{})
			released := false
			defer func() {
				if !released {
					close(release)
				}
			}()
			firstRepository := &PostgresRepository{
				queries: userdb.New(pool),
				transactions: &commitBarrierStarter{
					delegate: pool, committed: committed, release: release,
				},
			}
			laterRepository := NewPostgresRepository(pool)
			type updateResult struct {
				profile Profile
				err     error
			}
			result := make(chan updateResult, 1)
			go func() {
				profile, updateErr := firstRepository.Update(ctx, Identity{Provider: "test", Subject: "owner"}, Update{
					DisplayName: "Committed response", ExpectedVersion: 1,
				})
				result <- updateResult{profile: profile, err: updateErr}
			}()

			select {
			case <-committed:
			case <-ctx.Done():
				t.Fatalf("first update did not reach the post-commit barrier: %v", ctx.Err())
			}
			testCase.mutate(t, ctx, pool, laterRepository, userID)
			close(release)
			released = true
			var first updateResult
			select {
			case first = <-result:
			case <-ctx.Done():
				t.Fatalf("first update did not return after barrier release: %v", ctx.Err())
			}
			if first.err != nil {
				t.Fatalf("first update: %v", first.err)
			}
			if first.profile.DisplayName != "Committed response" || first.profile.Version != 2 {
				t.Fatalf("later mutation replaced committed response: %+v", first.profile)
			}
			if len(first.profile.Consents) != 1 || first.profile.Consents[0].DocumentKey != "test_notice" {
				t.Fatalf("committed response has incoherent consents: %+v", first.profile.Consents)
			}
		})
	}
}

func TestConcurrentSameVersionPatchReturnsConflictAfterSerializationRetry(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	databaseConfig, cleanup := createDisposableDatabase(t, ctx)
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
	defer pool.Close()
	queries := userdb.New(pool)
	userID, _ := uuid.NewV7()
	identityID, _ := uuid.NewV7()
	if err := queries.CreateUser(ctx, userID); err != nil {
		t.Fatal(err)
	}
	if err := queries.CreateIdentityRef(ctx, userdb.CreateIdentityRefParams{
		ID: identityID, UserID: userID, Provider: "repforge-dev", Subject: "local-user",
	}); err != nil {
		t.Fatal(err)
	}
	if err := queries.CreateProfile(ctx, userdb.CreateProfileParams{UserID: userID, DisplayName: "Original"}); err != nil {
		t.Fatal(err)
	}

	ready := make(chan struct{})
	release := make(chan struct{})
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	transactions := &snapshotBarrierStarter{delegate: pool, ready: ready, release: release}
	repository := &PostgresRepository{queries: userdb.New(pool), transactions: transactions}
	handler := protectedHandler(NewHandler(NewService(repository)).Update)
	type patchResult struct {
		displayName string
		response    *httptest.ResponseRecorder
	}
	results := make(chan patchResult, 2)
	start := make(chan struct{})
	for _, displayName := range []string{"First contender", "Second contender"} {
		displayName := displayName
		go func() {
			<-start
			request := httptest.NewRequest(
				http.MethodPatch,
				"/v1/me",
				strings.NewReader(fmt.Sprintf(`{"displayName":%q,"expectedVersion":1}`, displayName)),
			)
			request.Header.Set("Authorization", "Bearer configured-local-token")
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			results <- patchResult{displayName: displayName, response: response}
		}()
	}
	close(start)
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatalf("concurrent PATCH transactions did not both establish pre-commit snapshots: %v", ctx.Err())
	}
	close(release)
	released = true

	var succeeded *patchResult
	var conflicted *patchResult
	for range 2 {
		select {
		case result := <-results:
			switch result.response.Code {
			case http.StatusOK:
				result := result
				succeeded = &result
			case http.StatusConflict:
				result := result
				conflicted = &result
			default:
				t.Fatalf("PATCH %q status = %d; body=%s", result.displayName, result.response.Code, result.response.Body.String())
			}
		case <-ctx.Done():
			t.Fatalf("concurrent PATCH operations did not complete: %v", ctx.Err())
		}
	}
	if succeeded == nil || conflicted == nil {
		t.Fatalf("want exactly one success and one conflict; success=%v conflict=%v", succeeded != nil, conflicted != nil)
	}
	if attempts := transactions.attemptCount(); attempts != 3 {
		t.Fatalf("transaction attempts = %d, want two overlapping attempts plus one bounded retry", attempts)
	}
	var conflictEnvelope httpx.ErrorEnvelope
	if err := json.Unmarshal(conflicted.response.Body.Bytes(), &conflictEnvelope); err != nil {
		t.Fatalf("decode conflict response: %v", err)
	}
	if conflictEnvelope.Error.Code != "version_conflict" || conflictEnvelope.Error.Message != "The profile changed. Refresh and try again." {
		t.Fatalf("unexpected conflict response: %+v", conflictEnvelope.Error)
	}
	var successBody meResponse
	if err := json.Unmarshal(succeeded.response.Body.Bytes(), &successBody); err != nil {
		t.Fatalf("decode success response: %v", err)
	}
	if successBody.Profile.DisplayName != succeeded.displayName || successBody.Profile.Version != 2 {
		t.Fatalf("unexpected successful PATCH response: %+v", successBody.Profile)
	}
	var finalName string
	var finalVersion int64
	if err := pool.QueryRow(ctx, "SELECT display_name, version FROM user_profiles WHERE user_id = $1", userID).Scan(&finalName, &finalVersion); err != nil {
		t.Fatalf("read final profile: %v", err)
	}
	if finalName != succeeded.displayName || finalVersion != 2 {
		t.Fatalf("final profile = (%q, %d), want successful value %q at version 2", finalName, finalVersion, succeeded.displayName)
	}
}

func createDisposableDatabase(t *testing.T, ctx context.Context) (*pgx.ConnConfig, func()) {
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
		admin.Close(ctx)
		t.Fatalf("create disposable database: %v", err)
	}
	admin.Close(ctx)
	testConfig := baseConfig.Copy()
	testConfig.Database = databaseName
	if err := config.RevalidateDatabaseConfig(testConfig); err != nil {
		t.Fatalf("refusing disposable integration database operation: %v", err)
	}
	cleanup := func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if validationErr := config.RevalidateDatabaseConfig(adminConfig); validationErr != nil {
			t.Errorf("refusing integration database cleanup: %v", validationErr)
			return
		}
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
