package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProductionMiddlewareLogsMatchedRouteAndRealNotFound(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, nil))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/me", func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	})
	handler := newHTTPHandler(logger, mux)

	for _, target := range []string{"/v1/me", "/not-a-route"} {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, target, nil))
	}

	wantRoutes := []string{"GET /v1/me", "unmatched"}
	scanner := bufio.NewScanner(&buffer)
	for index, want := range wantRoutes {
		if !scanner.Scan() {
			t.Fatalf("missing log event %d", index)
		}
		var event map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("decode log event %d: %v", index, err)
		}
		if got := event["route"]; got != want {
			t.Errorf("event %d route = %q, want %q", index, got, want)
		}
	}
	if scanner.Scan() {
		t.Fatalf("unexpected extra log event: %s", scanner.Text())
	}
}
