package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/session"
)

// AppRoute is one app's routing and cookie configuration. M1 registers only
// the portal app; Application A/B are added in M2 by constructing another
// AppRoute with their own prefix, cookie names and OIDCClient.
type AppRoute struct {
	// Key identifies this app in session/log data ("portal",
	// "driving-licence", "revenue-licence").
	Key string
	// RoutePrefix is both the SPA route prefix this app owns and the BFF's
	// own path prefix ("/bff/portal"). Session and login-transaction
	// cookies are scoped to this exact path, so the browser never presents
	// one app's cookie to another's routes.
	RoutePrefix string
	// AppRouteReturnToPrefix is the SPA-side prefix ValidateReturnTo checks
	// returnTo against ("/" for the portal, "/apps/driving-licence" for
	// Application A, etc). Usually equals the SPA mount point, which is not
	// always the same string as RoutePrefix.
	ReturnToPrefix string

	Client OIDCClient

	SessionCookieName  string
	LoginTxnCookieName string
	CSRFCookieName     string

	PostLogoutRedirectURI string

	// ClientID is this app's registered OIDC client ID — surfaced to the SPA
	// so it can show "released to <clientId>" without ever seeing a token.
	ClientID string
	// AppName is this app's human-readable name as registered in WSO2 IS
	// ("Citizen Portal", "Driving Licence Service", "Vehicle Revenue Licence").
	AppName string
}

// Server holds everything the HTTP handlers need: the registered apps
// (keyed by Key), the shared session/transaction manager, and cross-cutting
// settings (cookie Secure flag, logger).
type Server struct {
	Apps               map[string]*AppRoute
	Sessions           *session.Manager
	CookieSecure       bool
	SessionIdleTimeout time.Duration
	Logger             *slog.Logger

	// Upstream calls gov-services-api on the citizen's behalf using the
	// access token captured in their session. See
	// internal/httpapi/portal_data.go, drivinglicence_data.go and
	// revenuelicence_data.go for the named handlers that use it.
	Upstream UpstreamClient

	// SPA serves the single-page app for every path no /bff/... route
	// matched, making the BFF the browser's only origin. It is built by
	// internal/devproxy and is optional: with no SPA configured, an
	// unmatched path is a JSON 404 (which is what the handler tests want,
	// and what an API-only deployment should do).
	SPA http.Handler
	// SPADevMode must be true exactly when SPA proxies to the Vite dev
	// server, since it selects the looser development
	// Content-Security-Policy (security.SPAHeaders). It has no effect
	// without SPA.
	SPADevMode bool

	// replaySeen remembers back-channel logout token jtis already
	// processed, so a replayed token is rejected rather than re-destroying
	// (harmlessly, but noisily) the same sessions.
	replaySeen *session.Store[struct{}]
}

// NewServer constructs a Server ready for RegisterRoutes, initializing the
// internal replay-detection store.
func NewServer(apps map[string]*AppRoute, sessions *session.Manager, cookieSecure bool, sessionIdleTimeout time.Duration, logger *slog.Logger, upstreamClient UpstreamClient) *Server {
	return &Server{
		Apps:               apps,
		Sessions:           sessions,
		CookieSecure:       cookieSecure,
		SessionIdleTimeout: sessionIdleTimeout,
		Logger:             logger,
		Upstream:           upstreamClient,
		replaySeen:         session.NewStore[struct{}](5000),
	}
}
