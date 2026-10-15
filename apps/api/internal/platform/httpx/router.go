// Package httpx holds the HTTP router and edge concerns of `msp api`.
package httpx

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"msp/internal/api"
	"msp/internal/platform/auth"
)

// ReadyChecker reports whether `msp api` can serve traffic. A non-nil error
// makes /readyz answer 503 NOT_READY. The error is logged, never returned
// to the client, so it may carry internal details.
type ReadyChecker func(ctx context.Context) error

// NewRouter returns the handler for `msp api`. All routes come from the
// generated contract (contracts/openapi.yaml); routes outside /api/v1 are
// limited to health checks (docs/04 §8). Operations need a Kratos session
// (sessions) unless they are public, and state-changing requests must come
// from publicURL (docs/04 §9). Every error, including unknown routes, is
// answered as problem+json by writeError.
func NewRouter(ready ReadyChecker, sessions *auth.Client, publicURL string, logger *slog.Logger) http.Handler {
	s := server{ready: ready, sessions: sessions, logger: logger}
	badRequest := func(w http.ResponseWriter, r *http.Request, err error) {
		s.writeError(w, r, fmt.Errorf("%w: %w", errBadRequest, err))
	}
	strict := api.NewStrictHandlerWithOptions(s, []api.StrictMiddlewareFunc{s.authenticate}, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  badRequest,
		ResponseErrorHandlerFunc: s.writeError,
	})
	mux := http.NewServeMux()
	api.HandlerWithOptions(strict, api.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: badRequest})
	return s.sameOrigin(publicURL, s.routes(mux))
}

// server implements api.StrictServerInterface.
type server struct {
	ready    ReadyChecker
	sessions *auth.Client
	logger   *slog.Logger
}

// GetHealthz reports that the process is alive. It has no dependencies;
// readiness belongs to GetReadyz.
func (server) GetHealthz(context.Context, api.GetHealthzRequestObject) (api.GetHealthzResponseObject, error) {
	return api.GetHealthz200JSONResponse{Status: api.HealthStatusOk}, nil
}

// GetReadyz reports whether the process can serve traffic.
func (s server) GetReadyz(ctx context.Context, _ api.GetReadyzRequestObject) (api.GetReadyzResponseObject, error) {
	if err := s.ready(ctx); err != nil {
		p := newProblem(http.StatusServiceUnavailable, api.ProblemCodeNOTREADY, "the service is not ready")
		s.logger.WarnContext(ctx, "not ready", "error", err, "request_id", p.RequestId)
		return api.GetReadyz503ApplicationProblemPlusJSONResponse(p), nil
	}
	return api.GetReadyz200JSONResponse{Status: api.HealthStatusOk}, nil
}

// GetMe returns the identity of the session (placeholder until WP-10).
func (server) GetMe(ctx context.Context, _ api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	p, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return nil, auth.ErrUnauthenticated
	}
	return api.GetMe200JSONResponse{
		IdentityId: p.IdentityID.String(),
		Email:      p.Email,
		Verified:   p.Verified,
		Aal:        api.MeAal(p.AAL),
	}, nil
}
