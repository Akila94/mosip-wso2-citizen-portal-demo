# citizen-portal-integration

The implementation of `PORTAL-INTEGRATION-PLAN.md` (repo root). Everything this integration adds
lives in this directory — it does not touch `esignet-bridge/` (the informal variant, left
alone) and builds on top of `setup-without-bridge/` (the production-correct federation the plan
joins to the demo UI).

**New to this repo? Start with [`GETTING-STARTED.md`](GETTING-STARTED.md)** — a
fresh-clone-to-running-demo walkthrough that sequences everything below (and
`setup-without-bridge/`'s own setup) into one path. Everything past this point is reference
material for once that's running.

**Status: M1 complete and live-verified (see `M1-SESSION-NOTES.md`); M2 code-complete, pending
live verification; M3 code-complete and independently re-verified, pending live verification (see
`M3-SESSION-NOTES.md`); M4 code-complete — `citizen-portal-demo-app/` now has real routes instead
of a state switch (see `M4-SESSION-NOTES.md`); M5 code-complete, and live-verified against the
running stack — a real 401-vs-401 bug (an expired session confused with a token
`gov-services-api` rejected for an unrelated reason) was found and fixed in that pass; M6
code-complete — `portal-demo.sh` (below) replaces the four hand-run terminals with one
orchestration script, and its `preflight` subcommand has been run against the live stack with
`failed=0` (see `M6-SESSION-NOTES.md`).** M1 built a
walking skeleton — one WSO2 IS application ("Citizen
Portal"), the BFF's full OIDC round trip (login → IS → MOSIP eSignet or local Username & Password
→ callback → session), and RP-initiated + back-channel logout — deliberately built generically
(`Server.Apps map[string]*AppRoute`, `Manager.DestroyBySid` spans every registered app) even
though only one app was wired up. M2 registers the other two apps ("Driving Licence Service",
"Vehicle Revenue Licence") into that same generic machinery — no changes to the OIDC round trip,
session store, or back-channel logout were needed. M3 adds `gov-services-api`, a real resource
server validating each app's JWT access token (signature, `iss`, `exp`, then a per-router
required audience and scope, so one app's token is genuinely rejected by another's router), plus
the BFF's `internal/upstream` typed client and `/bff/{app}/api/...` data routes that call it,
using the OAuth2 access token now captured at login. M4 migrated the React app onto `react-router`
with its mock services untouched — a large mechanical diff kept separate from a behavioural one.
M5 joined the two halves: the BFF now serves the SPA (so it is the browser's only origin), and the
SPA's `AuthContext` and every `*Service.ts` call the real stack. M6 adds `portal-demo.sh` (setup,
build, start, stop, restart, status, preflight, logs, clean), and closes out repo hygiene: the
`package-lock.json` `.gitignore` rule is anchored to `esignet-bridge/` only, so
`citizen-portal-demo-app/package-lock.json` is now committable, and `bin/` (this script's Go
build output) is gitignored. See `M6-SESSION-NOTES.md` for what was verified live and what
remains a documented gap, and `MANUAL-STEPS.md` for the Console registration steps every
milestone needs.

## How the SPA is served (M5 onwards)

**Open `http://localhost:8090` — never `http://localhost:5173`.** The BFF is the only
browser-facing origin; a page loaded straight from Vite has no BFF session and every data call
fails. Two modes, chosen by one variable in `citizen-portal-bff/.env`:

| `DEV_PROXY_TARGET` | Behaviour |
|---|---|
| `http://localhost:5173` | The BFF reverse-proxies everything that is not `/bff/…` to the Vite dev server, passing WebSocket upgrades through so HMR still works. Run `npm run dev` in `citizen-portal-demo-app/` alongside it. |
| empty (the default) | The BFF serves `STATIC_DIR` (`../../citizen-portal-demo-app/dist`) with SPA fallback, so deep links survive a refresh. Run `npm run build` first. |

Registered redirect URIs are identical in both modes, which is the point — changing them would
mean redoing Console work.

## Layout

