// This file holds the shared plumbing the three apps' /bff/{app}/api/...
// data-route handlers (portal_data.go, drivinglicence_data.go,
// revenuelicence_data.go) all use: reading the session requireSession
// stored, and forwarding an upstream.Response through to the caller —
// verbatim, except for the one status a session-backed route must not
// forward blindly (see forwardSessionUpstreamResponse).
package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/session"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/upstream"
)

// errMissingSessionInContext indicates a data-route handler ran without
// requireSession middleware having populated the request context first —
// a wiring bug, never a caller-triggerable condition, hence the 500 rather
// than a 401.
var errMissingSessionInContext = errors.New("httpapi: session missing from request context (requireSession middleware not applied)")

// defaultUpstreamContentType is used when gov-services-api's response
// carries no Content-Type header, matching this project's convention of
// never leaving a response's Content-Type unset.
const defaultUpstreamContentType = "application/json"

// Bodies returned in place of an upstream 401. Neither repeats
// gov-services-api's own error body: that body describes a token the browser
// must never learn anything about, and its wording is not ours to promise.
const (
	// expiredAccessTokenMessage accompanies the one upstream 401 that really
	// is the citizen's session: signing in again is the fix.
	expiredAccessTokenMessage = "your session has expired; sign in again to continue"
	// rejectedAccessTokenMessage accompanies an upstream 401 the citizen
	// cannot fix, so it deliberately blames this deployment rather than them.
	rejectedAccessTokenMessage = "the government service rejected this application's credentials"
)

// forwardSessionUpstreamResponse answers a session-backed data route with
// gov-services-api's response, translating an upstream 401 into the failure it
// actually is before it reaches the SPA.
//
// A 401 from gov-services-api and a 401 from requireSession mean opposite
// things, and forwarding the first verbatim made them indistinguishable: the
// SPA renders any 401 as "your session has expired", which for a rejected —
// but live — access token is both false and unactionable, and has already
// cost a real diagnosis (an application registered in the WSO2 IS Console with
// the Default, opaque access-token type, which the resource server cannot
// validate; the citizen's session was never involved). So:
//
//   - the stored access token has demonstrably expired → 401, because signing
//     in again genuinely fixes it. There is no refresh-token support, by
//     documented design — see session.AuthSession.AccessTokenExpiresAt.
//   - anything else → 502, because the fault is on this side of the browser:
//     wrong audience, wrong issuer, a token the resource server cannot parse,
//     or a trust/config problem. Re-authenticating cannot fix any of them.
//
// Every other status, 403 included, is forwarded untouched: gov-services-api
// answering 403 is already an unambiguous statement about audience or scope.
func (s *Server) forwardSessionUpstreamResponse(w http.ResponseWriter, r *http.Request, sess session.AuthSession, resp upstream.Response, err error) {
	if err == nil && resp.StatusCode == http.StatusUnauthorized {
		s.respondToRejectedAccessToken(w, r, sess, resp.StatusCode)
		return
	}
	s.writeUpstreamResponse(w, resp, err)
}

// respondToRejectedAccessToken answers the upstream 401 forwardSessionUpstreamResponse
// intercepted, and logs the line whoever debugs this next will read.
//
// The two cases are logged at different levels on purpose: an expired token is
// an expected consequence of having no refresh-token support, while a rejected
// live token is a misconfiguration nobody will notice unless it is loud.
func (s *Server) respondToRejectedAccessToken(w http.ResponseWriter, r *http.Request, sess session.AuthSession, upstreamStatus int) {
	// The path can carry caller-supplied segments (a vehicle id, say), so it
	// is sanitized before it reaches a log line — guideline §1.7. The token
	// itself is never logged, expired or not.
	path := security.SanitizeForLog(r.URL.Path)

	if accessTokenExpired(sess, time.Now()) {
		s.Logger.Warn("gov-services-api rejected an expired access token; the citizen must sign in again (this BFF has no refresh-token support, by design)",
			"path", path,
			"upstreamStatus", upstreamStatus,
			"appKey", security.SanitizeForLog(sess.AppKey))
		http.Error(w, expiredAccessTokenMessage, http.StatusUnauthorized)
		return
	}

	s.Logger.Error("gov-services-api rejected an access token that has not expired: a configuration fault on our side, not a citizen session problem — re-authenticating cannot fix it",
		"path", path,
		"upstreamStatus", upstreamStatus,
		"appKey", security.SanitizeForLog(sess.AppKey),
		"accessTokenExpiresAt", accessTokenExpiryForLog(sess),
		"checkFirst", "the access-token type on this application's WSO2 IS Console registration: left as Default it issues an opaque token the resource server cannot validate, and it must be JWT",
		"thenCheck", "that the token's issuer and audience are the ones gov-services-api validates against")
	http.Error(w, rejectedAccessTokenMessage, http.StatusBadGateway)
}

// accessTokenExpired reports whether sess's access token is known to have
// expired by now. A zero AccessTokenExpiresAt means IS never told us when the
// token expires, which is not evidence of expiry — treating that as "expired"
// would blame the citizen's session on a guess, which is the exact mistake
// this whole split exists to undo.
func accessTokenExpired(sess session.AuthSession, now time.Time) bool {
	return !sess.AccessTokenExpiresAt.IsZero() && now.After(sess.AccessTokenExpiresAt)
}

// accessTokenExpiryForLog renders the access token's expiry for a log line,
// naming the unknown case explicitly rather than logging a zero timestamp that
// reads like a real (and wildly stale) one.
func accessTokenExpiryForLog(sess session.AuthSession) string {
	if sess.AccessTokenExpiresAt.IsZero() {
		return "unknown"
	}
	return sess.AccessTokenExpiresAt.UTC().Format(time.RFC3339)
}

// forwardPublicUpstreamResponse answers the one session-less route this BFF
// serves (handlePublicPortalCatalogue) with gov-services-api's response
// verbatim.
//
// It exists as its own function, rather than a flag on
// forwardSessionUpstreamResponse, because a public call has no session and no
// access token to reason about — mirroring upstream.Client's own do/doPublic
// split. Passing a zero-valued session through the session path would invent a
// token expiry that does not exist and make an upstream 401 on this route look
// like a credentials failure, when it is simply gov-services-api's answer
// about its own public route.
func (s *Server) forwardPublicUpstreamResponse(w http.ResponseWriter, resp upstream.Response, err error) {
	s.writeUpstreamResponse(w, resp, err)
}

// writeUpstreamResponse writes resp's status code and body through to w
// verbatim when err is nil. A non-nil err means the call to
// gov-services-api itself failed (a genuine transport failure — see
// upstream.Client.do's doc comment) rather than gov-services-api returning
// a non-2xx status, so it is logged (sanitized) and answered with 502; it
// is never gov-services-api's own error response, which is instead
// forwarded as-is by the resp branch below.
func (s *Server) writeUpstreamResponse(w http.ResponseWriter, resp upstream.Response, err error) {
	if err != nil {
		s.Logger.Error("upstream call to gov-services-api failed", "error", security.SanitizeForLog(err.Error()))
		http.Error(w, "upstream service unavailable", http.StatusBadGateway)
		return
	}

	contentType := resp.ContentType
	if contentType == "" {
		contentType = defaultUpstreamContentType
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(resp.StatusCode)
	if _, writeErr := w.Write(resp.Body); writeErr != nil {
		s.Logger.Warn("failed writing upstream response body to client", "error", writeErr.Error())
	}
}
