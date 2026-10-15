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
	"sync/atomic"
	"testing"

	"msp/internal/api"
	"msp/internal/platform/auth"
)

const publicURL = "https://app.example.com"

func ready(context.Context) error { return nil }
func notReady(context.Context) error {
	return errors.New("db: not ready: dial 10.0.0.7:5432 as msp_app")
}

func discard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// Kratos whoami bodies (docs/05: Kratos is faked in tests).
const (
	verified = `{"active":true,"authenticator_assurance_level":"aal1","identity":{"id":"8f9a0c1e-4b2d-4f6a-9c3e-1d2b3a4c5e6f",
		"traits":{"email":"ana@example.com"},"verifiable_addresses":[{"value":"ana@example.com","via":"email","verified":true}]}}`
	unverified = `{"active":true,"authenticator_assurance_level":"aal1","identity":{"id":"8f9a0c1e-4b2d-4f6a-9c3e-1d2b3a4c5e6f",
		"traits":{"email":"ana@example.com"},"verifiable_addresses":[{"value":"ana@example.com","via":"email","verified":false}]}}`
)

// fakeKratos serves GET /sessions/whoami: body for the cookie
// "ory_kratos_session=valid", 401 otherwise. It counts calls.
func fakeKratos(t *testing.T, status int, body string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Cookie") != "ory_kratos_session=valid" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

// newRouter returns the router with a fake Kratos answering status/body.
func newRouter(t *testing.T, check ReadyChecker, status int, body string) (http.Handler, *atomic.Int32) {
	t.Helper()
	kratos, calls := fakeKratos(t, status, body)
	return NewRouter(check, auth.NewClient(kratos.URL), publicURL, discard()), calls
}

func serve(h http.Handler, method, path string, header map[string]string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body == "" {
		req.ContentLength = 0
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// decode decodes a JSON body into v, rejecting fields that are not in the contract.
func decode(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(rec.Body.Bytes()))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
}

// problem asserts a problem+json response with status and code and returns it.
func problem(t *testing.T, rec *httptest.ResponseRecorder, status int, code api.ProblemCode) api.Problem {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, status, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json", ct)
	}
	var p api.Problem
	decode(t, rec, &p)
	if p.Code != code || p.Status != status || p.Type != "about:blank" || p.Title != http.StatusText(status) || p.RequestId == "" {
		t.Fatalf("problem = %+v, want status %d and code %s", p, status, code)
	}
	return p
}

func TestHealth(t *testing.T) {
	tests := []struct {
		name       string
		ready      ReadyChecker
		path       string
		wantStatus int
	}{
		{name: "healthz ignores readiness", ready: notReady, path: "/healthz", wantStatus: http.StatusOK},
		{name: "readyz ready", ready: ready, path: "/readyz", wantStatus: http.StatusOK},
		{name: "readyz not ready", ready: notReady, path: "/readyz", wantStatus: http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// No session: health checks are public and never ask Kratos.
			h, calls := newRouter(t, tt.ready, http.StatusOK, verified)
			rec := serve(h, http.MethodGet, tt.path, nil, "")
			if tt.wantStatus != http.StatusOK {
				problem(t, rec, tt.wantStatus, api.ProblemCodeNOTREADY)
				return
			}
			var got api.Health
			decode(t, rec, &got)
			if rec.Code != http.StatusOK || got.Status != api.HealthStatusOk {
				t.Fatalf("%s = %d %+v, want 200 ok", tt.path, rec.Code, got)
			}
			if calls.Load() != 0 {
				t.Fatalf("whoami calls = %d, want 0", calls.Load())
			}
		})
	}
}

func TestReadyzProblem(t *testing.T) {
	var logs bytes.Buffer
	kratos, _ := fakeKratos(t, http.StatusOK, verified)
	h := NewRouter(notReady, auth.NewClient(kratos.URL), publicURL, slog.New(slog.NewTextHandler(&logs, nil)))
	got := problem(t, serve(h, http.MethodGet, "/readyz", nil, ""), http.StatusServiceUnavailable, api.ProblemCodeNOTREADY)
	if got.Detail == nil || strings.Contains(*got.Detail, "10.0.0.7") {
		t.Errorf("detail = %v, want a fixed text without internal details", got.Detail)
	}
	if !strings.Contains(logs.String(), "10.0.0.7") || !strings.Contains(logs.String(), got.RequestId) {
		t.Errorf("log = %q, want the cause and request_id %s", logs.String(), got.RequestId)
	}
}