```
citizen-portal-integration/
├── MANUAL-STEPS.md          Console work this code cannot do: register all three applications
│                            (§1–§4), add MOSIP eSignet + Username & Password to each one's
│                            Login Flow Step 1, then the M1/M2/M3 live-verification checklists
├── portal-demo.sh           M6 orchestration: setup/build/start/stop/restart/status/preflight/
│                            logs/clean for citizen-portal-bff + gov-services-api. Never starts
│                            WSO2 IS or eSignet — checks they're up and points at
│                            setup-without-bridge/demo.sh if not
├── certs/                   IS's exported self-signed cert, for the BFF's pinned trust store
├── citizen-portal-bff/      the Go BFF — three registered apps, data routes calling
│                            gov-services-api (M3), and the browser's only origin (M5)
│   ├── cmd/bff/             main.go — wiring all three AppRoutes + the upstream client,
│   │                        graceful shutdown
│   └── internal/
│       ├── config/          env-only configuration for all three apps' OIDC clients,
│       │                    validated at boot
│       ├── security/        log-forging prevention, returnTo/open-redirect guard, PKCE,
│       │                    CSRF, security headers — no dependency on the other packages
│       ├── session/         TTL-bounded stores for login transactions and authenticated
│       │                    sessions, shared across every app so `DestroyBySid` can end all
│       │                    three apps' sessions from one IdP logout; assurance-level
│       │                    derivation from `amr`; now also holds each session's OAuth2
│       │                    access token, server-side only, for calling gov-services-api
│       ├── oidcrp/          OIDC discovery, authorization-code + PKCE round trip, ID-token
│       │                    and back-channel logout-token verification — one RP instance per
│       │                    app, same discovered provider
│       ├── upstream/         typed HTTP client for gov-services-api — one named method per
│       │                    route, response passthrough, no generic proxy (M3)
│       ├── devproxy/        serves the SPA (M5): reverse-proxies Vite in dev with WebSocket
│       │                    upgrades passed through, or serves dist/ with SPA fallback —
│       │                    path-traversal-guarded, and never answers an unmatched /bff/
│       │                    path with index.html
│       └── httpapi/         chi routes: login/callback/session/logout/backchannel-logout
│                            per app, plus each app's own /api/... data routes calling
│                            internal/upstream (M3); step-up, the session inspector and the
│                            public catalogue (M5)
└── gov-services-api/        the Go resource server (M3) — validates every request's JWT
    ├── cmd/govapi/          main.go — wiring, graceful shutdown
    └── internal/
        ├── config/           env-only config: the service port, IS issuer/CA, and the three
        │                    apps' client IDs (used as each router's expected audience)
        ├── security/         log-forging prevention (a standalone copy — separate Go module)
        ├── httpclient/       CA-pinned HTTP client for talking to IS (a copy of oidcrp's)
        ├── authmw/           JWT signature verification (IS's JWKS, cached with rotation)
        │                    plus per-router required-audience-and-scope middleware — this is
        │                    what makes Application A's token genuinely rejected by
        │                    Application B's router
        ├── registry/         the sub-keyed citizen registry — NIC/address/vehicles come from
        │                    here, never from the token, since eSignet's `sub` is a pairwise
        │                    pseudonymous identifier
        └── httpapi/         chi routes: /portal/*, /driving-licence/*, /vehicle-registry/*,
                             /citizen/profile — fixture data ported verbatim from
                             citizen-portal-demo-app's mock services; plus /public/* (M5),
                             the one unauthenticated surface, serving the catalogue a
                             signed-out visitor sees
```

## Running it

**`portal-demo.sh` (M6) is the orchestration entry point.** It never starts WSO2 IS or eSignet
itself — that stays `setup-without-bridge/demo.sh`'s job — it only checks they are up and points
there if not.

```bash
# 1. Bring up eSignet + WSO2 IS (unchanged):
cd ../setup-without-bridge && ./demo.sh status   # confirm both are up, preflight green

# 2. Register all three applications — see MANUAL-STEPS.md §1-§4, fill in both .env files.
cd ../citizen-portal-integration
./portal-demo.sh setup      # creates both .env files from .env.example; checks the six
                             # client credentials and the IS cert are in place

# 3. Build both Go services and the SPA (needs Node >= 18 — `nvm use 22` if the check fails)
./portal-demo.sh build

# 4. Start gov-services-api then citizen-portal-bff
./portal-demo.sh start

# 5. Open the portal — the BFF serves it, so this is the only URL that works
open http://localhost:8090

# Later: ./portal-demo.sh status | preflight | logs [bff|govapi] | stop | restart | clean [--all]
```

`run-bff.sh` and `run-govapi.sh` still work directly (via `go run`, no build step) for iterating on
one service in isolation; `portal-demo.sh start` runs built binaries from `bin/` instead, so both
services can be stopped and health-checked by PID rather than by a foreground terminal.

`MANUAL-STEPS.md` §7 (M1+M2) and §8 (M3) have the `curl`-level verification checklists;
`./portal-demo.sh preflight` automates the no-browser subset of them (run clean — `failed=0` —
against the live stack; see `M6-SESSION-NOTES.md`); `PORTAL-INTEGRATION-PLAN.md`'s "Manual, end
to end" section has the full demo walk-through, which still needs a browser.

## Testing

```bash
cd citizen-portal-bff  && gofmt -l . && go build ./... && go vet ./... && go test ./... -race && gosec ./... && govulncheck ./...
cd ../gov-services-api && gofmt -l . && go build ./... && go vet ./... && go test ./... -race && gosec ./... && govulncheck ./...
cd ../../citizen-portal-demo-app && npm run build      # tsc -b must pass
```

