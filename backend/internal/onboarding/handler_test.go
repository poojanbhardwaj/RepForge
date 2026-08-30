package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"repforge.local/backend/internal/auth"
	"repforge.local/backend/internal/platform/httpx"
)

type fakeRepository struct {
	state       State
	identity    Identity
	key         string
	update      Update
	err         error
	updateCalls int
}

func (repository *fakeRepository) Get(_ context.Context, identity Identity, _, _ string) (State, error) {
	repository.identity = identity
	return repository.state, repository.err
}

func (repository *fakeRepository) Update(_ context.Context, identity Identity, key string, update Update, _, _ string) (State, error) {
	repository.identity, repository.key, repository.update = identity, key, update
	repository.updateCalls++
	return repository.state, repository.err
}

type blockingRepository struct{ cancelled chan struct{} }

func (repository *blockingRepository) wait(ctx context.Context) (State, error) {
	<-ctx.Done()
	close(repository.cancelled)
	return State{}, ctx.Err()
}

func (repository *blockingRepository) Get(ctx context.Context, _ Identity, _, _ string) (State, error) {
	return repository.wait(ctx)
}

func (repository *blockingRepository) Update(ctx context.Context, _ Identity, _ string, _ Update, _, _ string) (State, error) {
	return repository.wait(ctx)
}

func onboardingHandler(repository Repository) *Handler {
	return NewHandler(NewService(repository, "terms-1", "privacy-1"), map[string]string{"repforge-dev-client": "synthetic_local"})
}

func protectOnboarding(handler http.HandlerFunc) http.Handler {
	unauthorized := func(response http.ResponseWriter, request *http.Request) {
		httpx.WriteError(response, request, http.StatusUnauthorized, "unauthenticated", "Authentication is required.", nil)
	}
	return httpx.RequestIDMiddleware(auth.Middleware(auth.NewDevAuthenticator("configured-local-token", "repforge-dev", "local-user"), unauthorized)(handler))
}

func TestOnboardingGetRequiresAuthenticationAndUsesPrincipalIdentity(t *testing.T) {
	repository := &fakeRepository{state: State{UserID: uuid.New(), State: "in_progress"}}
	handler := onboardingHandler(repository)
	request := httptest.NewRequest(http.MethodGet, "/v1/onboarding", nil)
	response := httptest.NewRecorder()
	protectOnboarding(handler.Get).ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/v1/onboarding", nil)
	request.Header.Set("Authorization", "Bearer configured-local-token")
	response = httptest.NewRecorder()
	protectOnboarding(handler.Get).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("authenticated status = %d; body=%s", response.Code, response.Body.String())
	}
	if repository.identity != (Identity{Provider: "repforge-dev", Subject: "local-user"}) {
		t.Fatalf("identity = %+v", repository.identity)
	}
}

func TestOnboardingUpdateDerivesConsentSourceAndRequiresOneIdempotencyHeader(t *testing.T) {
	for name, headers := range map[string][]string{
		"missing":   nil,
		"duplicate": {"1234567890123456", "abcdefghijklmnop"},
		"unsafe":    {"short key"},
	} {
		t.Run(name, func(t *testing.T) {
			repository := &fakeRepository{}
			request := httptest.NewRequest(http.MethodPatch, "/v1/onboarding", strings.NewReader(`{"adultAttested":true}`))
			request.Header.Set("Authorization", "Bearer configured-local-token")
			request.Header.Set("Content-Type", "application/json")
			for _, value := range headers {
				request.Header.Add("Idempotency-Key", value)
			}
			response := httptest.NewRecorder()
			protectOnboarding(onboardingHandler(repository).Update).ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest || repository.updateCalls != 0 {
				t.Fatalf("status=%d calls=%d body=%s", response.Code, repository.updateCalls, response.Body.String())
			}
		})
	}

	repository := &fakeRepository{state: State{UserID: uuid.New(), State: "in_progress"}}
	request := httptest.NewRequest(http.MethodPatch, "/v1/onboarding", strings.NewReader(`{"adultAttested":true}`))
	request.Header.Set("Authorization", "Bearer configured-local-token")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "1234567890123456")
	response := httptest.NewRecorder()
	protectOnboarding(onboardingHandler(repository).Update).ServeHTTP(response, request)
	if response.Code != http.StatusOK || repository.update.Source != "synthetic_local" || repository.key != "1234567890123456" {
		t.Fatalf("status=%d update=%+v key=%q body=%s", response.Code, repository.update, repository.key, response.Body.String())
	}
}

