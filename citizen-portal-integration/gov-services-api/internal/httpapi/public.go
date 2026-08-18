package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// mountPublicRoutes registers the /public/* handlers on pr.
//
// This is the first — and, for now, only — unauthenticated surface in either
// service, so it is worth being explicit about what that means:
//
//   - Everything mounted here is readable by anyone who can reach the port,
//     with no token, no audience check and no `sub`. Only genuinely public
//     information belongs here. A government service catalogue qualifies: it
//     is the list of services the state offers, identical for every visitor,
//     and the portal's landing page shows it to signed-out visitors by
//     design.
//   - Nothing here may read the citizen registry, take a subject, or project
//     anything per citizen. Those all live behind
//     authmw.RequireAudienceAndScope and must stay there.
//   - The protections that are not authentication still apply: NewRouter's
//     middleware chain (RealIP, Recoverer, requestBodyLimit's 64 kB cap)
//     wraps this group like every other, and cmd/govapi/main.go's
//     Read/ReadHeader/Write/Idle timeouts are set on the http.Server, so they
//     bound every connection regardless of which route it reaches.
//   - There is no rate limiting anywhere in this project — no helper exists
//     in either module — so this endpoint is not rate limited either. That is
//     a known gap for a public endpoint, called out rather than papered over
//     with something invented here.
func (s *Server) mountPublicRoutes(pr chi.Router) {
	pr.Get("/catalogue", s.handlePublicCatalogue)
}

// handlePublicCatalogue returns the service catalogue in its signed-out
// state: the same serviceCategories fixture the authenticated
// /portal/catalogue handler serves (one source of truth — see portal.go),
// with each service's own `state` exactly as the fixture defines it and no
// READY/STEP_UP promotion.
//
// An `assuranceLevel` query parameter is **ignored**, not rejected: this
// handler never reads the query string at all, so there is no code path by
// which a caller-supplied assurance level could influence the response.
// Ignoring is preferred over answering 400 because a rejection would turn
// the parameter into a probe for how the server treats it, whereas silence
// makes the honest point — there is no session here, so there is no
// assurance level to speak of, and trusting a caller's claim about their own
// assurance is exactly the privilege escalation the authenticated route's
// design note warns against.
//
// Promotion is deliberately left to the authenticated route: a signed-out
// visitor should see what services exist, and be asked to sign in when they
// choose one.
func (s *Server) handlePublicCatalogue(w http.ResponseWriter, r *http.Request) {
	// serviceCategories is written, never mutated — the authenticated
	// handler copies before it promotes states, precisely so this one can
	// serve the fixture as-is.
	s.writeJSON(w, serviceCategories)
}
