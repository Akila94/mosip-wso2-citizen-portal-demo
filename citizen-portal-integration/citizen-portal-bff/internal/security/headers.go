package security

import "net/http"

// SecurityHeaders sets a conservative baseline of response headers on every
// request: no framing (clickjacking), no MIME sniffing, no referrer leakage,
// and a same-origin-only CSP — appropriate for a BFF that renders no
// third-party content and expects no inline scripts.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}
