package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"repforge.local/backend/internal/auth"
)

type readyStub struct{}

func (readyStub) Ping(context.Context) error { return nil }

type handlerStub struct{}

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
