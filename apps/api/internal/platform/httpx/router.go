// Package httpx holds the HTTP router and edge concerns of `msp api`.
package httpx

import (
	"context"
	"crypto/rand"
	"net/http"

	"msp/internal/api"
)

// ReadyChecker reports whether `msp api` can serve traffic. A non-nil error
// makes /readyz answer 503; its text becomes the problem detail, so it must
// not carry internal details such as hosts or credentials.
type ReadyChecker func(ctx context.Context) error

// NewRouter returns the handler for `msp api`. All routes come from the
// generated contract (contracts/openapi.yaml); routes outside /api/v1 are
// limited to health checks (docs/04 §8).
func NewRouter(ready ReadyChecker) http.Handler {
	return api.Handler(api.NewStrictHandler(server{ready: ready}, nil))
}

// server implements api.StrictServerInterface.
type server struct {
	ready ReadyChecker
}

// GetHealthz reports that the process is alive. It has no dependencies;
// readiness belongs to GetReadyz.
func (server) GetHealthz(context.Context, api.GetHealthzRequestObject) (api.GetHealthzResponseObject, error) {
	return api.GetHealthz200JSONResponse{Status: api.HealthStatusOk}, nil
}

// GetReadyz reports whether the process can serve traffic.
func (s server) GetReadyz(ctx context.Context, _ api.GetReadyzRequestObject) (api.GetReadyzResponseObject, error) {
	if err := s.ready(ctx); err != nil {
		p := newProblem(http.StatusServiceUnavailable, api.ProblemCodeINTERNAL, err.Error())
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
