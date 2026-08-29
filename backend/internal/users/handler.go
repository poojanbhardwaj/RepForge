package users

import (
	"context"
	"errors"
	"net/http"
	"time"

	"repforge.local/backend/internal/auth"
	"repforge.local/backend/internal/platform/httpx"
)

const profileRequestTimeout = 10 * time.Second

type Handler struct {
	service        *Service
	requestTimeout time.Duration
}

type consentResponse struct {
	DocumentKey     string     `json:"documentKey"`
	DocumentVersion string     `json:"documentVersion"`
	AcceptedAt      time.Time  `json:"acceptedAt"`
	WithdrawnAt     *time.Time `json:"withdrawnAt"`
}

type profileResponse struct {
	DisplayName string    `json:"displayName"`
	Version     int64     `json:"version"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type meResponse struct {
	ID       string            `json:"id"`
	Profile  profileResponse   `json:"profile"`
	Consents []consentResponse `json:"consents"`
}

type updateRequest struct {
	DisplayName     string `json:"displayName"`
	ExpectedVersion int64  `json:"expectedVersion"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service, requestTimeout: profileRequestTimeout}
}

func (h *Handler) Get(response http.ResponseWriter, request *http.Request) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		httpx.WriteError(response, request, http.StatusUnauthorized, "unauthenticated", "Authentication is required.", nil)
		return
	}
	request, cancel := h.withDeadline(request)
	defer cancel()
	profile, err := h.service.Get(request.Context(), Identity{Provider: principal.Provider, Subject: principal.Subject})
	if err != nil {
		h.writeServiceError(response, request, err)
		return
	}
	if errors.Is(request.Context().Err(), context.DeadlineExceeded) {
		h.writeServiceError(response, request, context.DeadlineExceeded)
		return
	}
	httpx.WriteJSON(response, http.StatusOK, present(profile))
}

func (h *Handler) Update(response http.ResponseWriter, request *http.Request) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		httpx.WriteError(response, request, http.StatusUnauthorized, "unauthenticated", "Authentication is required.", nil)
		return
	}
	request, cancel := h.withDeadline(request)
	defer cancel()
	var body updateRequest
	if err := httpx.DecodeJSON(response, request, &body); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, httpx.ErrUnsupportedMediaType) {
			status = http.StatusUnsupportedMediaType
		}
		httpx.WriteError(response, request, status, "invalid_request", "The request body is invalid.", nil)
		return
	}
	profile, err := h.service.Update(request.Context(), Identity{Provider: principal.Provider, Subject: principal.Subject}, Update{
		DisplayName: body.DisplayName, ExpectedVersion: body.ExpectedVersion,
	})
	if err != nil {
		h.writeServiceError(response, request, err)
		return
	}
	if errors.Is(request.Context().Err(), context.DeadlineExceeded) {
		h.writeServiceError(response, request, context.DeadlineExceeded)
		return
	}
	httpx.WriteJSON(response, http.StatusOK, present(profile))
}

func (h *Handler) withDeadline(request *http.Request) (*http.Request, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(request.Context(), h.requestTimeout)
	return request.WithContext(ctx), cancel
}

func (h *Handler) writeServiceError(response http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(request.Context().Err(), context.DeadlineExceeded), errors.Is(err, context.DeadlineExceeded):
		httpx.WriteError(response, request, http.StatusGatewayTimeout, "profile_request_timeout", "The profile request timed out.", nil)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(response, request, http.StatusNotFound, "profile_not_found", "The profile was not found.", nil)
	case errors.Is(err, ErrConflict):
		httpx.WriteError(response, request, http.StatusConflict, "version_conflict", "The profile changed. Refresh and try again.", nil)
	case errors.Is(err, ErrValidation):
		httpx.WriteError(response, request, http.StatusUnprocessableEntity, "validation_failed", "The profile is invalid.", map[string]any{"displayName": err.Error()})
	default:
		httpx.WriteError(response, request, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.", nil)
	}
}

func present(profile Profile) meResponse {
	result := meResponse{
		ID:       profile.UserID.String(),
		Profile:  profileResponse{DisplayName: profile.DisplayName, Version: profile.Version, CreatedAt: profile.CreatedAt, UpdatedAt: profile.UpdatedAt},
		Consents: make([]consentResponse, 0, len(profile.Consents)),
	}
	for _, consent := range profile.Consents {
		result.Consents = append(result.Consents, consentResponse{
			DocumentKey: consent.DocumentKey, DocumentVersion: consent.DocumentVersion,
			AcceptedAt: consent.AcceptedAt, WithdrawnAt: consent.WithdrawnAt,
		})
	}
	return result
}
