package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"msp/internal/api"
)

func ready(context.Context) error { return nil }
func notReady(context.Context) error {
	return errors.New("db: not ready: dial 10.0.0.7:5432 as msp_app")
}

func discard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestRouter(t *testing.T) {
	tests := []struct {
		name            string
		ready           ReadyChecker
		method          string
		path            string
		wantStatus      int
		wantContentType string
	}{
		{name: "healthz ignores readiness", ready: notReady, method: http.MethodGet, path: "/healthz", wantStatus: http.StatusOK, wantContentType: "application/json"},
		{name: "healthz wrong method", ready: ready, method: http.MethodPost, path: "/healthz", wantStatus: http.StatusMethodNotAllowed},
		{name: "readyz ready", ready: ready, method: http.MethodGet, path: "/readyz", wantStatus: http.StatusOK, wantContentType: "application/json"},
		{name: "readyz not ready", ready: notReady, method: http.MethodGet, path: "/readyz", wantStatus: http.StatusServiceUnavailable, wantContentType: "application/problem+json"},
		{name: "unknown path", ready: ready, method: http.MethodGet, path: "/nope", wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			NewRouter(tt.ready, discard()).ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantContentType != "" {
				if got := rec.Header().Get("Content-Type"); got != tt.wantContentType {
					t.Fatalf("Content-Type = %q, want %q", got, tt.wantContentType)
				}
			}
		})
	}
}

func TestHealthBody(t *testing.T) {
	for _, path := range []string{"/healthz", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			var got api.Health
			get(t, NewRouter(ready, discard()), path, &got)
			if got.Status != api.HealthStatusOk {
				t.Fatalf("status = %q, want %q", got.Status, api.HealthStatusOk)
			}
		})
	}
}

func TestReadyzProblem(t *testing.T) {
	var logs bytes.Buffer
	var got api.Problem
	get(t, NewRouter(notReady, slog.New(slog.NewTextHandler(&logs, nil))), "/readyz", &got)

	if got.Type != "about:blank" || got.Title != "Service Unavailable" || got.Status != http.StatusServiceUnavailable {
		t.Errorf("type/title/status = %q/%q/%d", got.Type, got.Title, got.Status)
	}
	if got.Code != api.ProblemCodeNOTREADY {
		t.Errorf("code = %q, want %q", got.Code, api.ProblemCodeNOTREADY)
	}
	if got.Detail == nil || strings.Contains(*got.Detail, "10.0.0.7") {
		t.Errorf("detail = %v, want a fixed text without internal details", got.Detail)
	}
	if got.RequestId == "" {
		t.Error("request_id is empty")
	}
	if !strings.Contains(logs.String(), "10.0.0.7") || !strings.Contains(logs.String(), got.RequestId) {
		t.Errorf("log = %q, want the cause and request_id %s", logs.String(), got.RequestId)
	}
}

// get serves a GET request and decodes the JSON body into v, rejecting fields
// that are not in the contract.
func get(t *testing.T, h http.Handler, path string, v any) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	dec := json.NewDecoder(rec.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		t.Fatalf("decode %s body: %v", path, err)
	}
}
