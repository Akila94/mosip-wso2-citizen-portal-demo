# citizen-portal-integration

The implementation of `PORTAL-INTEGRATION-PLAN.md` (repo root). Everything this integration adds
lives in this directory — it does not touch `esignet-bridge/` (the informal variant, left
alone) and builds on top of `setup-without-bridge/` (the production-correct federation the plan
joins to the demo UI).

**Status: M1 complete and live-verified (see `M1-SESSION-NOTES.md`); M2 code-complete, pending
live verification; M3 code-complete and independently re-verified, pending live verification (see
`M3-SESSION-NOTES.md`).** M1 built a walking skeleton — one WSO2 IS application ("Citizen
Portal"), the BFF's full OIDC round trip (login → IS → MOSIP eSignet or local Username & Password
→ callback → session), and RP-initiated + back-channel logout — deliberately built generically
(`Server.Apps map[string]*AppRoute`, `Manager.DestroyBySid` spans every registered app) even
though only one app was wired up. M2 registers the other two apps ("Driving Licence Service",
"Vehicle Revenue Licence") into that same generic machinery — no changes to the OIDC round trip,
session store, or back-channel logout were needed. M3 adds `gov-services-api`, a real resource
server validating each app's JWT access token (signature, `iss`, `exp`, then a per-router
required audience and scope, so one app's token is genuinely rejected by another's router), plus
the BFF's `internal/upstream` typed client and `/bff/{app}/api/...` data routes that call it,
using the OAuth2 access token now captured at login. No SPA changes yet; `GET /bff/{app}/session`
and the new `/bff/{app}/api/...` routes return real data via `curl`/an HTTP client so SSO, single
logout, and the audience/scope story can all be verified end to end before the React app is
touched (M4/M5). See `PORTAL-INTEGRATION-PLAN.md`'s "Phasing" section for M4–M6, and
`MANUAL-STEPS.md` for the Console registration and live-verification steps each milestone needs.

## Layout

```
citizen-portal-integration/
├── MANUAL-STEPS.md          Console work this code cannot do: register all three applications
│                            (§1–§4), add MOSIP eSignet + Username & Password to each one's
│                            Login Flow Step 1, then the M1/M2/M3 live-verification checklists
├── certs/                   IS's exported self-signed cert, for the BFF's pinned trust store
├── citizen-portal-bff/      the Go BFF — three registered apps, now with data routes calling
│                            gov-services-api (M3)
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
│       └── httpapi/         chi routes: login/callback/session/logout/backchannel-logout
│                            per app, plus each app's own /api/... data routes calling
│                            internal/upstream (M3)
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
                             citizen-portal-demo-app's mock services
```

## Running M1 + M2 + M3

```bash
# 1. Bring up eSignet + WSO2 IS (already done for this session):
cd ../setup-without-bridge && ./demo.sh status   # confirm both are up, preflight green

# 2. Register all three applications — see MANUAL-STEPS.md §1-§4.

# 3. Configure and run gov-services-api
cd gov-services-api
cp .env.example .env && chmod 600 .env
# fill in PORTAL_CLIENT_ID, DL_CLIENT_ID, VRL_CLIENT_ID (same values as the BFF's .env below —
# these are not secrets), then:
./run-govapi.sh

# 4. Configure and run the BFF
cd ../citizen-portal-bff
cp .env.example .env && chmod 600 .env
# fill in PORTAL_CLIENT_ID/_SECRET, DL_CLIENT_ID/_SECRET, VRL_CLIENT_ID/_SECRET, then:
./run-bff.sh

# 5. Try it — MANUAL-STEPS.md §7 (M1+M2) and §8 (M3) have the full verification checklists
open http://localhost:8090/bff/portal/login?returnTo=/
```

## Testing

```bash
cd citizen-portal-bff && go build ./... && go vet ./... && go test ./... -race
cd ../gov-services-api && go build ./... && go vet ./... && go test ./... -race
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
required audience — the literal scenario M3 exists to prove.

## What M1 + M2 + M3 deliberately do not include yet

- Custom per-app OAuth scopes on `gov-services-api`'s `/driving-licence/*` and
  `/vehicle-registry/*` routers — an earlier draft had `driving_licence.write`/
  `vehicle_registry.read` gating those routers, but a live debugging session found this added
  real friction (a WSO2 IS Console "Authorization Policy" that defaults to a Role-Based Access
  Control policy no citizen would ever satisfy) for no benefit any app in this project actually
  needs — audience matching alone already proves a citizen holds a validly-authenticated session
  with that specific app. The scope requirement was removed project-wide; see
  `M3-SESSION-NOTES.md`'s live-verification section for the full history.
- The SPA dev-proxy / static-file serving `DEV_PROXY_TARGET`/`STATIC_DIR` reserve — M4/M5, once
  the React app is being migrated onto this BFF.
- Step-up authentication (`prompt=login`) — added once there is a concrete step-up scenario
  (the driving-licence submission flow) to drive it.
- Access-token refresh — if a citizen's access token expires before the BFF's own session idle
  timeout, an `/api/...` call simply fails with whatever `gov-services-api` returns for an
  expired token; there is no refresh-token handling yet. A documented gap, not an oversight.
- The BFF's `/bff/{app}/api/...` routes have not yet been exercised against a live stack
  end to end — `M3-SESSION-NOTES.md` and `MANUAL-STEPS.md` §8.3 have the exact unverified paths.
