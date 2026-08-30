package onboarding

import (
	"context"
	"errors"
	"net/http"
	"time"

	"repforge.local/backend/internal/auth"
	"repforge.local/backend/internal/platform/httpx"
)

const onboardingRequestTimeout = 10 * time.Second

type Handler struct {
	service        *Service
	clientSources  map[string]string
	requestTimeout time.Duration
}

func NewHandler(service *Service, clientSources map[string]string) *Handler {
	return &Handler{service: service, clientSources: clientSources, requestTimeout: onboardingRequestTimeout}
}

func (handler *Handler) Get(response http.ResponseWriter, request *http.Request) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		httpx.WriteError(response, request, http.StatusUnauthorized, "unauthenticated", "Authentication is required.", nil)
		return
	}
	request, cancel := handler.withDeadline(request)
	defer cancel()
	state, err := handler.service.Get(request.Context(), Identity{Provider: principal.Provider, Subject: principal.Subject})
	if err != nil {
		handler.writeError(response, request, err)
		return
	}
	httpx.WriteJSON(response, http.StatusOK, state)
}

func (handler *Handler) Update(response http.ResponseWriter, request *http.Request) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		httpx.WriteError(response, request, http.StatusUnauthorized, "unauthenticated", "Authentication is required.", nil)
		return
	}
	source, ok := handler.clientSources[principal.ClientID]
	if !ok {
		httpx.WriteError(response, request, http.StatusUnauthorized, "unauthenticated", "Authentication is required.", nil)
		return
	}
	idempotencyHeaders := request.Header.Values("Idempotency-Key")
	if len(idempotencyHeaders) != 1 || ValidateIdempotencyKey(idempotencyHeaders[0]) != nil {
		httpx.WriteError(response, request, http.StatusBadRequest, "invalid_idempotency_key", "A valid Idempotency-Key header is required.", nil)
		return
	}
	request, cancel := handler.withDeadline(request)
	defer cancel()
	var update Update
	if err := httpx.DecodeJSON(response, request, &update); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, httpx.ErrUnsupportedMediaType) {
			status = http.StatusUnsupportedMediaType
		}
		httpx.WriteError(response, request, status, "invalid_request", "The request body is invalid.", nil)
		return
	}
	update.Source = source
	state, err := handler.service.Update(request.Context(), Identity{Provider: principal.Provider, Subject: principal.Subject}, idempotencyHeaders[0], update)
	if err != nil {
		handler.writeError(response, request, err)
		return
	}
	httpx.WriteJSON(response, http.StatusOK, state)
}

func (handler *Handler) withDeadline(request *http.Request) (*http.Request, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(request.Context(), handler.requestTimeout)
	return request.WithContext(ctx), cancel
}

func (handler *Handler) writeError(response http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(request.Context().Err(), context.DeadlineExceeded), errors.Is(err, context.DeadlineExceeded):
		httpx.WriteError(response, request, http.StatusGatewayTimeout, "onboarding_request_timeout", "The onboarding request timed out.", nil)
	case errors.Is(err, ErrValidation):
		httpx.WriteError(response, request, http.StatusUnprocessableEntity, "validation_failed", "The onboarding information is invalid.", nil)
	case errors.Is(err, ErrIdempotencyConflict):
		httpx.WriteError(response, request, http.StatusConflict, "idempotency_key_conflict", "That Idempotency-Key was already used for a different request.", nil)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(response, request, http.StatusNotFound, "onboarding_not_found", "Onboarding was not found.", nil)
	default:
		httpx.WriteError(response, request, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.", nil)
	}
}
