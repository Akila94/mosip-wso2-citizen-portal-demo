# citizen-portal-integration

The implementation of `PORTAL-INTEGRATION-PLAN.md` (repo root). Everything this integration adds
lives in this directory — it does not touch `esignet-bridge/` (the informal variant, left
alone) and builds on top of `setup-without-bridge/` (the production-correct federation the plan
joins to the demo UI).

**Status: M1 in progress.** M1 is a walking skeleton — one WSO2 IS application ("Citizen
Portal"), the BFF's full OIDC round trip (login → IS → MOSIP eSignet or local
Username & Password → callback → session), and RP-initiated + back-channel logout. No SPA
changes yet; `GET /bff/portal/session` returns the real ID-token claims as JSON so the flow can
be verified end to end before the React app is touched. See `PORTAL-INTEGRATION-PLAN.md`'s
"Phasing" section for M2–M6.

## Layout

```
citizen-portal-integration/
├── MANUAL-STEPS.md          Console work this code cannot do (register the app, add MOSIP
│                            eSignet + Username & Password to Login Flow Step 1)
├── certs/                   IS's exported self-signed cert, for the BFF's pinned trust store
└── citizen-portal-bff/      the Go BFF (M1: portal app only)
    ├── cmd/bff/             main.go — wiring, graceful shutdown
    └── internal/
        ├── config/          env-only configuration, validated at boot
        ├── security/        log-forging prevention, returnTo/open-redirect guard, PKCE,
        │                    CSRF, security headers — no dependency on the other packages
        ├── session/         TTL-bounded stores for login transactions and authenticated
        │                    sessions; assurance-level derivation from `amr`
        ├── oidcrp/          OIDC discovery, authorization-code + PKCE round trip, ID-token
        │                    and back-channel logout-token verification
        └── httpapi/         chi routes: login/callback/session/logout/backchannel-logout
```

## Running M1

```bash
# 1. Bring up eSignet + WSO2 IS (already done for this session):
cd ../setup-without-bridge && ./demo.sh status   # confirm both are up, preflight green

# 2. Register the Citizen Portal application — see MANUAL-STEPS.md.

# 3. Configure and run the BFF
cd citizen-portal-bff
cp .env.example .env && chmod 600 .env
# fill in PORTAL_CLIENT_ID / PORTAL_CLIENT_SECRET from step 2, then:
go run ./cmd/bff

# 4. Try it
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

## What M1 deliberately does not include yet

- Applications A (Driving Licence) and B (Vehicle Revenue Licence) — M2.
- The SPA dev-proxy / static-file serving `DEV_PROXY_TARGET`/`STATIC_DIR` reserve — M4/M5, once
  the React app is being migrated onto this BFF.
- `gov-services-api` (the mock resource server with per-audience JWT checks) — M3.
- Step-up authentication (`prompt=login`) — added alongside Applications A/B once there is a
  concrete step-up scenario (the driving-licence submission flow) to drive it.