func TestMe(t *testing.T) {
	session := map[string]string{"Cookie": "ory_kratos_session=valid"}
	tests := []struct {
		name         string
		status       int
		body         string
		header       map[string]string
		wantStatus   int
		wantCode     api.ProblemCode
		wantVerified bool
	}{
		{name: "valid session", status: http.StatusOK, body: verified, header: session, wantStatus: http.StatusOK, wantVerified: true},
		{name: "unverified email still sees itself", status: http.StatusOK, body: unverified, header: session, wantStatus: http.StatusOK},
		{name: "no cookie", status: http.StatusOK, body: verified, wantStatus: http.StatusUnauthorized, wantCode: api.ProblemCodeUNAUTHENTICATED},
		{name: "invalid session", status: http.StatusOK, body: verified, header: map[string]string{"Cookie": "ory_kratos_session=expired"}, wantStatus: http.StatusUnauthorized, wantCode: api.ProblemCodeUNAUTHENTICATED},
		{name: "second factor required", status: http.StatusForbidden, body: `{}`, header: session, wantStatus: http.StatusForbidden, wantCode: api.ProblemCodeMFAREQUIRED},
		{name: "kratos error", status: http.StatusBadGateway, body: `{}`, header: session, wantStatus: http.StatusServiceUnavailable, wantCode: api.ProblemCodeNOTREADY},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newRouter(t, ready, tt.status, tt.body)
			rec := serve(h, http.MethodGet, "/api/v1/me", tt.header, "")
			if tt.wantCode != "" {
				problem(t, rec, tt.wantStatus, tt.wantCode)
				return
			}
			var got api.Me
			decode(t, rec, &got)
			want := api.Me{IdentityId: "8f9a0c1e-4b2d-4f6a-9c3e-1d2b3a4c5e6f", Email: "ana@example.com", Verified: tt.wantVerified, Aal: api.MeAalAal1}
			if rec.Code != http.StatusOK || got != want {
				t.Fatalf("GET /api/v1/me = %d %+v, want 200 %+v", rec.Code, got, want)
			}
		})
	}
}

func TestMeKratosDown(t *testing.T) {
	kratos, _ := fakeKratos(t, http.StatusOK, verified)
	kratos.Close()
	var logs bytes.Buffer
	h := NewRouter(ready, auth.NewClient(kratos.URL), publicURL, slog.New(slog.NewTextHandler(&logs, nil)))
	rec := serve(h, http.MethodGet, "/api/v1/me", map[string]string{"Cookie": "ory_kratos_session=valid"}, "")
	problem(t, rec, http.StatusServiceUnavailable, api.ProblemCodeNOTREADY)
	if host := strings.TrimPrefix(kratos.URL, "http://"); strings.Contains(rec.Body.String(), host) {
		t.Fatalf("body %s leaks the Kratos address", rec.Body.String())
	}
	if !strings.Contains(logs.String(), "level=ERROR") {
		t.Fatalf("log = %q, want the cause at error level", logs.String())
	}
}

func TestMeCache(t *testing.T) {
	h, calls := newRouter(t, ready, http.StatusOK, verified)
	for range 3 {
		if rec := serve(h, http.MethodGet, "/api/v1/me", map[string]string{"Cookie": "ory_kratos_session=valid"}, ""); rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("whoami calls = %d, want 1 (cache hit)", calls.Load())
	}
}

// Every operation except GetMe and the health checks needs a verified email.
// Until WP-10 adds such an operation, the middleware is tested on its own.
func TestAuthenticateNeedsVerifiedEmail(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		operation string
		wantErr   error
	}{
		{name: "data operation, unverified", body: unverified, operation: "ListProspects", wantErr: auth.ErrEmailNotVerified},
		{name: "data operation, verified", body: verified, operation: "ListProspects"},
		{name: "me, unverified", body: unverified, operation: "GetMe"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kratos, _ := fakeKratos(t, http.StatusOK, tt.body)
			s := server{sessions: auth.NewClient(kratos.URL), logger: discard()}
			var got auth.Principal
			next := func(ctx context.Context, _ http.ResponseWriter, _ *http.Request, _ any) (any, error) {
				got, _ = auth.PrincipalFrom(ctx)
				return "ok", nil
			}
			req := httptest.NewRequest(http.MethodGet, "/api/v1/x", nil)
			req.Header.Set("Cookie", "ory_kratos_session=valid")
			res, err := s.authenticate(next, tt.operation)(req.Context(), httptest.NewRecorder(), req, nil)
			if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil && err != nil) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				rec := httptest.NewRecorder()
				s.writeError(rec, req, err)
				problem(t, rec, http.StatusForbidden, api.ProblemCodeEMAILNOTVERIFIED)
				return
			}
			if res != "ok" || got.Email != "ana@example.com" {
				t.Fatalf("next got principal %+v, result %v; want it called with the principal", got, res)
			}
		})
	}
}

