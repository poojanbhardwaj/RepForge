package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONRejectsUnknownFieldsAndOversizedBodies(t *testing.T) {
	for name, body := range map[string]string{
		"unknown":   `{"known":"value","secret":"must not pass"}`,
		"oversized": `{"known":"` + strings.Repeat("x", MaxJSONBody) + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPatch, "/v1/me", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			var target struct {
				Known string `json:"known"`
			}
			if err := DecodeJSON(response, request, &target); err == nil {
				t.Fatal("expected strict decoder rejection")
			}
		})
	}
}

func TestDecodeJSONClassifiesUnsupportedMediaType(t *testing.T) {
	request := httptest.NewRequest(http.MethodPatch, "/v1/me", strings.NewReader(`{"known":"value"}`))
	request.Header.Set("Content-Type", "text/plain")
	response := httptest.NewRecorder()
	var target struct {
		Known string `json:"known"`
	}
	if err := DecodeJSON(response, request, &target); !errors.Is(err, ErrUnsupportedMediaType) {
		t.Fatalf("error = %v, want unsupported media type", err)
	}
}

func TestAccessLogDoesNotRecordHeadersOrBody(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, nil))
	handler := RequestIDMiddleware(AccessLog(logger, http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		WriteJSON(response, http.StatusOK, map[string]string{"status": "ok"})
	})))
	request := httptest.NewRequest(http.MethodPost, "/v1/me", strings.NewReader(`{"displayName":"Sensitive Name"}`))
	request.Header.Set("Authorization", "Bearer very-sensitive-token")
	handler.ServeHTTP(httptest.NewRecorder(), request)
	logged := buffer.String()
	for _, prohibited := range []string{"Sensitive Name", "very-sensitive-token", "Authorization"} {
		if strings.Contains(logged, prohibited) {
			t.Fatalf("log contains prohibited value %q: %s", prohibited, logged)
		}
	}
	var event map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &event); err != nil {
		t.Fatalf("log is not JSON: %v", err)
	}
}
