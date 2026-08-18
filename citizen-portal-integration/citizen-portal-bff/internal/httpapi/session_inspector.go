package httpapi

import (
	"net/http"
	"sort"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/session"
	"github.com/go-chi/chi/v5"
)

// maxReleasedClaims bounds the ID-token claim map kept per session and
// returned by the inspector. A claim set from WSO2 IS is small (a dozen
// claims at most), so this only ever bites if something upstream changes
// dramatically — at which point a bounded map is exactly what is wanted
// rather than unbounded per-session memory.
const maxReleasedClaims = 64

// nonceClaim is dropped from every stored claim set: it is a per-transaction
// handshake value, already verified during login, with no meaning to show a
// citizen — and no reason to keep beyond the callback.
const nonceClaim = "nonce"

// sessionInspectorView is the SSO evidence behind one app's session, all of
// it computed from this BFF's own session state — nothing here comes from
// gov-services-api, and nothing here is a token.
//
// It carries facts only. There is deliberately no human-readable
// "comparison note" and no preformatted display rows: how two apps' claim
// sets are presented side by side is the SPA's decision, not the BFF's.
type sessionInspectorView struct {
	AppKey      string `json:"appKey"`
	ClientID    string `json:"clientId"`
	ClientLabel string `json:"clientLabel"`
	Subject     string `json:"subject"`
	// IDP is derived from `amr` (session.DeriveIdentityProvider); IS emits
	// no `idp` claim.
	IDP string `json:"idp"`
	// Acr is omitted when empty: IS emits `acr` only when a value was
	// actually resolved for the flow.
	Acr string `json:"acr,omitempty"`
	// Amr and ClientsInSession always serialize as arrays, and
	// ReleasedClaims always as an object, so the SPA never has to null-check
	// them on a render.
	Amr              []string              `json:"amr"`
	Sid              string                `json:"sid"`
	AssuranceLevel   string                `json:"assuranceLevel"`
	AuthTime         int64                 `json:"authTime,omitempty"`
	ExpiresAt        int64                 `json:"expiresAt,omitempty"`
	ClientsInSession []clientInSessionView `json:"clientsInSession"`
	ReleasedClaims   map[string]any        `json:"releasedClaims"`
}

// clientInSessionView names one app that currently holds a live session on
// the same IdP session as the caller.
type clientInSessionView struct {
	AppKey  string `json:"appKey"`
	AppName string `json:"appName"`
}

// mountSessionInspectorRoute registers GET /session-inspector on an app's
// already-session-gated /api router. It is called for all three apps: the
// screen's whole point is holding two of them side by side.
func (s *Server) mountSessionInspectorRoute(r chi.Router, app *AppRoute) {
	r.Get("/session-inspector", s.handleSessionInspector(app))
}

// handleSessionInspector answers with the facts behind the caller's session.
//
// It never calls gov-services-api: everything it reports is already known
// to this process, so the inspector keeps working (and keeps telling the
// truth about the session) even when the resource server is down.
func (s *Server) handleSessionInspector(app *AppRoute) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, ok := sessionFromContext(r.Context())
		if !ok {
			s.internalError(w, errMissingSessionInContext)
			return
		}

		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		s.writeJSON(w, sessionInspectorView{
			AppKey:      app.Key,
			ClientID:    app.ClientID,
			ClientLabel: app.AppName,
			Subject:     sess.User.Sub,
			IDP:         session.DeriveIdentityProvider(sess.Amr),
			Acr:         sess.Acr,
			Amr:         nonNilStrings(sess.Amr),
			Sid:         sess.Sid,
			// Assurance is always derived server-side from the verified
			// session, never taken from the request — the same invariant
			// handlePortalCatalogue documents.
			AssuranceLevel:   string(session.DeriveAssuranceLevel(sess.Amr)),
			AuthTime:         unixOrZero(sess.AuthTime),
			ExpiresAt:        unixOrZero(sess.ExpiresAt),
			ClientsInSession: s.clientsInSession(app, sess),
			ReleasedClaims:   releasedClaims(sess.IDTokenClaims),
		})
	}
}

// clientsInSession lists every registered app currently holding a live
// session with the same IdP `sid` as sess — the BFF's own, always-available
// answer to "which clients is this one SSO session covering", and the thing
// that makes SSO visible on screen.
//
// The calling app is always included, even when its session carries no
// `sid`: a token with no sid cannot be correlated with any other, so the
// honest answer is "this app only" rather than every sid-less session in the
// process (session.Manager.FindBySid enforces that).
//
// The result is sorted by app key and deduplicated, so repeated polls return
// a stable list — a second login for one app leaves two store entries for it,
// which must not show up as two clients.
func (s *Server) clientsInSession(app *AppRoute, sess session.AuthSession) []clientInSessionView {
	appKeys := map[string]bool{app.Key: true}
	for _, peer := range s.Sessions.FindBySid(sess.Sid) {
		appKeys[peer.AppKey] = true
	}

	clients := make([]clientInSessionView, 0, len(appKeys))
	for key := range appKeys {
		peerApp, registered := s.Apps[key]
		if !registered {
			// A session for an app this Server does not serve can only mean
			// the app registry changed under a live session; naming it would
			// tell the SPA about something it cannot link to.
			s.Logger.Warn("skipping a session for an unregistered app", "appKey", security.SanitizeForLog(key))
			continue
		}
		clients = append(clients, clientInSessionView{AppKey: peerApp.Key, AppName: peerApp.AppName})
	}
	sort.Slice(clients, func(i, j int) bool { return clients[i].AppKey < clients[j].AppKey })
	return clients
}

// releasedClaims returns a bounded, defensive copy of an ID token's claim
// set, with the per-transaction `nonce` removed.
//
// It is applied when the claim set is stored (callback.go) and again when it
// is returned, so neither the session store nor a response can carry more
// than maxReleasedClaims entries. Keys are selected in sorted order when the
// bound bites, so truncation is deterministic rather than dependent on Go's
// randomized map iteration.
//
// The raw ID token itself is never a member of this map: it is a JWS string,
// not a claim, and lives only in AuthSession.RawIDToken for the logout
// id_token_hint.
func releasedClaims(claims map[string]any) map[string]any {
	released := make(map[string]any, len(claims))
	if len(claims) <= maxReleasedClaims {
		for name, value := range claims {
			if name == nonceClaim {
				continue
			}
			released[name] = value
		}
		return released
	}

	names := make([]string, 0, len(claims))
	for name := range claims {
		if name == nonceClaim {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) > maxReleasedClaims {
		names = names[:maxReleasedClaims]
	}
	for _, name := range names {
		released[name] = claims[name]
	}
	return released
}

// nonNilStrings returns values, or an empty slice when it is nil, so the
// field serializes as [] rather than null.
func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