func TestSameOrigin(t *testing.T) {
	fromApp := map[string]string{"Origin": publicURL, "Content-Type": "application/json; charset=utf-8"}
	tests := []struct {
		name       string
		method     string
		header     map[string]string
		body       string
		wantStatus int // 405: the CSRF check passed and the router answered (only GET /api/v1/me exists)
	}{
		{name: "GET needs no origin", method: http.MethodGet, wantStatus: http.StatusUnauthorized},
		{name: "POST without origin", method: http.MethodPost, header: map[string]string{"Content-Type": "application/json"}, body: `{}`, wantStatus: http.StatusForbidden},
		{name: "POST from another origin", method: http.MethodPost, header: map[string]string{"Origin": "https://evil.example", "Content-Type": "application/json"}, body: `{}`, wantStatus: http.StatusForbidden},
		{name: "POST of a form", method: http.MethodPost, header: map[string]string{"Origin": publicURL, "Content-Type": "application/x-www-form-urlencoded"}, body: "a=1", wantStatus: http.StatusForbidden},
		{name: "POST of text/plain", method: http.MethodPost, header: map[string]string{"Origin": publicURL, "Content-Type": "text/plain"}, body: `{}`, wantStatus: http.StatusForbidden},
		{name: "POST of JSON from the app", method: http.MethodPost, header: fromApp, body: `{}`, wantStatus: http.StatusMethodNotAllowed},
		{name: "DELETE without body from the app", method: http.MethodDelete, header: map[string]string{"Origin": publicURL}, wantStatus: http.StatusMethodNotAllowed},
		{name: "DELETE without body from another origin", method: http.MethodDelete, header: map[string]string{"Origin": "null"}, wantStatus: http.StatusForbidden},
		{name: "PATCH of JSON from the app", method: http.MethodPatch, header: fromApp, body: `{}`, wantStatus: http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newRouter(t, ready, http.StatusOK, verified)
			rec := serve(h, tt.method, "/api/v1/me", tt.header, tt.body)
			code := map[int]api.ProblemCode{
				http.StatusUnauthorized:     api.ProblemCodeUNAUTHENTICATED,
				http.StatusForbidden:        api.ProblemCodeFORBIDDEN,
				http.StatusMethodNotAllowed: api.ProblemCodeMETHODNOTALLOWED,
			}[tt.wantStatus]
			problem(t, rec, tt.wantStatus, code)
		})
	}
}

func TestUnmatchedRoutes(t *testing.T) {
	h, _ := newRouter(t, ready, http.StatusOK, verified)

	problem(t, serve(h, http.MethodGet, "/nope", nil, ""), http.StatusNotFound, api.ProblemCodeNOTFOUND)
	problem(t, serve(h, http.MethodGet, "/api/v1/nope", nil, ""), http.StatusNotFound, api.ProblemCodeNOTFOUND)

	rec := serve(h, http.MethodPut, "/healthz", map[string]string{"Origin": publicURL}, "")
	problem(t, rec, http.StatusMethodNotAllowed, api.ProblemCodeMETHODNOTALLOWED)
	if allow := rec.Header().Get("Allow"); !strings.Contains(allow, http.MethodGet) {
		t.Fatalf("Allow = %q, want GET", allow)
	}

	// ServeMux still redirects to the clean path.
	if rec := serve(h, http.MethodGet, "/api/v1/../healthz", nil, ""); rec.Code != http.StatusMovedPermanently {
		t.Fatalf("unclean path status = %d, want 301", rec.Code)
	}
}

func TestWriteErrorHidesInternalErrors(t *testing.T) {
	var logs bytes.Buffer
	s := server{logger: slog.New(slog.NewTextHandler(&logs, nil))}
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   api.ProblemCode
	}{
		{name: "unknown error", err: errors.New("pq: password authentication failed for user msp_app"), wantStatus: http.StatusInternalServerError, wantCode: api.ProblemCodeINTERNAL},
		{name: "request error", err: errors.Join(errBadRequest, errors.New("json: cannot unmarshal into Go struct field secretField")), wantStatus: http.StatusUnprocessableEntity, wantCode: api.ProblemCodeVALIDATIONFAILED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			s.writeError(rec, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil), tt.err)
			problem(t, rec, tt.wantStatus, tt.wantCode)
			if strings.Contains(rec.Body.String(), "msp_app") || strings.Contains(rec.Body.String(), "secretField") {
				t.Fatalf("body %s leaks the error text", rec.Body.String())
			}
		})
	}
	if !strings.Contains(logs.String(), "password authentication failed") {
		t.Fatalf("log = %q, want the internal error logged", logs.String())
	}
}
