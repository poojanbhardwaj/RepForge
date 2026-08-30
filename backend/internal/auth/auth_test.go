package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDevAuthenticatorAcceptsOnlyConfiguredToken(t *testing.T) {
	authenticator := NewDevAuthenticator("configured-local-token", "repforge-dev", "local-user")
	principal, err := authenticator.Authenticate(context.Background(), "configured-local-token")
	if err != nil || principal.Subject != "local-user" {
		t.Fatalf("valid token: principal=%+v error=%v", principal, err)
	}
	if _, err := authenticator.Authenticate(context.Background(), "different-local-token"); err == nil {
		t.Fatal("expected invalid token rejection")
	}
}

type staticAuthenticator struct{ principal Principal }

func (authenticator staticAuthenticator) Authenticate(context.Context, string) (Principal, error) {
	return authenticator.principal, nil
}

func TestMiddlewareRejectsMalformedAndDuplicateAuthorizationHeaders(t *testing.T) {
	for _, headers := range [][]string{{}, {"Basic abc"}, {"Bearer"}, {"Bearer one", "Bearer two"}} {
		request := httptest.NewRequest(http.MethodGet, "/protected", nil)
		for _, header := range headers {
			request.Header.Add("Authorization", header)
		}
		response := httptest.NewRecorder()
		called := false
		handler := Middleware(staticAuthenticator{}, func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusUnauthorized)
		})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized || called {
			t.Fatalf("headers %v were not rejected safely", headers)
		}
	}
}

func TestRequireScopesRejectsMissingScope(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	called := false
	handler := Middleware(staticAuthenticator{principal: Principal{Subject: "subject", Scopes: []string{"profile:read"}}}, func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
	})(RequireScopes([]string{"profile:write"}, func(writer http.ResponseWriter, _ *http.Request, _ []string) {
		writer.WriteHeader(http.StatusForbidden)
	})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })))
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || called {
		t.Fatalf("status=%d called=%v, want forbidden", response.Code, called)
	}
}
