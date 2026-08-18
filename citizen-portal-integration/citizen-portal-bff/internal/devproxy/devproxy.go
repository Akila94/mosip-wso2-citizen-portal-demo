// Package devproxy serves the citizen-portal SPA from the BFF's own origin,
// so the browser only ever talks to one host (`http://localhost:8090` by
// default) and the SPA needs no CORS, no client id and no issuer URL of its
// own.
//
// It has two modes, chosen by configuration:
//
//   - DEV_PROXY_TARGET set — reverse-proxy every non-API request to the Vite
//     dev server, WebSocket upgrades included, so HMR works while the SPA is
//     being developed behind the real BFF.
//   - DEV_PROXY_TARGET empty — serve the built SPA from STATIC_DIR, falling
//     back to index.html for any path that is not a file, so React Router
//     deep links survive a hard refresh.
//
// The handler produced here is registered as the router's NotFound handler
// (internal/httpapi.Server.Router), so every registered /bff/... route is
// matched first and only genuinely unrouted requests arrive here.
package devproxy

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
)

// apiPathPrefix is the prefix every BFF API route lives under. Requests
// under it never reach the SPA: an unknown /bff/... path is a typo'd or
// stale API call, and answering it with index.html would turn a clear 404
// into a silent "the API returned HTML" mystery.
const apiPathPrefix = "/bff/"

// Config selects and configures the SPA serving mode. It mirrors the
// DEV_PROXY_TARGET / STATIC_DIR pair in internal/config.
type Config struct {
	// DevProxyTarget is the Vite dev server origin (for example
	// "http://localhost:5173"). Empty selects static serving.
	DevProxyTarget string
	// StaticDir is the directory holding the built SPA (Vite's dist/).
	// Used only when DevProxyTarget is empty.
	StaticDir string
	// Logger receives boundary and error events. Required.
	Logger *slog.Logger
}

// New builds the SPA handler for cfg.
//
// It fails rather than degrading: a malformed DEV_PROXY_TARGET, a missing
// STATIC_DIR or a STATIC_DIR without an index.html all return an error so
// the process refuses to start, instead of booting into a deployment where
// every page is an unexplained 404 at demo time.
func New(cfg Config) (http.Handler, error) {
	if cfg.Logger == nil {
		return nil, fmt.Errorf("devproxy: a logger is required")
	}

	var spa http.Handler
	var err error
	if cfg.DevProxyTarget != "" {
		spa, err = newReverseProxy(cfg.DevProxyTarget, cfg.Logger)
	} else {
		spa, err = newStaticHandler(cfg.StaticDir, cfg.Logger)
	}
	if err != nil {
		return nil, err
	}

	return guardAPIPaths(spa), nil
}

// guardAPIPaths answers any unmatched /bff/... request with a JSON 404
// instead of passing it to the SPA. Both modes need this: the static
// fallback would otherwise return index.html, and the dev proxy would
// forward the request to Vite, which returns index.html for the same reason.
func guardAPIPaths(spa http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, apiPathPrefix) {
			writeJSONNotFound(w)
			return
		}
		spa.ServeHTTP(w, r)
	})
}

// writeJSONNotFound emits the same {"error": ...} shape the BFF's other
// error responses use, so an SPA fetch of a mistyped endpoint sees a normal
// API error rather than a document.
//
// It restores the strict API Content-Security-Policy, because the router
// wraps this whole handler in the SPA policy (which is right for HTML and
// assets, and wrong for an API error response).
func writeJSONNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", security.APIContentSecurityPolicy)
	w.WriteHeader(http.StatusNotFound)
	// Encoding a two-field literal cannot fail; a write failure means the
	// client is already gone, and there is nothing further to send it.
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
}

// parseProxyTarget validates a dev-proxy target the same way config.Load
// does. The duplication is deliberate: this package must be safe to use
// directly (its own tests do), and a reverse proxy pointed at a scheme-less
// or relative URL fails in a way that is very hard to read at runtime.
func parseProxyTarget(raw string) (*url.URL, error) {
	target, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("devproxy: DEV_PROXY_TARGET is not a valid URL: %w", err)
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return nil, fmt.Errorf("devproxy: DEV_PROXY_TARGET %q must be an absolute http:// or https:// URL", raw)
	}
	if target.Host == "" {
		return nil, fmt.Errorf("devproxy: DEV_PROXY_TARGET %q has no host", raw)
	}
	return target, nil
}
