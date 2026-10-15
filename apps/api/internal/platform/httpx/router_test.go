package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"msp/internal/api"
)

func ready(context.Context) error    { return nil }
func notReady(context.Context) error { return errors.New("database not configured") }

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
			NewRouter(tt.ready).ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
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
			get(t, NewRouter(ready), path, &got)
			if got.Status != api.HealthStatusOk {
				t.Fatalf("status = %q, want %q", got.Status, api.HealthStatusOk)
			}
		})
	}
}

func TestReadyzProblem(t *testing.T) {
	var got api.Problem
	get(t, NewRouter(notReady), "/readyz", &got)

	if got.Type != "about:blank" || got.Title != "Service Unavailable" || got.Status != http.StatusServiceUnavailable {
		t.Errorf("type/title/status = %q/%q/%d", got.Type, got.Title, got.Status)
	}
	if got.Code != api.ProblemCodeINTERNAL {
		t.Errorf("code = %q, want %q", got.Code, api.ProblemCodeINTERNAL)
	}
	if got.Detail == nil || *got.Detail != "database not configured" {
		t.Errorf("detail = %v, want %q", got.Detail, "database not configured")
	}
	if got.RequestId == "" {
		t.Error("request_id is empty")
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
