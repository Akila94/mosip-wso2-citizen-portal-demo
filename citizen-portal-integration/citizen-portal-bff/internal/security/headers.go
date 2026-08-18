package security

import (
	"net/http"
	"strings"
)

// APIContentSecurityPolicy is the policy for the BFF's own /bff/... JSON
// responses. They embed no markup, load no subresources and are never
// framed, so nothing beyond same-origin is permitted.
const APIContentSecurityPolicy = "default-src 'self'; frame-ancestors 'none'"

// SecurityHeaders sets a conservative baseline of response headers on every
// request: no framing (clickjacking), no MIME sniffing, no referrer leakage,
// and a same-origin-only CSP — appropriate for a BFF that renders no
// third-party content and expects no inline scripts.
//
// The SPA is served from this same origin (see internal/devproxy) and needs
// a looser — but still tightly bounded — policy; SPAHeaders layers over this
// middleware to replace the CSP on those responses only.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", APIContentSecurityPolicy)
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// spaCSPDirectives lists the SPA's policy in source order. Each directive
// carries the reason it is what it is, because every one of them is a place
// where a well-meaning "tidy-up" would either break the demo or weaken it.
//
// devMode is true only when DEV_PROXY_TARGET is set (the Vite dev server);
// the extra relaxations it enables must never reach a static/production
// deployment, which is why the two policies are built here together rather
// than by mutating one into the other at request time.
func spaCSPDirectives(devMode bool) []string {
	// The React screens set element style attributes (style={{...}}) in 650
	// places across 80 files; CSP blocks those without 'unsafe-inline' in
	// style-src. This is a property of the vendored WSO2 Design System
	// screens, not something the BFF can nonce away.
	styleSrc := "style-src 'self' 'unsafe-inline'"

	// eSignet returns the citizen's photo in the `picture` claim as a
	// data:image/jpeg;base64 URI (setup-without-bridge/demo.sh's
	// create_citizen posts exactly that), so the header avatar needs data:.
	imgSrc := "img-src 'self' data:"

	// Fonts are bundled by Vite and served from this origin. The design
	// system's fonts.css also @imports Google Fonts, but that @import sits
	// after an @font-face rule, so it is invalid per the CSS spec and is
	// dropped — verified: the built dist/assets/*.css contains no
	// fonts.googleapis.com or fonts.gstatic.com reference at all. If that
	// @import is ever moved to the top of the file, this directive (and
	// style-src) must gain the corresponding hosts.
	fontSrc := "font-src 'self'"

	scriptSrc := "script-src 'self'"
	// The SPA only ever calls same-origin /bff/... paths — it holds no
	// issuer URL, client id or token, which is the whole point of the BFF
	// pattern.
	connectSrc := "connect-src 'self'"

	if devMode {
		// Vite injects an inline module preamble into index.html and its
		// dev-time transforms rely on eval; both are dev-server artifacts
		// that never appear in a production build.
		scriptSrc = "script-src 'self' 'unsafe-inline' 'unsafe-eval'"
		// Vite's HMR client opens a WebSocket back through this origin. The
		// scheme sources are deliberately host-agnostic: the dev server is
		// reached as localhost, 127.0.0.1 or a LAN address depending on how
		// the developer opened the page.
		connectSrc = "connect-src 'self' ws: wss:"
	}

	return []string{
		"default-src 'self'",
		scriptSrc,
		styleSrc,
		imgSrc,
		fontSrc,
		connectSrc,
		// Hard invariants, in both modes: no plugin content, no <base>
		// rewriting of relative URLs, no framing (clickjacking), and forms
		// may only post back to this origin.
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
	}
}

// SPAContentSecurityPolicy returns the Content-Security-Policy header value
// for SPA (HTML and asset) responses. devMode must be true only when the BFF
// is proxying to the Vite dev server.
func SPAContentSecurityPolicy(devMode bool) string {
	return strings.Join(spaCSPDirectives(devMode), "; ")
}

// SPAHeaders returns middleware that replaces the strict API CSP set by
// SecurityHeaders with the SPA policy, for the handler that serves the
// single-page app. It is a Set (not an Add), so a response carries exactly
// one Content-Security-Policy header and the API policy never leaks onto an
// HTML response — nor the SPA policy onto a JSON one.
//
// X-Frame-Options is re-asserted so this middleware is correct even if it is
// ever used without SecurityHeaders in front of it.
func SPAHeaders(devMode bool) func(http.Handler) http.Handler {
	policy := SPAContentSecurityPolicy(devMode)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Content-Security-Policy", policy)
			h.Set("X-Frame-Options", "DENY")
			next.ServeHTTP(w, r)
		})
	}
}
