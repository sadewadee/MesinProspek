package httpx

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"msp/internal/api"
	"msp/internal/platform/auth"
)

// Errors raised at the HTTP edge itself.
var (
	errBadRequest       = errors.New("request does not match the contract")
	errNotFound         = errors.New("no such route")
	errMethodNotAllowed = errors.New("method not allowed on this route")
	errCrossSite        = errors.New("state-changing request from another origin or without a JSON body")
)

// writeError is the one mapping from errors to problem+json (STD-GO-02).
// Errors it does not know become 500 INTERNAL without their text, which is
// logged with the request_id instead.
func (s server) writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, detail := http.StatusInternalServerError, api.ProblemCodeINTERNAL, "internal error"
	switch {
	case errors.Is(err, errBadRequest):
		status, code, detail = http.StatusUnprocessableEntity, api.ProblemCodeVALIDATIONFAILED, "the request does not match the API contract"
	case errors.Is(err, errNotFound):
		status, code, detail = http.StatusNotFound, api.ProblemCodeNOTFOUND, "no such resource"
	case errors.Is(err, errMethodNotAllowed):
		status, code, detail = http.StatusMethodNotAllowed, api.ProblemCodeMETHODNOTALLOWED, "method not allowed"
	case errors.Is(err, errCrossSite):
		status, code, detail = http.StatusForbidden, api.ProblemCodeFORBIDDEN, "state-changing requests need Origin = MSP_PUBLIC_URL and a JSON body"
	case errors.Is(err, auth.ErrUnauthenticated):
		status, code, detail = http.StatusUnauthorized, api.ProblemCodeUNAUTHENTICATED, "sign in first"
	case errors.Is(err, auth.ErrEmailNotVerified):
		status, code, detail = http.StatusForbidden, api.ProblemCodeEMAILNOTVERIFIED, "verify the email address first"
	case errors.Is(err, auth.ErrAAL2Required):
		status, code, detail = http.StatusForbidden, api.ProblemCodeMFAREQUIRED, "complete the second factor (TOTP) first"
	case errors.Is(err, auth.ErrUnavailable):
		status, code, detail = http.StatusServiceUnavailable, api.ProblemCodeNOTREADY, "sign-in is temporarily unavailable"
	}

	p := newProblem(status, code, detail)
	level := slog.LevelDebug
	if status >= http.StatusInternalServerError {
		level = slog.LevelError
	}
	s.logger.Log(r.Context(), level, "request failed", "error", err, "request_id", p.RequestId,
		"method", r.Method, "route", r.Pattern, "status", status)
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
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
