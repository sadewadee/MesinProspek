// Package auth turns the Kratos session of a request into a Principal
// (docs/04 §9). Identities live in Kratos; this package only asks Kratos
// who the session belongs to.
package auth

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Errors of Principal. The HTTP edge maps them to problems (docs/04 §8).
var (
	// ErrUnauthenticated: the request carries no valid session.
	ErrUnauthenticated = errors.New("auth: no valid session")
	// ErrAAL2Required: the account uses TOTP and the session has not passed
	// it yet (Kratos session.whoami.required_aal: highest_available).
	ErrAAL2Required = errors.New("auth: session needs the second factor")
	// ErrEmailNotVerified: the session is valid but its email address is not
	// verified yet (FR-AUTH-01).
	ErrEmailNotVerified = errors.New("auth: email address not verified")
	// ErrUnavailable: Kratos did not answer as expected.
	ErrUnavailable = errors.New("auth: kratos unavailable")
)

// Principal is the signed-in identity of a request.
type Principal struct {
	IdentityID uuid.UUID
	Email      string
	Verified   bool   // the email address is verified
	AAL        string // authenticator assurance level: aal1, or aal2 with TOTP
}

const (
	// cacheTTL is how long a verified session is trusted without asking
	// Kratos again (docs/04 §9).
	cacheTTL = 30 * time.Second
	// cacheMax bounds the cache; when full, expired entries are dropped,
	// then everything if that is not enough.
	cacheMax = 10_000
)

// Client asks the Kratos public API who a session belongs to. It caches
// verified sessions in process for cacheTTL, keyed by a hash of the Cookie
// header, so a cookie value is never kept in memory.
type Client struct {
	whoamiURL string
	http      *http.Client
	now       func() time.Time

	mu    sync.Mutex
	cache map[[sha256.Size]byte]cached
}

type cached struct {
	principal Principal
	expires   time.Time
}

// NewClient returns a Client for the Kratos public API at publicURL
// (MSP_KRATOS_PUBLIC_URL, e.g. http://kratos:4433).
func NewClient(publicURL string) *Client {
	return &Client{
		whoamiURL: strings.TrimSuffix(publicURL, "/") + "/sessions/whoami",
		http:      &http.Client{Timeout: 5 * time.Second},
		now:       time.Now,
		cache:     make(map[[sha256.Size]byte]cached),
	}
}

// Principal returns the identity of the Kratos session in cookie, the Cookie
// header of the request.
func (c *Client) Principal(ctx context.Context, cookie string) (Principal, error) {
	if cookie == "" {
		return Principal{}, ErrUnauthenticated
	}
	key := sha256.Sum256([]byte(cookie))
	if p, ok := c.cached(key); ok {
		return p, nil
	}
	p, err := c.whoami(ctx, cookie)
	if err != nil {
		return Principal{}, err
	}
	// Only verified sessions are cached: a user who has just verified the
	// email must not wait for an unverified entry to expire.
	if p.Verified {
		c.store(key, p)
	}
	return p, nil
}

func (c *Client) whoami(ctx context.Context, cookie string) (Principal, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.whoamiURL, nil)
	if err != nil {
		return Principal{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return Principal{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return Principal{}, ErrUnauthenticated
	case http.StatusForbidden:
		return Principal{}, ErrAAL2Required
	default:
		return Principal{}, fmt.Errorf("%w: whoami answered %d", ErrUnavailable, resp.StatusCode)
	}

	var s session
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&s); err != nil {
		return Principal{}, fmt.Errorf("%w: decode whoami: %w", ErrUnavailable, err)
	}
	if !s.Active {
		return Principal{}, ErrUnauthenticated
	}
	id, err := uuid.Parse(s.Identity.ID)
	if err != nil {
		return Principal{}, fmt.Errorf("%w: identity id: %w", ErrUnavailable, err)
	}
	p := Principal{IdentityID: id, Email: s.Identity.Traits.Email, AAL: s.AAL}
	for _, a := range s.Identity.VerifiableAddresses {
		if a.Via == "email" && a.Verified && strings.EqualFold(a.Value, p.Email) {
			p.Verified = true
		}
	}
	return p, nil
}

// session is the part of the Kratos whoami response that msp reads.
type session struct {
	Active   bool   `json:"active"`
	AAL      string `json:"authenticator_assurance_level"`
	Identity struct {
		ID     string `json:"id"`
		Traits struct {
			Email string `json:"email"`
		} `json:"traits"`
		VerifiableAddresses []struct {
			Value    string `json:"value"`
			Via      string `json:"via"`
			Verified bool   `json:"verified"`
		} `json:"verifiable_addresses"`
	} `json:"identity"`
}

func (c *Client) cached(key [sha256.Size]byte) (Principal, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.cache[key]
	if !ok || !c.now().Before(e.expires) {
		return Principal{}, false
	}
	return e.principal, true
}

func (c *Client) store(key [sha256.Size]byte, p Principal) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if len(c.cache) >= cacheMax {
		for k, e := range c.cache {
			if !now.Before(e.expires) {
				delete(c.cache, k)
			}
		}
		if len(c.cache) >= cacheMax {
			clear(c.cache)
		}
	}
	c.cache[key] = cached{principal: p, expires: now.Add(cacheTTL)}
}

type principalKey struct{}

// WithPrincipal returns ctx carrying p.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFrom returns the Principal that WithPrincipal put in ctx.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}
