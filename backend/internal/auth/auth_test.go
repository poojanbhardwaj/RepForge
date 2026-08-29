package auth

import (
	"context"
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
