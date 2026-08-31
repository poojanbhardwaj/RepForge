package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"repforge.local/backend/internal/auth"
)

type readyStub struct{}

func (readyStub) Ping(context.Context) error { return nil }

type handlerStub struct{}

type scopedAuthenticator struct{ scopes []string }

func (authenticator scopedAuthenticator) Authenticate(_ context.Context, token string) (auth.Principal, error) {
	if token != "scoped-token" {
		return auth.Principal{}, auth.ErrUnauthenticated
	}
	return auth.Principal{Provider: "test", Subject: "subject", Scopes: authenticator.scopes}, nil
}

func (handlerStub) Get(response http.ResponseWriter, _ *http.Request) {
	response.WriteHeader(http.StatusOK)
}
func (handlerStub) Update(response http.ResponseWriter, _ *http.Request) {
	response.WriteHeader(http.StatusOK)
}

func testRouter() http.Handler {
	return NewRouter(Options{
		Readiness: readyStub{}, ReadinessTimeout: time.Second, Users: handlerStub{}, Onboarding: handlerStub{},
		Authenticator:     auth.NewDevAuthenticator("configured-local-token", "test", "subject"),
		CORSAllowedOrigin: "http://127.0.0.1:3000",
	})
}

func TestCORSUsesAnExactOriginWithoutCredentialsOrReflection(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		origin     string
		wantOrigin string
	}{
		{name: "allowed", origin: "http://127.0.0.1:3000", wantOrigin: "http://127.0.0.1:3000"},
		{name: "localhost is not equivalent", origin: "http://localhost:3000"},
		{name: "host suffix is rejected", origin: "http://127.0.0.1:3000.attacker.invalid"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			request.Header.Set("Origin", testCase.origin)
			response := httptest.NewRecorder()
			testRouter().ServeHTTP(response, request)
			if response.Header().Get("Access-Control-Allow-Origin") != testCase.wantOrigin {
				t.Fatalf("allow origin=%q", response.Header().Get("Access-Control-Allow-Origin"))
			}
			if response.Header().Get("Access-Control-Allow-Credentials") != "" || response.Header().Get("Access-Control-Allow-Origin") == "*" {
				t.Fatalf("unsafe CORS headers: %#v", response.Header())
			}
		})
	}
}

func TestCORSAllowsOnlyConfiguredPreflightSurface(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "/v1/onboarding", nil)
	request.Header.Set("Origin", "http://127.0.0.1:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodPatch)
	request.Header.Set("Access-Control-Request-Headers", "content-type,idempotency-key")
	response := httptest.NewRecorder()
	testRouter().ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Access-Control-Allow-Methods") != "GET, PATCH, OPTIONS" ||
		response.Header().Get("Access-Control-Allow-Headers") != "Authorization, Content-Type, Idempotency-Key" {
		t.Fatalf("preflight headers=%#v", response.Header())
	}
}

func TestOnboardingReadAndWriteScopeMatrix(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		scopes     []string
		wantStatus int
	}{
		{name: "authenticated read succeeds", method: http.MethodGet, scopes: []string{"profile:read"}, wantStatus: http.StatusOK},
		{name: "write without permission is forbidden", method: http.MethodPatch, scopes: []string{"profile:read"}, wantStatus: http.StatusForbidden},
		{name: "write with permission succeeds", method: http.MethodPatch, scopes: []string{"profile:read", "profile:write"}, wantStatus: http.StatusOK},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			router := NewRouter(Options{
				Readiness: readyStub{}, ReadinessTimeout: time.Second, Users: handlerStub{}, Onboarding: handlerStub{},
				Authenticator: scopedAuthenticator{scopes: testCase.scopes},
			})
			request := httptest.NewRequest(testCase.method, "/v1/onboarding", nil)
			request.Header.Set("Authorization", "Bearer scoped-token")
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != testCase.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", response.Code, testCase.wantStatus, response.Body.String())
			}
			if testCase.wantStatus == http.StatusForbidden {
				if !strings.Contains(response.Header().Get("WWW-Authenticate"), `error="insufficient_scope"`) ||
					!strings.Contains(response.Body.String(), `"code":"insufficient_scope"`) {
					t.Fatalf("missing insufficient-scope response: headers=%v body=%s", response.Header(), response.Body.String())
				}
			}
		})
	}
}
