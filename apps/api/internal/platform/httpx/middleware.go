package httpx

import (
	"context"
	"mime"
	"net/http"

	"msp/internal/api"
	"msp/internal/platform/auth"
)

// authenticate is the session middleware (docs/04 §9). Every operation needs
// a Kratos session and a verified email (FR-AUTH-01), except the public
// health checks and GetMe, which needs a session only (role "login", docs/04
// §8) so the web app can tell an unverified user to verify.
func (s server) authenticate(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
	switch operationID {
	case "GetHealthz", "GetReadyz":
		return next
	}
	needsVerifiedEmail := operationID != "GetMe"
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		p, err := s.sessions.Principal(ctx, r.Header.Get("Cookie"))
		if err != nil {
			return nil, err
		}
		// TODO(WP-10): create the users row just in time (docs/04 §9) once
		// the users table exists; until then msp keeps no user data.
		if needsVerifiedEmail && !p.Verified {
			return nil, auth.ErrEmailNotVerified
		}
		return next(auth.WithPrincipal(ctx, p), w, r, request)
	}
}

// sameOrigin is the CSRF check for state-changing requests (docs/04 §9): the
// Kratos session cookie is SameSite=Lax, Origin must be the public origin,
// and a request with a body must send it as JSON. A request without a body
// (e.g. DELETE) needs only the Origin. Kratos flows have their own CSRF
// tokens and never reach msp api.
func (s server) sameOrigin(publicURL string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		hasBody := r.ContentLength != 0 // -1 = unknown length, so a body
		if r.Header.Get("Origin") != publicURL || (hasBody && !isJSON(r.Header.Get("Content-Type"))) {
			s.writeError(w, r, errCrossSite)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isJSON(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/json"
}

// routes serves mux and answers its own misses as problems: ServeMux would
// write plain-text 404 and 405 responses.
func (s server) routes(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, pattern := mux.Handler(r)
		if pattern != "" {
			mux.ServeHTTP(w, r) // sets the path values the generated wrappers read
			return
		}
		// No route matched. Run ServeMux's own answer against a probe to learn
		// whether it is a 404, a 405 (with Allow) or a redirect to a clean path.
		probe := &probeWriter{header: http.Header{}}
		h.ServeHTTP(probe, r)
		switch probe.status {
		case http.StatusNotFound:
			s.writeError(w, r, errNotFound)
		case http.StatusMethodNotAllowed:
			w.Header().Set("Allow", probe.header.Get("Allow"))
			s.writeError(w, r, errMethodNotAllowed)
		default:
			mux.ServeHTTP(w, r)
		}
	})
}

// probeWriter records the status and headers of a response and drops its body.
type probeWriter struct {
	header http.Header
	status int
}

func (p *probeWriter) Header() http.Header { return p.header }

func (p *probeWriter) Write(b []byte) (int, error) {
	if p.status == 0 {
		p.status = http.StatusOK
	}
	return len(b), nil
}

func (p *probeWriter) WriteHeader(status int) {
	if p.status == 0 {
		p.status = status
	}
}
