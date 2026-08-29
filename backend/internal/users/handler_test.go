package users

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
	profile Profile
	update  Update
	err     error
}

type blockingRepository struct {
	deadline  chan time.Time
	cancelled chan struct{}
}

func (repository *blockingRepository) block(ctx context.Context) (Profile, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return Profile{}, errors.New("profile context has no deadline")
	}
	repository.deadline <- deadline
	<-ctx.Done()
	close(repository.cancelled)
	return Profile{}, ctx.Err()
}

func (repository *blockingRepository) Get(ctx context.Context, _ Identity) (Profile, error) {
	return repository.block(ctx)
}

func (repository *blockingRepository) Update(ctx context.Context, _ Identity, _ Update) (Profile, error) {
	return repository.block(ctx)
}

type headerCountingRecorder struct {
	*httptest.ResponseRecorder
	headerWrites int
}

func (recorder *headerCountingRecorder) WriteHeader(status int) {
	recorder.headerWrites++
	recorder.ResponseRecorder.WriteHeader(status)
}

func (f *fakeRepository) Get(_ context.Context, _ Identity) (Profile, error) {
	return f.profile, f.err
}

func (f *fakeRepository) Update(_ context.Context, _ Identity, update Update) (Profile, error) {
	f.update = update
	return f.profile, f.err
}

func protectedHandler(handler http.HandlerFunc) http.Handler {
	authenticator := auth.NewDevAuthenticator("configured-local-token", "repforge-dev", "local-user")
	unauthorized := func(response http.ResponseWriter, request *http.Request) {
		httpx.WriteError(response, request, http.StatusUnauthorized, "unauthenticated", "Authentication is required.", nil)
	}
	return httpx.RequestIDMiddleware(auth.Middleware(authenticator, unauthorized)(handler))
}

func TestGetMeRequiresAuthentication(t *testing.T) {
	repository := &fakeRepository{}
	handler := NewHandler(NewService(repository))
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	response := httptest.NewRecorder()
	protectedHandler(handler.Get).ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}

func TestGetMeReturnsAuthenticatedProfile(t *testing.T) {
	now := time.Now().UTC()
	repository := &fakeRepository{profile: Profile{UserID: uuid.New(), DisplayName: "Local Athlete", Version: 1, CreatedAt: now, UpdatedAt: now}}
	handler := NewHandler(NewService(repository))
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request.Header.Set("Authorization", "Bearer configured-local-token")
	response := httptest.NewRecorder()
	protectedHandler(handler.Get).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Local Athlete") {
		t.Fatalf("status/body = %d %s", response.Code, response.Body.String())
	}
}

func TestUpdateMeValidatesAndDetectsConflict(t *testing.T) {
	for name, testCase := range map[string]struct {
		repository *fakeRepository
		body       string
		want       int
	}{
		"invalid":  {repository: &fakeRepository{}, body: `{"displayName":"","expectedVersion":1}`, want: http.StatusUnprocessableEntity},
		"conflict": {repository: &fakeRepository{err: ErrConflict}, body: `{"displayName":"Athlete","expectedVersion":1}`, want: http.StatusConflict},
	} {
		t.Run(name, func(t *testing.T) {
			handler := NewHandler(NewService(testCase.repository))
			request := httptest.NewRequest(http.MethodPatch, "/v1/me", strings.NewReader(testCase.body))
			request.Header.Set("Authorization", "Bearer configured-local-token")
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			protectedHandler(handler.Update).ServeHTTP(response, request)
			if response.Code != testCase.want {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, testCase.want, response.Body.String())
			}
		})
	}
}

func TestUpdateMeRejectsUnsupportedMediaType(t *testing.T) {
	repository := &fakeRepository{}
	handler := NewHandler(NewService(repository))
	request := httptest.NewRequest(http.MethodPatch, "/v1/me", strings.NewReader(`{"displayName":"Athlete","expectedVersion":1}`))
	request.Header.Set("Authorization", "Bearer configured-local-token")
	request.Header.Set("Content-Type", "text/plain")
	response := httptest.NewRecorder()
	protectedHandler(handler.Update).ServeHTTP(response, request)
	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415; body=%s", response.Code, response.Body.String())
	}
}

func TestUpdateMeHidesRepositoryErrors(t *testing.T) {
	repository := &fakeRepository{err: errors.New("database contains Sensitive Name")}
	handler := NewHandler(NewService(repository))
	request := httptest.NewRequest(http.MethodPatch, "/v1/me", strings.NewReader(`{"displayName":"Athlete","expectedVersion":1}`))
	request.Header.Set("Authorization", "Bearer configured-local-token")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	protectedHandler(handler.Update).ServeHTTP(response, request)
	if strings.Contains(response.Body.String(), "database") {
		t.Fatalf("internal error leaked: %s", response.Body.String())
	}
}

func TestProfileRequestsCancelBlockedRepositoriesAtTheBoundedDeadline(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		method      string
		body        string
		contentType string
		endpoint    func(*Handler) http.HandlerFunc
	}{
		{name: "get", method: http.MethodGet, endpoint: func(handler *Handler) http.HandlerFunc { return handler.Get }},
		{
			name: "update", method: http.MethodPatch,
			body: `{"displayName":"Athlete","expectedVersion":1}`, contentType: "application/json",
			endpoint: func(handler *Handler) http.HandlerFunc { return handler.Update },
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			repository := &blockingRepository{deadline: make(chan time.Time, 1), cancelled: make(chan struct{})}
			handler := NewHandler(NewService(repository))
			handler.requestTimeout = 50 * time.Millisecond
			request := httptest.NewRequest(testCase.method, "/v1/me", strings.NewReader(testCase.body))
			request.Header.Set("Authorization", "Bearer configured-local-token")
			if testCase.contentType != "" {
				request.Header.Set("Content-Type", testCase.contentType)
			}
			response := &headerCountingRecorder{ResponseRecorder: httptest.NewRecorder()}
			started := time.Now()
			done := make(chan struct{})
			go func() {
				protectedHandler(testCase.endpoint(handler)).ServeHTTP(response, request)
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("profile request exceeded its bounded completion time")
			}
			if elapsed := time.Since(started); elapsed > time.Second {
				t.Fatalf("profile request completed in %s, want at most 1s", elapsed)
			}
			deadline := <-repository.deadline
			if remaining := deadline.Sub(started); remaining <= 0 || remaining > 250*time.Millisecond {
				t.Fatalf("repository deadline was %s after request start", remaining)
			}
			select {
			case <-repository.cancelled:
			default:
				t.Fatal("repository did not observe context cancellation")
			}
			if response.Code != http.StatusGatewayTimeout {
				t.Fatalf("status = %d, want 504; body=%s", response.Code, response.Body.String())
			}
			if response.headerWrites != 1 {
				t.Fatalf("WriteHeader calls = %d, want exactly 1", response.headerWrites)
			}
			var envelope httpx.ErrorEnvelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Error.Code != "profile_request_timeout" || envelope.Error.Message != "The profile request timed out." {
				t.Fatalf("unexpected timeout response: %+v", envelope.Error)
			}
			if strings.Contains(response.Body.String(), "deadline") || strings.Contains(response.Body.String(), "database") {
				t.Fatalf("timeout response leaked internals: %s", response.Body.String())
			}
		})
	}
}
