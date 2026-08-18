# citizen-portal-integration

The implementation of `PORTAL-INTEGRATION-PLAN.md` (repo root). Everything this integration adds
lives in this directory — it does not touch `esignet-bridge/` (the informal variant, left
alone) and builds on top of `setup-without-bridge/` (the production-correct federation the plan
joins to the demo UI).

**Status: M1 complete and live-verified (see `M1-SESSION-NOTES.md`); M2 code-complete, pending
live verification; M3 code-complete and independently re-verified, pending live verification (see
`M3-SESSION-NOTES.md`); M4 code-complete — `citizen-portal-demo-app/` now has real routes instead
of a state switch (see `M4-SESSION-NOTES.md`); M5 code-complete — the SPA now runs on the real
stack, with every automated gate green but _nothing yet exercised in a browser_ (see
`M5-SESSION-NOTES.md`, whose "Still not proven" section is the honest risk list).** M1 built a
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
SPA's `AuthContext` and every `*Service.ts` call the real stack. See
`PORTAL-INTEGRATION-PLAN.md`'s "Phasing" section for M6, and `MANUAL-STEPS.md` for the Console
registration and live-verification steps each milestone needs.

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

```bash
# 1. Bring up eSignet + WSO2 IS:
cd ../setup-without-bridge && ./demo.sh status   # confirm both are up, preflight green

# 2. Register all three applications — see MANUAL-STEPS.md §1-§4.

# 3. Configure and run gov-services-api
cd gov-services-api
cp .env.example .env && chmod 600 .env
# fill in PORTAL_CLIENT_ID, DL_CLIENT_ID, VRL_CLIENT_ID (same values as the BFF's .env below —
# these are not secrets), then:
./run-govapi.sh

# 4. Build the SPA (static mode), or run `npm run dev` alongside if using DEV_PROXY_TARGET
cd ../../citizen-portal-demo-app && npm install && npm run build

# 5. Configure and run the BFF
cd ../citizen-portal-integration/citizen-portal-bff
cp .env.example .env && chmod 600 .env
# fill in PORTAL_CLIENT_ID/_SECRET, DL_CLIENT_ID/_SECRET, VRL_CLIENT_ID/_SECRET, then:
./run-bff.sh

# 6. Open the portal — the BFF serves it, so this is the only URL that works
open http://localhost:8090
```

`MANUAL-STEPS.md` §7 (M1+M2) and §8 (M3) have the `curl`-level verification checklists;
`PORTAL-INTEGRATION-PLAN.md`'s "Manual, end to end" section has the full demo walk-through.

## Testing

```bash
cd citizen-portal-bff  && go build ./... && go vet ./... && go test ./... -race && gosec ./... && govulncheck ./...
cd ../gov-services-api && go build ./... && go vet ./... && go test ./... -race && gosec ./... && govulncheck ./...
cd ../../citizen-portal-demo-app && npm run build      # tsc -b must pass
```

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
- **A live browser run.** Neither the BFF's `/bff/{app}/api/...` routes nor anything M5 added has
  been exercised end to end against the running stack in a browser. The two CSPs are the most
  likely thing to be subtly wrong, because a CSP failure is silent in tests and total in a
  browser. `M5-SESSION-NOTES.md`'s "Still not proven" section is the full list;
  `MANUAL-STEPS.md` §8.3 has the M3-era unverified paths.
