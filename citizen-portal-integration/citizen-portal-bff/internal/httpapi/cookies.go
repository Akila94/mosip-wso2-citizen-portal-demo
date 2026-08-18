package httpapi

import (
	"net/http"
	"time"
)

// setCookie writes an HttpOnly, SameSite=Lax cookie scoped to app's own
// route prefix, so the browser only ever presents it to that app's own
// endpoints — the path-scoping isolation described in
// PORTAL-INTEGRATION-PLAN.md's "three apps, three clients, one origin"
// section.
//
// Every cookie this BFF sets is HttpOnly, with no opt-out: the CSRF token
// used to be JS-readable so the SPA could echo it back, but a cookie scoped
// to Path=/bff/{app} is never exposed to document.cookie on an SPA page
// under RFC 6265 §5.4 path-matching, so that never worked. The token now
// travels to the SPA in GET /bff/{app}/session's response body instead, and
// the cookie is closed to script entirely.
func setCookie(w http.ResponseWriter, app *AppRoute, secure bool, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- HttpOnly and SameSite are unconditionally set below; Secure is intentionally the CookieSecure config value (config.Load derives it from BFF_PUBLIC_URL's scheme, with an explicit override for a TLS-terminating front) rather than a literal, which is the only thing gosec cannot prove here
		Name:     name,
		Value:    value,
		Path:     app.RoutePrefix,
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearCookie expires a cookie previously set with setCookie.
func clearCookie(w http.ResponseWriter, app *AppRoute, secure bool, name string) {
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- same justification as setCookie: HttpOnly/SameSite are literals here, only Secure comes from configuration
		Name:     name,
		Value:    "",
		Path:     app.RoutePrefix,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func readCookie(r *http.Request, name string) (string, bool) {
	c, err := r.Cookie(name)
	if err != nil {
		return "", false
	}
	return c.Value, true
}
