package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"repforge.local/backend/internal/auth"
	"repforge.local/backend/internal/platform/httpx"
)

type ReadinessChecker interface {
	Ping(context.Context) error
}

type UserHandler interface {
	Get(http.ResponseWriter, *http.Request)
	Update(http.ResponseWriter, *http.Request)
}

type OnboardingHandler interface {
	Get(http.ResponseWriter, *http.Request)
	Update(http.ResponseWriter, *http.Request)
}

type BuildInfo struct {
	Version string
	Commit  string
	BuiltAt string
}

type Options struct {
	Readiness         ReadinessChecker
	ReadinessTimeout  time.Duration
	Users             UserHandler
	Onboarding        OnboardingHandler
	Authenticator     auth.Authenticator
	CORSAllowedOrigin string
	Build             BuildInfo
}

type Route struct {
	Method      string
	Path        string
	OperationID string
}

var routes = []Route{
	{Method: http.MethodGet, Path: "/healthz", OperationID: "getHealth"},
	{Method: http.MethodGet, Path: "/readyz", OperationID: "getReadiness"},
	{Method: http.MethodGet, Path: "/version", OperationID: "getVersion"},
	{Method: http.MethodGet, Path: "/v1/me", OperationID: "getMe"},
	{Method: http.MethodPatch, Path: "/v1/me", OperationID: "updateMe"},
	{Method: http.MethodGet, Path: "/v1/onboarding", OperationID: "getOnboarding"},
	{Method: http.MethodPatch, Path: "/v1/onboarding", OperationID: "updateOnboarding"},
}

func Routes() []Route {
	result := make([]Route, len(routes))
	copy(result, routes)
	return result
}

func NewRouter(options Options) http.Handler {
	unauthorized := func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("WWW-Authenticate", `Bearer realm="repforge", error="invalid_token"`)
		httpx.WriteError(response, request, http.StatusUnauthorized, "unauthenticated", "Authentication is required.", nil)
	}
	forbidden := func(response http.ResponseWriter, request *http.Request, scopes []string) {
		response.Header().Set("WWW-Authenticate", `Bearer realm="repforge", error="insufficient_scope", scope="`+strings.Join(scopes, " ")+`"`)
		httpx.WriteError(response, request, http.StatusForbidden, "insufficient_scope", "The access token does not grant the required scope.", nil)
	}
	protected := func(handler http.Handler, scopes ...string) http.Handler {
		required := auth.RequireScopes(scopes, forbidden)(handler)
		return auth.Middleware(options.Authenticator, unauthorized)(required)
	}
	handlers := map[string]http.Handler{
		"getHealth": http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			httpx.WriteJSON(response, http.StatusOK, map[string]string{"status": "ok"})
		}),
		"getReadiness": http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			readyCtx, readyCancel := context.WithTimeout(request.Context(), options.ReadinessTimeout)
			defer readyCancel()
			if err := options.Readiness.Ping(readyCtx); err != nil {
				httpx.WriteError(response, request, http.StatusServiceUnavailable, "not_ready", "The service is not ready.", nil)
				return
			}
			httpx.WriteJSON(response, http.StatusOK, map[string]string{"status": "ready"})
		}),
		"getVersion": http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			httpx.WriteJSON(response, http.StatusOK, map[string]string{
				"version": options.Build.Version,
				"commit":  options.Build.Commit,
				"builtAt": options.Build.BuiltAt,
			})
		}),
		"getMe":            protected(http.HandlerFunc(options.Users.Get), "profile:read"),
		"updateMe":         protected(http.HandlerFunc(options.Users.Update), "profile:write"),
		"getOnboarding":    protected(http.HandlerFunc(options.Onboarding.Get), "profile:read"),
		"updateOnboarding": protected(http.HandlerFunc(options.Onboarding.Update), "profile:write"),
	}

	mux := http.NewServeMux()
	for _, route := range routes {
		mux.Handle(route.Method+" "+route.Path, handlers[route.OperationID])
	}
	return cors(options.CORSAllowedOrigin, mux)
}

func cors(allowedOrigin string, next http.Handler) http.Handler {
	if allowedOrigin == "" {
		return next
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if origin == allowedOrigin {
			response.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			response.Header().Set("Access-Control-Allow-Methods", "GET, PATCH, OPTIONS")
			response.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key")
			response.Header().Add("Vary", "Origin")
			if request.Method == http.MethodOptions {
				response.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(response, request)
	})
}