func TestOnboardingUpdateAcceptsCompleteJSONWithExplicitDietClear(t *testing.T) {
	repository := &fakeRepository{state: State{UserID: uuid.New(), State: "complete"}}
	body := `{
		"adultAttested": true,
		"termsVersion": "terms-1",
		"privacyVersion": "privacy-1",
		"timezone": "Asia/Kolkata",
		"units": "metric",
		"primaryGoal": "strength",
		"experienceLevel": "beginner",
		"weeklyAvailability": 3,
		"sessionDurationMinutes": 45,
		"equipmentAccess": ["bodyweight"],
		"dietPreference": null,
		"safetyAcknowledged": true,
		"currentStep": 6
	}`
	request := httptest.NewRequest(http.MethodPatch, "/v1/onboarding", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer configured-local-token")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "complete-json-1234567890")
	response := httptest.NewRecorder()
	protectOnboarding(onboardingHandler(repository).Update).ServeHTTP(response, request)

	if response.Code != http.StatusOK || repository.updateCalls != 1 {
		t.Fatalf("status=%d calls=%d body=%s", response.Code, repository.updateCalls, response.Body.String())
	}
	if repository.update.Source != "synthetic_local" || !repository.update.ClearDietPreference || repository.update.DietPreference != nil {
		t.Fatalf("complete JSON was not preserved: %+v", repository.update)
	}
}

func TestOnboardingUpdateReturnsSafeErrors(t *testing.T) {
	tests := map[string]struct {
		body        string
		contentType string
		repoError   error
		status      int
		code        string
	}{
		"unknown field": {body: `{"diagnosis":"secret"}`, contentType: "application/json", status: http.StatusBadRequest, code: "invalid_request"},
		"media type":    {body: `{"adultAttested":true}`, contentType: "text/plain", status: http.StatusUnsupportedMediaType, code: "invalid_request"},
		"validation":    {body: `{"adultAttested":false}`, contentType: "application/json", status: http.StatusUnprocessableEntity, code: "validation_failed"},
		"idempotency":   {body: `{"adultAttested":true}`, contentType: "application/json", repoError: ErrIdempotencyConflict, status: http.StatusConflict, code: "idempotency_key_conflict"},
		"internal":      {body: `{"adultAttested":true}`, contentType: "application/json", repoError: errors.New("database leaked sensitive value"), status: http.StatusInternalServerError, code: "internal_error"},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			repository := &fakeRepository{err: testCase.repoError}
			request := httptest.NewRequest(http.MethodPatch, "/v1/onboarding", strings.NewReader(testCase.body))
			request.Header.Set("Authorization", "Bearer configured-local-token")
			request.Header.Set("Content-Type", testCase.contentType)
			request.Header.Set("Idempotency-Key", "1234567890123456")
			response := httptest.NewRecorder()
			protectOnboarding(onboardingHandler(repository).Update).ServeHTTP(response, request)
			if response.Code != testCase.status {
				t.Fatalf("status=%d want=%d body=%s", response.Code, testCase.status, response.Body.String())
			}
			var envelope httpx.ErrorEnvelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Error.Code != testCase.code || strings.Contains(response.Body.String(), "database") || strings.Contains(response.Body.String(), "secret") {
				t.Fatalf("unsafe envelope: %+v", envelope.Error)
			}
		})
	}
}

func TestOnboardingRequestHasBoundedDeadline(t *testing.T) {
	repository := &blockingRepository{cancelled: make(chan struct{})}
	handler := onboardingHandler(repository)
	handler.requestTimeout = 25 * time.Millisecond
	request := httptest.NewRequest(http.MethodGet, "/v1/onboarding", nil)
	request.Header.Set("Authorization", "Bearer configured-local-token")
	response := httptest.NewRecorder()
	protectOnboarding(handler.Get).ServeHTTP(response, request)
	if response.Code != http.StatusGatewayTimeout {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	select {
	case <-repository.cancelled:
	default:
		t.Fatal("repository did not observe cancellation")
	}
}
