package httpapi

import (
	"net/http"
	"time"
)

// setCookie writes an HttpOnly, SameSite=Lax cookie scoped to app's own
// route prefix, so the browser only ever presents it to that app's own
// endpoints — the path-scoping isolation described in
// PORTAL-INTEGRATION-PLAN.md's "three apps, three clients, one origin"
// section. httpOnly is a parameter (not always true) because the CSRF
// cookie must be readable by the SPA's JavaScript to echo back as a header.
func setCookie(w http.ResponseWriter, app *AppRoute, secure bool, name, value string, ttl time.Duration, httpOnly bool) {
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- Secure/SameSite are always set; httpOnly=false is intentional only for the CSRF cookie, which must be JS-readable for the double-submit pattern
		Name:     name,
		Value:    value,
		Path:     app.RoutePrefix,
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: httpOnly,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearCookie expires a cookie previously set with setCookie.
func clearCookie(w http.ResponseWriter, app *AppRoute, secure bool, name string) {
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- HttpOnly is always true here; Secure is intentionally derived from CookieSecure (config), not a literal
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
