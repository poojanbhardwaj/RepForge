package openapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"

	"repforge.local/backend/internal/auth"
	"repforge.local/backend/internal/onboarding"
	apiserver "repforge.local/backend/internal/platform/api"
	"repforge.local/backend/internal/platform/httpx"
	"repforge.local/backend/internal/users"
)

func TestContractIsValidOpenAPI31(t *testing.T) {
	document := loadContract(t)
	if err := document.Validate(context.Background()); err != nil {
		t.Fatalf("validate OpenAPI: %v", err)
	}
	contractRoutes := make(map[string]string)
	for path, pathItem := range document.Paths.Map() {
		for method, operation := range map[string]*openapi3.Operation{
			http.MethodGet: pathItem.Get, http.MethodPost: pathItem.Post, http.MethodPut: pathItem.Put,
			http.MethodPatch: pathItem.Patch, http.MethodDelete: pathItem.Delete,
		} {
			if operation != nil {
				contractRoutes[method+" "+path] = operation.OperationID
			}
		}
	}
	serverRoutes := apiserver.Routes()
	if len(serverRoutes) != len(contractRoutes) {
		t.Fatalf("server route count = %d, contract route count = %d", len(serverRoutes), len(contractRoutes))
	}
	for _, route := range serverRoutes {
		key := route.Method + " " + route.Path
		if operationID := contractRoutes[key]; operationID != route.OperationID {
			t.Errorf("server route %s operation = %q, contract operation = %q", key, route.OperationID, operationID)
		}
	}
}

type readyChecker struct{}

func (readyChecker) Ping(context.Context) error { return nil }

type contractRepository struct {
	profile users.Profile
}

type contractOnboardingHandler struct {
	state onboarding.State
}

func (handler contractOnboardingHandler) Get(response http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(response, http.StatusOK, handler.state)
}

func (handler contractOnboardingHandler) Update(response http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(response, http.StatusOK, handler.state)
}

func (repository *contractRepository) Get(context.Context, users.Identity) (users.Profile, error) {
	return repository.profile, nil
}

func (repository *contractRepository) Update(_ context.Context, _ users.Identity, update users.Update) (users.Profile, error) {
	profile := repository.profile
	profile.DisplayName = update.DisplayName
	profile.Version++
	return profile, nil
}

func TestProfileHandlerResponsesMatchContractSchemas(t *testing.T) {
	document := loadContract(t)
	router := newContractRouter()

	t.Run("get success", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		request.Header.Set("Authorization", "Bearer configured-local-token")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d; body=%s", response.Code, response.Body.String())
		}
		validateResponseSchema(t, document, "Me", response)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d; body=%s", response.Code, response.Body.String())
		}
		validateResponseSchema(t, document, "ErrorEnvelope", response)
	})

	t.Run("unsupported media type", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPatch, "/v1/me", strings.NewReader(`{"displayName":"Athlete","expectedVersion":1}`))
		request.Header.Set("Authorization", "Bearer configured-local-token")
		request.Header.Set("Content-Type", "text/plain")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("status = %d; body=%s", response.Code, response.Body.String())
		}
		validateResponseSchema(t, document, "ErrorEnvelope", response)
	})
}

func TestOnboardingHandlerResponseMatchesContractSchema(t *testing.T) {
	document := loadContract(t)
	request := httptest.NewRequest(http.MethodGet, "/v1/onboarding", nil)
	request.Header.Set("Authorization", "Bearer configured-local-token")
	response := httptest.NewRecorder()
	newContractRouter().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", response.Code, response.Body.String())
	}
	validateResponseSchema(t, document, "OnboardingState", response)
}

func TestContractRoutesExecuteAgainstRealRouter(t *testing.T) {
	router := newContractRouter()
	for _, route := range apiserver.Routes() {
		t.Run(route.OperationID, func(t *testing.T) {
			var body *strings.Reader
			if route.Method == http.MethodPatch {
				body = strings.NewReader(`{"displayName":"Athlete","expectedVersion":1}`)
			} else {
				body = strings.NewReader("")
			}
			request := httptest.NewRequest(route.Method, route.Path, body)
			if strings.HasPrefix(route.Path, "/v1/") {
				request.Header.Set("Authorization", "Bearer configured-local-token")
			}
			if route.Method == http.MethodPatch {
				request.Header.Set("Content-Type", "application/json")
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code == http.StatusNotFound || response.Code == http.StatusMethodNotAllowed {
				t.Fatalf("%s %s returned %d", route.Method, route.Path, response.Code)
			}
		})
	}
}

func TestVersionRouteReturnsInjectedBuildMetadata(t *testing.T) {
	response := httptest.NewRecorder()
	newContractRouter().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/version", nil))
	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"version": "0.1.0-test", "commit": strings.Repeat("a", 40), "builtAt": "2026-08-29T00:00:00Z",
	}
	for key, value := range want {
		if payload[key] != value {
			t.Errorf("%s = %q, want %q", key, payload[key], value)
		}
	}
}

func newContractRouter() http.Handler {
	now := time.Date(2026, time.August, 28, 12, 0, 0, 0, time.UTC)
	repository := &contractRepository{profile: users.Profile{
		UserID: uuid.MustParse("01993c86-8fc9-7a5a-9b8d-3302c7e917ec"), DisplayName: "Local Athlete",
		Version: 1, CreatedAt: now, UpdatedAt: now, Consents: []users.Consent{},
	}}
	return apiserver.NewRouter(apiserver.Options{
		Readiness: readyChecker{}, ReadinessTimeout: time.Second,
		Users: users.NewHandler(users.NewService(repository)),
		Onboarding: contractOnboardingHandler{state: onboarding.State{
			UserID: uuid.MustParse("01993c86-8fc9-7a5a-9b8d-3302c7e917ec"), State: "in_progress",
			CurrentStep: 1, Version: 1, EquipmentAccess: []string{}, CreatedAt: now, UpdatedAt: now,
			RequiredTermsVersion: "draft-local-1", RequiredPrivacyVersion: "draft-local-1",
		}},
		Authenticator: auth.NewDevAuthenticator("configured-local-token", "repforge-dev", "local-user"),
		Build: apiserver.BuildInfo{
			Version: "0.1.0-test", Commit: strings.Repeat("a", 40), BuiltAt: "2026-08-29T00:00:00Z",
		},
	})
}

func loadContract(t *testing.T) *openapi3.T {
	t.Helper()
	path, err := filepath.Abs("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	document, err := openapi3.NewLoader().LoadFromFile(path)
	if err != nil {
		t.Fatalf("load OpenAPI: %v", err)
	}
	return document
}

func validateResponseSchema(t *testing.T, document *openapi3.T, schemaName string, response *httptest.ResponseRecorder) {
	t.Helper()
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}
	var payload any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response JSON: %v", err)
	}
	schema, ok := document.Components.Schemas[schemaName]
	if !ok || schema.Value == nil {
		t.Fatalf("contract schema %q is missing", schemaName)
	}
	if err := schema.Value.VisitJSON(payload, openapi3.EnableJSONSchema2020()); err != nil {
		t.Fatalf("response does not match %s: %v\nbody=%s", schemaName, err, response.Body.String())
	}
}
