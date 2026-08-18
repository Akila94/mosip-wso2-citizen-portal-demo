package devproxy

import (
	"log/slog"
	"net/http"
	"net/http/httputil"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
)

// newReverseProxy builds the development-mode handler: everything that is
// not a BFF API route is forwarded to the Vite dev server.
//
// httputil.ReverseProxy is used rather than a hand-rolled forwarder because
// it already implements the one requirement that is easy to get wrong —
// protocol upgrades. When the client sends `Connection: Upgrade`, it
// forwards the Upgrade headers, recognises the upstream's 101 Switching
// Protocols response, hijacks the client connection and copies bytes in both
// directions afterwards. That is exactly what Vite's HMR WebSocket needs.
func newReverseProxy(rawTarget string, logger *slog.Logger) (http.Handler, error) {
	target, err := parseProxyTarget(rawTarget)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// The dev server is not a trusted source of security headers. Anything
	// it sets is dropped so the BFF's own headers (set by the middleware
	// chain before the request got here) are the only ones on the response
	// — otherwise a duplicate Content-Security-Policy would silently
	// intersect with, and could contradict, the policy this BFF chose.
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Content-Security-Policy")
		resp.Header.Del("Content-Security-Policy-Report-Only")
		resp.Header.Del("X-Frame-Options")
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, proxyErr error) {
		logger.Error("dev proxy could not reach the SPA dev server",
			"target", target.Redacted(),
			"path", security.SanitizeForLog(r.URL.Path),
			"error", security.SanitizeForLog(proxyErr.Error()))
		http.Error(w, "SPA dev server unavailable", http.StatusBadGateway)
	}

	logger.Info("serving the SPA by reverse proxy", "target", target.Redacted())
	return proxy, nil
}
