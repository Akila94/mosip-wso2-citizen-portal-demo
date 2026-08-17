package httpapi

import (
	"log/slog"
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

	// replaySeen remembers back-channel logout token jtis already
	// processed, so a replayed token is rejected rather than re-destroying
	// (harmlessly, but noisily) the same sessions.
	replaySeen *session.Store[struct{}]
}

// NewServer constructs a Server ready for RegisterRoutes, initializing the
// internal replay-detection store.
func NewServer(apps map[string]*AppRoute, sessions *session.Manager, cookieSecure bool, sessionIdleTimeout time.Duration, logger *slog.Logger) *Server {
	return &Server{
		Apps:               apps,
		Sessions:           sessions,
		CookieSecure:       cookieSecure,
		SessionIdleTimeout: sessionIdleTimeout,
		Logger:             logger,
		replaySeen:         session.NewStore[struct{}](5000),
	}
}
