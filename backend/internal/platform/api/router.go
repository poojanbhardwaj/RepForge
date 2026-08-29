package api

import (
	"context"
	"net/http"
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

type BuildInfo struct {
	Version string
	Commit  string
	BuiltAt string
}

type Options struct {
	Readiness        ReadinessChecker
	ReadinessTimeout time.Duration
	Users            UserHandler
	Authenticator    auth.Authenticator
	Build            BuildInfo
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
}

func Routes() []Route {
	result := make([]Route, len(routes))
	copy(result, routes)
	return result
}

func NewRouter(options Options) http.Handler {
	unauthorized := func(response http.ResponseWriter, request *http.Request) {
		httpx.WriteError(response, request, http.StatusUnauthorized, "unauthenticated", "Authentication is required.", nil)
	}
	protected := func(handler http.Handler) http.Handler {
		return auth.Middleware(options.Authenticator, unauthorized)(handler)
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
		"getMe":    protected(http.HandlerFunc(options.Users.Get)),
		"updateMe": protected(http.HandlerFunc(options.Users.Update)),
	}

	mux := http.NewServeMux()
	for _, route := range routes {
		mux.Handle(route.Method+" "+route.Path, handlers[route.OperationID])
	}
	return mux
}
