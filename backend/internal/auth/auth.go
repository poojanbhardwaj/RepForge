package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
)

var ErrUnauthenticated = errors.New("unauthenticated")
var ErrForbidden = errors.New("forbidden")
var ErrEmailUnverified = errors.New("email is not verified")

type Principal struct {
	Provider string
	Subject  string
	ClientID string
	Scopes   []string
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
			ClientID: "repforge-dev-client",
			Scopes:   []string{"profile:read", "profile:write"},
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
			headers := request.Header.Values("Authorization")
			if len(headers) != 1 {
				unauthorized(response, request)
				return
			}
			parts := strings.Fields(headers[0])
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				unauthorized(response, request)
				return
			}
			principal, err := authenticator.Authenticate(request.Context(), parts[1])
			if err != nil {
				unauthorized(response, request)
				return
			}
			ctx := context.WithValue(request.Context(), principalKey{}, principal)
			next.ServeHTTP(response, request.WithContext(ctx))
		})
	}
}

func RequireScopes(required []string, forbidden func(http.ResponseWriter, *http.Request, []string)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			principal, ok := PrincipalFromContext(request.Context())
			if !ok {
				forbidden(response, request, required)
				return
			}
			granted := make(map[string]struct{}, len(principal.Scopes))
			for _, scope := range principal.Scopes {
				granted[scope] = struct{}{}
			}
			for _, scope := range required {
				if _, ok := granted[scope]; !ok {
					forbidden(response, request, required)
					return
				}
			}
			next.ServeHTTP(response, request)
		})
	}
}
