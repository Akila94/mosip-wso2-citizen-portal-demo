# citizen-portal-integration

The implementation of `PORTAL-INTEGRATION-PLAN.md` (repo root). Everything this integration adds
lives in this directory — it does not touch `esignet-bridge/` (the informal variant, left
alone) and builds on top of `setup-without-bridge/` (the production-correct federation the plan
joins to the demo UI).

**Status: M1 complete and live-verified (see `M1-SESSION-NOTES.md`); M2 code-complete, pending
live verification.** M1 built a walking skeleton — one WSO2 IS application ("Citizen Portal"),
the BFF's full OIDC round trip (login → IS → MOSIP eSignet or local Username & Password →
callback → session), and RP-initiated + back-channel logout — deliberately built generically
(`Server.Apps map[string]*AppRoute`, `Manager.DestroyBySid` spans every registered app) even
though only one app was wired up. M2 registers the other two apps ("Driving Licence Service",
"Vehicle Revenue Licence") into that same generic machinery — no changes to the OIDC round trip,
session store, or back-channel logout were needed. No SPA changes yet; `GET /bff/{app}/session`
returns the real ID-token claims as JSON so SSO and single logout can be verified end to end
before the React app is touched (M4/M5). See `PORTAL-INTEGRATION-PLAN.md`'s "Phasing" section
for M3–M6, and `MANUAL-STEPS.md` for the Console registration this milestone needs.

## Layout

```
citizen-portal-integration/
├── MANUAL-STEPS.md          Console work this code cannot do: register all three applications
│                            (§1–§4), add MOSIP eSignet + Username & Password to each one's
│                            Login Flow Step 1, then the M1 and M2 live-verification checklists
├── certs/                   IS's exported self-signed cert, for the BFF's pinned trust store
└── citizen-portal-bff/      the Go BFF — three registered apps as of M2
    ├── cmd/bff/             main.go — wiring all three AppRoutes, graceful shutdown
    └── internal/
        ├── config/          env-only configuration for all three apps' OIDC clients,
        │                    validated at boot
        ├── security/        log-forging prevention, returnTo/open-redirect guard, PKCE,
        │                    CSRF, security headers — no dependency on the other packages
        ├── session/         TTL-bounded stores for login transactions and authenticated
        │                    sessions, shared across every app so `DestroyBySid` can end all
        │                    three apps' sessions from one IdP logout; assurance-level
        │                    derivation from `amr`
        ├── oidcrp/          OIDC discovery, authorization-code + PKCE round trip, ID-token
        │                    and back-channel logout-token verification — one RP instance per
        │                    app, same discovered provider
        └── httpapi/         chi routes: login/callback/session/logout/backchannel-logout,
                             registered once per app (`Server.Apps`)
```

## Running M1 + M2

```bash
# 1. Bring up eSignet + WSO2 IS (already done for this session):
cd ../setup-without-bridge && ./demo.sh status   # confirm both are up, preflight green

# 2. Register all three applications — see MANUAL-STEPS.md §1-§4.

# 3. Configure and run the BFF
cd citizen-portal-bff
cp .env.example .env && chmod 600 .env
# fill in PORTAL_CLIENT_ID/_SECRET, DL_CLIENT_ID/_SECRET, VRL_CLIENT_ID/_SECRET, then:
go run ./cmd/bff

# 4. Try it — MANUAL-STEPS.md §7 has the full M1 + M2 verification checklist
open http://localhost:8090/bff/portal/login?returnTo=/
```

## Testing

```bash
cd citizen-portal-bff
go build ./...
go vet ./...
go test ./... -race
```

Every package other than `cmd/bff` has unit tests written before its implementation
(TDD): `security` and `session` are pure-logic tests; `oidcrp` runs its verification logic
against a real signed-JWS round trip through a stub OIDC server (issuer discovery, JWKS,
nonce/audience/expiry/signature checks, back-channel logout-token checks) — no live IS
required; `httpapi` exercises the HTTP layer (cookies, CSRF, returnTo validation, replay
rejection, back-channel session teardown) against a fake `OIDCClient`, so it is independent of
both `oidcrp`'s and IS's specifics.

## What M1 + M2 deliberately do not include yet

- `gov-services-api` (the mock resource server with per-audience JWT checks) and the custom
  `driving_licence.write`/`vehicle_registry.read` OAuth scopes it needs registered — M3. Until
  then, whether IS accepts or silently drops those two scope names on Applications A/B's
  authorization requests is **unverified** — see `MANUAL-STEPS.md` §5.
- The SPA dev-proxy / static-file serving `DEV_PROXY_TARGET`/`STATIC_DIR` reserve — M4/M5, once
  the React app is being migrated onto this BFF.
- Step-up authentication (`prompt=login`) — added once there is a concrete step-up scenario
  (the driving-licence submission flow) to drive it.
