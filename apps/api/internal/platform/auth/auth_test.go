package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

const identityID = "8f9a0c1e-4b2d-4f6a-9c3e-1d2b3a4c5e6f"

// Bodies in the shape of Kratos GET /sessions/whoami (only the fields msp reads, plus noise).
const (
	verifiedSession = `{"id":"s1","active":true,"authenticator_assurance_level":"aal2",
		"identity":{"id":"` + identityID + `","state":"active","traits":{"email":"Ana@Example.com"},
		"verifiable_addresses":[{"value":"ana@example.com","via":"email","verified":true,"status":"completed"}]}}`
	unverifiedSession = `{"id":"s2","active":true,"authenticator_assurance_level":"aal1",
		"identity":{"id":"` + identityID + `","traits":{"email":"ana@example.com"},
		"verifiable_addresses":[{"value":"ana@example.com","via":"email","verified":false,"status":"sent"}]}}`
	inactiveSession = `{"id":"s3","active":false,"identity":{"id":"` + identityID + `","traits":{"email":"ana@example.com"}}}`
)

// fakeKratos answers whoami with status and body, counts calls and records
// the forwarded Cookie header.
type fakeKratos struct {
	*httptest.Server
	calls  atomic.Int32
	cookie atomic.Value
}

func newFakeKratos(t *testing.T, status int, body string) *fakeKratos {
	t.Helper()
	f := &fakeKratos{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		f.cookie.Store(r.Header.Get("Cookie"))
		if r.Method != http.MethodGet || r.URL.Path != "/sessions/whoami" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(f.Close)
	return f
}

func TestPrincipal(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		cookie  string
		want    Principal
		wantErr error
		calls   int32
	}{
		{
			name: "valid session", status: http.StatusOK, body: verifiedSession, cookie: "ory_kratos_session=abc",
			want:  Principal{IdentityID: uuid.MustParse(identityID), Email: "Ana@Example.com", Verified: true, AAL: "aal2"},
			calls: 1,
		},
		{
			name: "unverified email", status: http.StatusOK, body: unverifiedSession, cookie: "ory_kratos_session=abc",
			want:  Principal{IdentityID: uuid.MustParse(identityID), Email: "ana@example.com", AAL: "aal1"},
			calls: 1,
		},
		{name: "no cookie, kratos not asked", status: http.StatusOK, body: verifiedSession, cookie: "", wantErr: ErrUnauthenticated},
		{name: "no valid session", status: http.StatusUnauthorized, body: `{"error":{"code":401}}`, cookie: "x=1", wantErr: ErrUnauthenticated, calls: 1},
		{name: "inactive session", status: http.StatusOK, body: inactiveSession, cookie: "x=1", wantErr: ErrUnauthenticated, calls: 1},
		{name: "second factor required", status: http.StatusForbidden, body: `{"error":{"id":"session_aal2_required"}}`, cookie: "x=1", wantErr: ErrAAL2Required, calls: 1},
		{name: "kratos error", status: http.StatusInternalServerError, body: `{}`, cookie: "x=1", wantErr: ErrUnavailable, calls: 1},
		{name: "malformed body", status: http.StatusOK, body: `{"active":`, cookie: "x=1", wantErr: ErrUnavailable, calls: 1},
		{name: "malformed identity id", status: http.StatusOK, body: `{"active":true,"identity":{"id":"nope"}}`, cookie: "x=1", wantErr: ErrUnavailable, calls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kratos := newFakeKratos(t, tt.status, tt.body)
			got, err := NewClient(kratos.URL+"/").Principal(t.Context(), tt.cookie)
			if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil && err != nil) {
				t.Fatalf("Principal() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("Principal() = %+v, want %+v", got, tt.want)
			}
			if n := kratos.calls.Load(); n != tt.calls {
				t.Fatalf("whoami calls = %d, want %d", n, tt.calls)
			}
			if tt.calls > 0 && kratos.cookie.Load() != tt.cookie {
				t.Fatalf("forwarded Cookie = %q, want %q", kratos.cookie.Load(), tt.cookie)
			}
		})
	}
}

func TestPrincipalKratosDown(t *testing.T) {
	kratos := newFakeKratos(t, http.StatusOK, verifiedSession)
	kratos.Close()
	_, err := NewClient(kratos.URL).Principal(t.Context(), "ory_kratos_session=abc")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Principal() error = %v, want %v", err, ErrUnavailable)
	}
}

func TestPrincipalCache(t *testing.T) {
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	newClient := func(url string) *Client {
		c := NewClient(url)
		c.now = func() time.Time { return now }
		return c
	}

	t.Run("verified session is cached for 30 s per cookie", func(t *testing.T) {
		kratos := newFakeKratos(t, http.StatusOK, verifiedSession)
		c := newClient(kratos.URL)
		ask := func(cookie string, wantCalls int32) {
			t.Helper()
			if _, err := c.Principal(t.Context(), cookie); err != nil {
				t.Fatal(err)
			}
			if n := kratos.calls.Load(); n != wantCalls {
				t.Fatalf("whoami calls = %d, want %d", n, wantCalls)
			}
		}
		ask("ory_kratos_session=a", 1)
		ask("ory_kratos_session=a", 1) // cache hit
		ask("ory_kratos_session=b", 2) // other cookie, other entry
		now = now.Add(29 * time.Second)
		ask("ory_kratos_session=a", 2)
		now = now.Add(time.Second)
		ask("ory_kratos_session=a", 3) // expired after 30 s
	})

	t.Run("unverified session is not cached", func(t *testing.T) {
		kratos := newFakeKratos(t, http.StatusOK, unverifiedSession)
		c := newClient(kratos.URL)
		for range 2 {
			if _, err := c.Principal(t.Context(), "ory_kratos_session=a"); err != nil {
				t.Fatal(err)
			}
		}
		if n := kratos.calls.Load(); n != 2 {
			t.Fatalf("whoami calls = %d, want 2", n)
		}
	})

	t.Run("errors are not cached", func(t *testing.T) {
		kratos := newFakeKratos(t, http.StatusUnauthorized, `{}`)
		c := newClient(kratos.URL)
		for range 2 {
			if _, err := c.Principal(t.Context(), "ory_kratos_session=a"); !errors.Is(err, ErrUnauthenticated) {
				t.Fatalf("Principal() error = %v, want %v", err, ErrUnauthenticated)
			}
		}
		if n := kratos.calls.Load(); n != 2 {
			t.Fatalf("whoami calls = %d, want 2", n)
		}
	})

	t.Run("a full cache drops expired entries first", func(t *testing.T) {
		c := newClient("http://unused")
		p := Principal{Verified: true}
		for i := range cacheMax {
			c.store([32]byte{byte(i), byte(i >> 8)}, p)
		}
		now = now.Add(cacheTTL)
		c.store([32]byte{0xff, 0xff, 0xff}, p)
		if len(c.cache) != 1 {
			t.Fatalf("cache size = %d, want 1 after dropping expired entries", len(c.cache))
		}
	})
}

func TestPrincipalContext(t *testing.T) {
	if _, ok := PrincipalFrom(context.Background()); ok {
		t.Fatal("PrincipalFrom(empty context) ok = true")
	}
	want := Principal{IdentityID: uuid.MustParse(identityID), Email: "ana@example.com", Verified: true, AAL: "aal1"}
	if got, ok := PrincipalFrom(WithPrincipal(context.Background(), want)); !ok || got != want {
		t.Fatalf("PrincipalFrom() = %+v, %t; want %+v", got, ok, want)
	}
}