Re-run in full for M6 (after the post-M5 401 fix, commit `0d99599`, touched
`internal/httpapi/data_routes.go` and three of its per-app callers): both modules pass every gate
above with **zero** gosec issues and **zero** govulncheck vulnerabilities — the pre-existing
`#nosec` suppressions (3 in `citizen-portal-bff`, 1 in `gov-services-api`) are unchanged and still
individually justified. This is M6's exit bar per `PORTAL-INTEGRATION-PLAN.md`; see
`M6-SESSION-NOTES.md` for the per-command results.

Every package other than each module's `cmd/` entrypoint has unit tests written before its
implementation (TDD): `security` and `session` are pure-logic tests; `oidcrp` runs its
verification logic against a real signed-JWS round trip through a stub OIDC server (issuer
discovery, JWKS, nonce/audience/expiry/signature checks, back-channel logout-token checks) — no
live IS required; `httpapi` exercises the HTTP layer (cookies, CSRF, returnTo validation, replay
rejection, back-channel session teardown, and — since M3 — the data routes through the real
router with a fake upstream client, including the security-critical proof that assurance level is
derived from the session and never taken from the request) against fake `OIDCClient`/
`UpstreamClient` doubles; `gov-services-api`'s `authmw` tests run the same kind of real-signed-JWT
round trip against a stub JWKS server, explicitly proving one app's token is rejected by another's
required audience — the literal scenario M3 exists to prove. M5 adds `devproxy` tests (SPA
fallback, path traversal, the `/bff/` guard, WebSocket upgrade), session-inspector tests, and
tests proving the public catalogue answers with no `Authorization` header and cannot be influenced
by an `assuranceLevel` parameter.

The SPA has **no test suite** — it is verified by `tsc -b` under `strict` plus a production build,
and otherwise by hand. That is a real gap, not a decision anyone should read as settled.

## What is deliberately not included yet

- Custom per-app OAuth scopes on `gov-services-api`'s `/driving-licence/*` and
  `/vehicle-registry/*` routers — an earlier draft had `driving_licence.write`/
  `vehicle_registry.read` gating those routers, but a live debugging session found this added
  real friction (a WSO2 IS Console "Authorization Policy" that defaults to a Role-Based Access
  Control policy no citizen would ever satisfy) for no benefit any app in this project actually
  needs — audience matching alone already proves a citizen holds a validly-authenticated session
  with that specific app. The scope requirement was removed project-wide; see
  `M3-SESSION-NOTES.md`'s live-verification section for the full history.
- **Step-up in the SPA.** `GET /bff/{app}/step-up` (`prompt=login`) is implemented and tested, but
  the driving-licence submit flow still routes through the wireframe step-up screen — a full-page
  round trip through IS mid-application would discard the in-memory wizard state. Consequently
  `AuthContext.raiseAssurance` is a presentation-only override that cannot grant anything; every
  server-side decision derives assurance from the verified session's `amr`. Owner decision, M5.
- **Consent revoke.** Revoking means calling WSO2 IS's consent-management API, a separate
  integration; `gov-services-api` serves consents read-only and the UI control is disabled with
  the reason in its tooltip, rather than wired to a write path that would silently revert on
  reload. Owner decision, M5.
- **Three SPA services stay mocked** — life events, service detail and the medical-review error
  page have no router in `gov-services-api`, because the plan's Component 2 enumerates exactly
  which routers exist. Each is labelled `MOCKED` at its definition; `citizen-portal-demo-app/README.md`
  carries the full real-vs-mocked table. Owner decision, M5.
- **Rate limiting** — nothing in either module implements it, which now matters more than it did:
  M5 introduced the first unauthenticated endpoint. Called out in `gov-services-api`'s
  `internal/httpapi/public.go` rather than papered over.
- **Access-token refresh** — if a citizen's access token expires before the BFF's own session idle
  timeout, an `/api/...` call simply fails with whatever `gov-services-api` returns for an
  expired token; there is no refresh-token handling yet. A documented gap, not an oversight.
- **A full live browser run, all ten steps of the plan's "Manual, end to end" checklist in one
  sitting.** A live pass after M5 did exercise real `/bff/{app}/api/...` calls and found a real
  defect — an upstream 401 from `gov-services-api` (a misconfigured access-token type, not an
  expired session) was indistinguishable from a genuinely expired BFF session, sending the citizen
  into a re-login loop that could never succeed; fixed in the commit right before M6
  ("Fix the incorrect 401 error in session scenario even when the session is valid"). That proves
  the CSP and the SSO round trip work well enough to reach a data call — but the full ten-step
  walk-through (both micro apps, the session inspector side by side, cold entry, single logout,
  the devtools cookie/header checks) has not been run and re-confirmed as one pass since. The two
  CSPs remain the most likely thing to be subtly wrong in a way tests cannot catch.
  `M5-SESSION-NOTES.md`'s "Still not proven" section and `M6-SESSION-NOTES.md` have the detail;
  `MANUAL-STEPS.md` §8.3 has the M3-era unverified paths.
