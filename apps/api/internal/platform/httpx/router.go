// Package httpx holds the HTTP router and edge concerns of `msp api`.
package httpx

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"

	"msp/internal/api"
)

// ReadyChecker reports whether `msp api` can serve traffic. A non-nil error
// makes /readyz answer 503 NOT_READY. The error is logged, never returned
// to the client, so it may carry internal details.
type ReadyChecker func(ctx context.Context) error

// NewRouter returns the handler for `msp api`. All routes come from the
// generated contract (contracts/openapi.yaml); routes outside /api/v1 are
// limited to health checks (docs/04 §8).
func NewRouter(ready ReadyChecker, logger *slog.Logger) http.Handler {
	return api.Handler(api.NewStrictHandler(server{ready: ready, logger: logger}, nil))
}

// server implements api.StrictServerInterface.
type server struct {
	ready  ReadyChecker
	logger *slog.Logger
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

// newProblem builds an RFC 9457 problem (docs/04 §8). The request ID is minted
// here until request logging exists; then it moves to the middleware that
// logs it, so both carry the same ID.
func newProblem(status int, code api.ProblemCode, detail string) api.Problem {
	return api.Problem{
		Type:      "about:blank",
		Title:     http.StatusText(status),
		Status:    status,
		Code:      code,
		Detail:    &detail,
		RequestId: rand.Text(),
	}
}
