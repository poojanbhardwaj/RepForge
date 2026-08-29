package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const MaxJSONBody = 16 << 10

var ErrUnsupportedMediaType = errors.New("unsupported media type")

type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	TraceID string         `json:"traceId"`
}

type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

type requestIDKey struct{}

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey{}).(string)
	return value
}

func WriteJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func WriteError(response http.ResponseWriter, request *http.Request, status int, code, message string, details map[string]any) {
	WriteJSON(response, status, ErrorEnvelope{Error: ErrorBody{
		Code: code, Message: message, Details: details, TraceID: RequestID(request.Context()),
	}})
}

func DecodeJSON(response http.ResponseWriter, request *http.Request, destination any) error {
	contentType := request.Header.Get("Content-Type")
	if mediaType := strings.TrimSpace(strings.Split(contentType, ";")[0]); mediaType != "application/json" {
		return fmt.Errorf("%w: Content-Type must be application/json", ErrUnsupportedMediaType)
	}
	request.Body = http.MaxBytesReader(response, request.Body, MaxJSONBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestID := strings.TrimSpace(request.Header.Get("X-Request-ID"))
		if _, err := uuid.Parse(requestID); err != nil {
			generated, generationErr := uuid.NewV7()
			if generationErr != nil {
				generated = uuid.New()
			}
			requestID = generated.String()
		}
		response.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(request.Context(), requestIDKey{}, requestID)
		next.ServeHTTP(response, request.WithContext(ctx))
	})
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("Referrer-Policy", "no-referrer")
		response.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(response, request)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *statusRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func AccessLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: response, status: http.StatusOK}
		next.ServeHTTP(recorder, request)
		route := request.Pattern
		if route == "" {
			route = "unmatched"
		}
		logger.InfoContext(request.Context(), "http_request",
			"method", request.Method,
			"route", route,
			"status", recorder.status,
			"duration_ms", time.Since(started).Milliseconds(),
			"request_id", RequestID(request.Context()),
		)
	})
}

func Recovery(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(request.Context(), "http_panic", "request_id", RequestID(request.Context()))
				WriteError(response, request, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.", nil)
			}
		}()
		next.ServeHTTP(response, request)
	})
}
