// Package httpx holds the HTTP router and edge concerns of `msp api`.
package httpx

import "net/http"

// NewRouter returns the handler for `msp api`. Routes outside /api/v1 are
// limited to health checks (docs/04 §8).
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	return mux
}

// healthz reports that the process is alive. It has no dependencies; database
// readiness belongs to /readyz.
func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
