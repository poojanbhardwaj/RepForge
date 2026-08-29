package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
)

var ErrUnauthenticated = errors.New("unauthenticated")

type Principal struct {
	Provider string
	Subject  string
	Roles    []string
}

type Authenticator interface {
	Authenticate(context.Context, string) (Principal, error)
}

type DevAuthenticator struct {
	token     string
	provider  string
	subject   string
	principal Principal
}

func NewDevAuthenticator(token, provider, subject string) *DevAuthenticator {
	return &DevAuthenticator{
		token:    token,
		provider: provider,
		subject:  subject,
		principal: Principal{
			Provider: provider,
			Subject:  subject,
			Roles:    []string{"user"},
		},
	}
}

func (a *DevAuthenticator) Authenticate(_ context.Context, token string) (Principal, error) {
	if len(token) != len(a.token) || subtle.ConstantTimeCompare([]byte(token), []byte(a.token)) != 1 {
		return Principal{}, ErrUnauthenticated
	}
	return a.principal, nil
}

type principalKey struct{}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	return principal, ok
}

func Middleware(authenticator Authenticator, unauthorized func(http.ResponseWriter, *http.Request)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			header := request.Header.Get("Authorization")
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
				unauthorized(response, request)
				return
			}
			principal, err := authenticator.Authenticate(request.Context(), strings.TrimSpace(parts[1]))
			if err != nil {
				unauthorized(response, request)
				return
			}
			ctx := context.WithValue(request.Context(), principalKey{}, principal)
			next.ServeHTTP(response, request.WithContext(ctx))
		})
	}
}
