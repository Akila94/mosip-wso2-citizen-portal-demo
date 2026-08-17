# Real WSO2 IS federation + SSO for the citizen portal demo

> **Status: proposed — not yet approved, nothing implemented.**
> This is the plan for joining `citizen-portal-demo-app/` to the working
> `setup-without-bridge/` federation. Read it end to end before starting.

# Important

 - Implement this in a new directory named "citizen-portal-integration" inside "mosip-wso2-citizen-portal-demo". So it is isolated from other folders. All the coding related to this plan should exist in the "citizen-portal-integration" directory.
 - All the implementation should be done according to the best practices.
 - WSO2 secure engineering guides should always be followed.
 - Coding best practices must be followed.
 - Use the "Backend Coding Agent" to do the backend coding if required.
 - Always and always follow best practices of UI, UX, coding, and security.
 - Absolutely no guessing, hallucinating, assuming. No destructive things. Keep the existing implementations and make the plan implementation isolated in the "citizen-portal-integration" directory.

## Context

The repo already proves MOSIP eSignet ↔ WSO2 IS federation at the protocol level
(`setup-without-bridge/` — a custom `EsignetOIDCAuthenticator` JAR), and separately holds a
polished React wireframe app (`citizen-portal-demo-app/`) whose authentication is entirely
fictional: `AuthContext` is two `useState` booleans, every `*Service.ts` resolves in-memory
fixtures behind a `setTimeout`, and there is no `fetch`, no router, no cookie and no URL
change anywhere in `src/`.

The two halves have never been joined. This plan joins them, so the demo stops *describing*
federation and SSO and starts *performing* it:

- The portal's sign-in is owned by WSO2 IS. IS's login page offers **Username & Password**
  and **MOSIP eSignet**; choosing eSignet runs the real federated flow already built in
  `setup-without-bridge/`.
- "Driving Licence Application" and "Renew Vehicle Revenue Licence" become genuinely separate
  OIDC applications in IS (Application A and Application B, per
  `setup-without-bridge/MANUAL-STEPS.md` §3–§4), each with its own client ID, its own token,
  its own audience and its own released claim set.
- Signing in once at the portal lets the citizen enter either micro app with **no second
  login** — a real `/authorize` round trip that IS answers instantly from the `commonauthId`
  SSO session. Entering a micro app cold, with no portal login, produces a login; if an IS
  session already exists it is silent.
- Logout is RP-initiated against IS and propagates to every app via **OIDC back-channel
  logout**, so one sign-out really does end all three sessions.

Tokens must never reach the browser, so a **Backend-for-Frontend** sits between the SPA and
IS. It is written in Go, as is the mock resource server that backs the Driving Licence micro
app and the vehicle registry.

**Hard constraint: `esignet-bridge/` is not touched.** It is the informal variant. All IS-side
work builds on `setup-without-bridge/`.

---

## Decisions taken

| Decision | Choice |
|---|---|
| Front-end packaging | One Vite project, one browser origin, **three independent BFF OIDC client sessions** with path-scoped cookies |
| BFF + mock service stack | **Go**, `github.com/coreos/go-oidc/v3` + `golang.org/x/oauth2` + `github.com/go-chi/chi/v5` |
| IS login page | Step 1 offers **Username & Password *and* MOSIP eSignet** — the citizen genuinely chooses |
| Mock backend | **One** Go resource server with per-audience JWT validation |
| Wireframe D | Session inspector becomes **real**; admin console and AI assistant stay mocked |

---

## Target architecture

```
browser ── http://localhost:8090 ────────────────► citizen-portal-bff  (Go, :8090)
             the ONLY browser-facing origin          │  three OIDC clients
                                                     │  three server-side sessions
                                                     │  tokens never leave this process
                                                     │
                        ┌────────────────────────────┼────────────────────────────┐
                        │                            │                            │
                        ▼                            ▼                            ▼
              WSO2 IS  https://localhost:9443   gov-services-api (Go, :8091)   Vite dev (:5173)
               │  authorize / token             validates the JWT access        proxied through
               │  userinfo / jwks               token: iss, exp, aud, scope      the BFF in dev;
               │  /oidc/logout                  per router                      dist/ served
               │  back-channel logout ─────────►                                statically otherwise
               ▼
      MOSIP eSignet (:3000 UI, :8088 API)
      via EsignetOIDCAuthenticator  (setup-without-bridge/, unchanged)
```

**The BFF is always the origin.** In development it reverse-proxies everything that is not
`/bff/*` to the Vite dev server (WebSocket upgrade passed through, so HMR still works);
otherwise it serves `citizen-portal-demo-app/dist`. This keeps every registered redirect URI
identical in both modes — changing them means redoing IS Console work, so stability matters
more than dev-server convenience.

### Three apps, three clients, one origin

| Route prefix | IS application | BFF namespace | Session cookie |
|---|---|---|---|
| `/` | `Citizen Portal` | `/bff/portal` | `cp_sid`, `Path=/bff/portal` |
| `/apps/driving-licence/*` | `Driving Licence Service` (**Application A**) | `/bff/driving-licence` | `dl_sid`, `Path=/bff/driving-licence` |
| `/apps/revenue-licence/*` | `Vehicle Revenue Licence` (**Application B**) | `/bff/revenue-licence` | `vrl_sid`, `Path=/bff/revenue-licence` |

Cookie `Path` scoping means the browser only ever presents an app's session cookie to that
app's own API namespace. To be documented honestly in the README: **this is isolation by path,
not an origin boundary** — adequate for a demo, and the reason the production answer is three
origins.

---

## Component 1 — `citizen-portal-bff/` (new, Go)

```
citizen-portal-bff/
├── cmd/bff/main.go              wiring, graceful shutdown, timeouts
├── internal/config/             env-only config, one struct per app, validated at boot
├── internal/oidcrp/             one go-oidc Provider (discovered), three RP configs
│                                authorize URL, code exchange, ID-token verify, refresh
├── internal/session/            server-side TTL store + HttpOnly cookie; login transactions
├── internal/httpapi/            chi routers: auth routes + per-app data routes
├── internal/upstream/           typed client for gov-services-api (token injected here)
├── internal/security/           CSRF, returnTo allowlist, log sanitiser, headers, rate limit
├── internal/devproxy/           reverse proxy to Vite / static dist server
└── openapi.yaml                 the BFF contract; generates the SPA's TS client
```

### Routes, per app `{portal | driving-licence | revenue-licence}`

| Route | Behaviour |
|---|---|
| `GET /bff/{app}/login?returnTo=…` | new login transaction (state, nonce, PKCE S256 verifier) stored server-side under a short-TTL HttpOnly `{app}_txn` cookie → 302 to IS `authorization_endpoint` |
| `GET /bff/{app}/callback` | validate `state` against the transaction, exchange code (+`code_verifier`), verify ID token (signature via JWKS, `iss`, `aud`, `exp`, `nonce`), create session, 302 to the validated `returnTo` |
| `GET /bff/{app}/session` | `200` session projection, or `401` |
| `POST /bff/{app}/logout` | destroy local session, return the IS RP-initiated logout URL (`id_token_hint`, `post_logout_redirect_uri`) for the SPA to navigate to |
| `POST /bff/{app}/backchannel-logout` | IS→BFF. Verify logout token, then **destroy every session in the process sharing that `sid`/`sub`** — this is what makes single logout real across all three apps |
| `GET /bff/{app}/step-up?returnTo=…` | re-authorize with `prompt=login` (+ ACR) to raise assurance |
| `GET /bff/{app}/api/…` | explicit, purpose-built data endpoints (below) |

There is deliberately **no generic token-injecting proxy** — an open `/api/*` passthrough would
let the SPA reach any upstream path with the app's access token. Every upstream call is a named
handler.

| App | Data endpoints |
|---|---|
| portal | `catalogue`, `timeline`, `attributes`, `consents`, `documents`, `department-records`, `session-inspector` |
| driving-licence | `config`, `test-slots?week=`, `identity`, `POST applications`, `session-inspector` |
| revenue-licence | `vehicles`, `identity`, `POST vehicles/{id}/renew`, `session-inspector` |

### Talking to IS

`oidc.NewProvider(ctx, "https://localhost:9443/oauth2/token")` — go-oidc appends
`/.well-known/openid-configuration` itself, and that issuer string is exactly what IS puts in
the `iss` claim, so issuer validation lines up with no special-casing. Everything else
(`authorization_endpoint`, `token_endpoint`, `jwks_uri`, `end_session_endpoint` = `/oidc/logout`)
comes from discovery rather than being hardcoded.

IS ships a self-signed certificate. The BFF loads it into a dedicated `x509.CertPool` used only
for the IS client — **never `InsecureSkipVerify`**. This mirrors the invariant both existing
`demo.sh` scripts were reviewed into ("`--insecure` is scoped to `$IS_URL_BASE` only").

### The session projection returned to the SPA

Tokens are excluded by construction — the struct has no token field.

```jsonc
{
  "authenticated": true,
  "clientId": "…",            // this app's client, drives "Released to"
  "appName": "Driving Licence Service",
  "user": { "sub": "…", "name": "…", "givenName": "…", "familyName": "…",
            "email": "…", "phoneNumber": "…", "birthdate": "…", "picture": "…" },
  "assuranceLevel": "substantial",   // derived; see Component 4
  "idp": "MOSIP eSignet",
  "acr": "…", "amr": ["…"], "sid": "…",
  "authTime": 1755…, "expiresAt": 1755…,
  "releasedClaims": { … }     // exactly what this client's scopes released
}
```

---

## Component 2 — `gov-services-api/` (new, Go)

A real OAuth 2.0 resource server, not a fixture file. Every request must carry a JWT access
token issued by IS; the middleware validates signature (IS JWKS, cached with rotation), `iss`,
`exp`/`nbf`, and then a **per-router required audience and scope**:

| Router | Requires | Serves |
|---|---|---|
| `/portal/*` | portal client audience | catalogue, timeline, attributes, consents, documents |
| `/driving-licence/*` | Application A audience + `driving_licence.write` | application config, test slots, submissions |
| `/vehicle-registry/*` | Application B audience + `vehicle_registry.read` | vehicles, renewals |
| `/citizen/profile` | any of the three | the registry record, **projected by the caller's scopes** |

App A's token is genuinely rejected by App B's endpoints. That rejection is worth showing.

**The citizen registry is keyed by the IS `sub`.** eSignet's `sub` is a pairwise PSUT and must
never be displayed as a national ID — so NIC, address and vehicle records come from this
registry, not from the token, which is exactly how a real government service works and matches
the comment already in `revenueLicenceService.ts` ("pulled from the Transport registry against
the citizen's verified NIC"). On first sight of an unknown `sub` the service seeds a record
from the existing fixture data, so the demo works with whatever PSUT eSignet mints.

Fixture content is lifted from the current mock services so the screens look unchanged:
`applicationService.ts` (config, test-week generator, confirmation),
`revenueLicenceService.ts` (vehicle `CAB-4471`), `portalService.ts` (catalogue, timeline,
attributes, consents, documents).

---

## Component 3 — `citizen-portal-demo-app/` (extend)

### 3.1 Routing

Add `react-router-dom` and replace the `Screen` switch in `src/App.tsx`. Screens keep their
existing `onNavigate(screen)` prop contract — a single `Screen → path` map plus a `goTo`
adapter built on `useNavigate()` means **no screen component's props change**. This is the
migration `citizen-portal-demo-app/README.md` already anticipates ("Every screen takes
`onNavigate`, so swapping in `react-router` doesn't touch screen code").

```
/                                   landing        ─┐
/timeline /profile /documents                        ├─ portal app
/services/:serviceId                                 │
/wireframes/*                       reference-only  ─┘
/apps/driving-licence/step/1..4                     ─┐
/apps/driving-licence/confirmation | /error          ├─ Application A
/apps/revenue-licence                               ─┘ Application B
```

Route naming is unified onto one scheme, closing README inconsistency #3. The screens that IS
now owns — `IdentityLoginScreen`, `FederatedIdpScreen`, `ConsentScreen` — leave the real flow
and stay reachable under `/wireframes/*` for the demo narrative, clearly labelled as
wireframes. IS's own login and consent pages take over; IS **Console → Branding** can be
themed to match the portal (optional polish step).

### 3.2 `AuthContext`

Same consumed shape (`isAuthenticated` / `user` / `assuranceLevel`) so all eight current
consumers compile unchanged, plus:

- `isLoading` — a bootstrap `GET /bff/{app}/session` runs on mount; a `RequireAuth` wrapper
  renders a splash while loading rather than flashing "signed out".
- `AuthUser` widened to the real claim set; `AuthProvider` takes the app key so each route
  tree talks to its own BFF namespace.
- `signIn()` → `window.location.assign('/bff/{app}/login?returnTo=' + current path)`.
- `signOut()` → `POST /bff/{app}/logout` then navigate to the returned IS logout URL.
- `raiseAssurance()` → `/bff/{app}/step-up`.

The dangling `user` dependency in the existing `useMemo` (`AuthContext.tsx:33-53`) is fixed
while widening it.

### 3.3 Services → real HTTP

New `src/services/http.ts`: `fetch` with `credentials: 'same-origin'`, the CSRF header on
writes, error normalisation to the `Error(message)` shape `useAsync` already expects, and a
`401 → re-login` interceptor. Every `*Service.ts` keeps its **exact exported signatures**;
only the bodies change from `simulate(fixture)` to a BFF call. Hooks and screens are untouched
because `src/hooks/useAsync.ts` already owns loading/error/reload and has a request-id race
guard.

`identityService.ts` loses `signInLocal` / `signInFederated` / `completeFederatedVerification`
(IS hosts those); `getConsentAttributes(serviceId)` starts honouring `serviceId`.

### 3.4 One client registry, replacing four scattered copies

New `src/config/clients.ts` — `{ appKey, clientName, agency, serviceId, bffBase, routeBase,
scopes[] }` for the three apps. It replaces the hardcoded identity in
`VehicleRevenueLicenceScreen.tsx`'s `NARROW_IDENTITY` literal, `applicationService.ts`'s
`verifiedIdentity`, `sessionInspectorService.ts`'s `byClient` keys, and `portalService.ts`'s
consent fixtures — closing README inconsistencies #1 and #7. `MicroAppHeader`'s `'J. Doe'`
fallback is removed; unauthenticated means no identity panel.

### 3.5 Session inspector — the SSO proof

`useSessionInspector` reads `/bff/{app}/api/session-inspector`. Side by side the two micro apps
now show the **same** `sub`, `sid`, `idp`, `acr`, `amr` and `auth_time`, and **different** `aud`
and released claims — from real tokens. `SessionInspectorData.claims` widens from
`Record<string, string | string[]>` to `Record<string, unknown>` (it currently cannot hold the
numeric `exp`/`iat`/`auth_time`), and "remaining lifetime" becomes a live countdown off the
real `exp` instead of the static `'51 min'`.

The `sid` claim is present in IS ID tokens **by default** for the authorization-code flow, which
is what makes both the cross-app "same session" row and back-channel logout work with no extra
configuration. The "clients in session" row is computed by the BFF from its own three sessions
sharing that `sid` — reliable and always available. IS's `GET /api/users/v1/me/sessions` does
return an `applications[]` list and would be a richer source, but it needs the `internal_login`
scope, which a JIT-provisioned citizen will not normally hold; treat it as an optional
enhancement, not the primary path.

---

## Component 4 — WSO2 IS Console work

Rewrites `setup-without-bridge/MANUAL-STEPS.md` §3–§4 into three registrations, and updates the
`§11–§13 applications` row of `setup-without-bridge/RUNBOOK-DELTA.md` to match. The runbook
itself (`esignet-bridge/…`) is not edited, per the repo's own rule.

**Template change:** these are **Traditional Web Application** (`oidc-web-application`,
confidential) registrations, not the Single-Page Application template §3 currently uses — a BFF
holds a client secret, which is the entire point of the pattern.

| | Portal | Application A | Application B |
|---|---|---|---|
| Name | `Citizen Portal` | `Driving Licence Service` | `Vehicle Revenue Licence` |
| Authorized redirect URLs | `…/bff/portal/callback` **and** `http://localhost:8090/` | `…/bff/driving-licence/callback` **and** `…/apps/driving-licence` | `…/bff/revenue-licence/callback` **and** `…/apps/revenue-licence` |
| Back-channel logout URL | `…/bff/portal/backchannel-logout` | `…/bff/driving-licence/backchannel-logout` | `…/bff/revenue-licence/backchannel-logout` |
| Scopes | `openid profile email` | `+ address`, `driving_licence.write` | `+ vehicle_registry.read` |

Three settings are **not** the template default and must be changed explicitly — each was
verified against the component versions IS 7.3.0 actually ships, and getting any of them wrong
produces a confusing failure:

- **Access token type → JWT.** The `oidc-web-application` template omits the `accessToken`
  block, so it falls back to `"Default"` = **opaque**. (The widely repeated claim that IS 7.x
  issues JWTs by default is wrong for this template.) Without this the resource server has
  nothing to validate. Console → **Protocol** → **Access Token** → **Token type**.
- **Mandatory PKCE → on.** Unlike the SPA and mobile templates, `oidc-web-application` ships no
  `pkce` block, so PKCE is available but *not* enforced.
- **Post-logout redirect URIs are not a separate field.** IS validates
  `post_logout_redirect_uri` against the app's **Authorized redirect URLs** list, which is why
  each app above registers two URLs.

Also: **Login Flow Step 1 = Username & Password + MOSIP eSignet** on all three (the change from
today's eSignet-only Step 1 — a single step may hold several sign-in options, with the one
documented exclusion that Identifier First cannot share a step with Username & Password); one
local demo user for the non-MOSIP path; custom OIDC scopes defined and bound to claims so the
per-app claim difference is enforced by IS, not by the BFF. Note the audience list lives under
**Protocol → ID Token → Audience** but is applied to the JWT *access* token's `aud` as well.

### Deriving `assuranceLevel` honestly

IS emits `amr` whenever authenticator references exist, but emits `acr` **only when a value was
actually resolved** — a plain app with no adaptive script and no `acr_values` gets an ID token
with no `acr` at all. So:

- **Primary:** the BFF maps `amr` → assurance (`BasicAuthenticator` → `basic`; the eSignet
  federated authenticator → `substantial`). This works with zero extra IS configuration.
- **Optional hardening:** an ACR-based conditional-authentication script on each app, branching
  on `context.requestedAcr`, so `acr` is real and step-up is enforced by IS rather than
  requested politely. `[[authentication_context.method_refs]]` in `deployment.toml` can also
  remap `amr` from internal authenticator names to RFC 8176 values.

Step-up uses `prompt=login` (and optionally `max_age`), both of which IS honours at
`/oauth2/authorize`.

---

## Component 5 — orchestration and repo hygiene

- New root `portal-demo.sh`, carrying the same hardening invariants the two existing `demo.sh`
  scripts were reviewed into: whitelisted subcommands, `.env` **parsed not sourced**, `chmod
  600` on every run, no secrets in `argv`, `sanitize()` on any remote text, PID-ownership check
  before any `kill`. Subcommands: `setup`, `build`, `start`, `stop`, `restart`, `status`,
  `preflight`, `logs`, `clean`. It checks that IS and eSignet are already up and points at
  `setup-without-bridge/demo.sh` if not — it never starts them itself.
- `.gitignore`: add `bin/`, `dist/`; **and fix the unanchored `package-lock.json` rule** by
  anchoring it to `/esignet-bridge/package-lock.json`, so the SPA's and any future lockfile
  become committable (`CLAUDE.md` records that swallowing the SPA lockfile was an accident, not
  a decision).
- Root `README.md` gains the third component alongside the two federation variants.

---

## Component 6 — configuration

Configuration is **environment variables only**, with working localhost defaults, matching the
convention `esignet-bridge/server.js` already established ("there is no config file"). Nothing
is read from a config file; nothing secret is passed on a command line.

### Port map (additions to the existing map)

`8090` and `8091` are chosen because everything else is taken:
`3000` eSignet UI · `4000` bridge · `5173` Vite · `5455` postgres · `6379` redis ·
`8082` mock identity · `8088` eSignet API · `9443` WSO2 IS.

| Port | Component |
|---|---|
| `8090` | `citizen-portal-bff` — **the only browser-facing origin** |
| `8091` | `gov-services-api` — never reached by the browser |
| `5173` | Vite dev server — reached only through the BFF's dev proxy |

### `citizen-portal-bff`

| Variable | Default | Notes |
|---|---|---|
| `BFF_PORT` | `8090` | |
| `BFF_PUBLIC_URL` | `http://localhost:8090` | **Every redirect URI and post-logout URI is derived from this.** Changing it means redoing Console work — see the warning below |
| `IS_ISSUER` | `https://localhost:9443/oauth2/token` | go-oidc discovers everything else from here |
| `IS_CA_FILE` | `../setup-without-bridge/wso2is-7.3.0/repository/resources/security/wso2carbon.pem` | IS's self-signed cert, loaded into a dedicated `x509.CertPool`. **No `InsecureSkipVerify` anywhere** |
| `PORTAL_CLIENT_ID` / `_SECRET` / `_SCOPES` | — / — / `openid profile email` | from the Console, §Component 4 |
| `DL_CLIENT_ID` / `_SECRET` / `_SCOPES` | — / — / `openid profile email address driving_licence.write` | Application A |
| `VRL_CLIENT_ID` / `_SECRET` / `_SCOPES` | — / — / `openid profile email vehicle_registry.read` | Application B |
| `SERVICES_API_URL` | `http://localhost:8091` | |
| `SESSION_IDLE_TIMEOUT` | `60m` | matched to the `[session.timeout]` block `setup-without-bridge/demo.sh` already writes into `deployment.toml`; a shorter value here silently defeats the SSO demo |
| `SESSION_MAX_ENTRIES` | `5000` | bounded store, same cap as the bridge's `TtlMap` |
| `LOGIN_TXN_TTL` | `5m` | lifetime of the state/nonce/PKCE transaction |
| `DEV_PROXY_TARGET` | `http://localhost:5173` | when set, proxy non-`/bff/*` to Vite; when empty, serve `STATIC_DIR` |
| `STATIC_DIR` | `../citizen-portal-demo-app/dist` | used only when `DEV_PROXY_TARGET` is empty |
| `COOKIE_SECURE` | derived from `BFF_PUBLIC_URL`'s scheme | explicit override for a TLS front |
| `LOG_LEVEL` | `info` | |

The three `*_CLIENT_SECRET` values are the only secrets. They are validated at boot — the
process **refuses to start** with a missing or empty client ID or secret, rather than failing
later inside a redirect where the error is unreadable.

### `gov-services-api`

| Variable | Default | Notes |
|---|---|---|
| `SERVICES_PORT` | `8091` | |
| `IS_ISSUER` | `https://localhost:9443/oauth2/token` | JWKS URI comes from discovery, cached with rotation |
| `IS_CA_FILE` | same as the BFF | |
| `PORTAL_CLIENT_ID` / `DL_CLIENT_ID` / `VRL_CLIENT_ID` | — | the **expected audiences** — this is what makes App A's token fail on App B's router |
| `LOG_LEVEL` | `info` | |

No secrets: a resource server validates tokens, it never presents credentials.

### `citizen-portal-demo-app`

**Deliberately zero configuration.** The SPA only ever calls same-origin `/bff/…` paths, so no
client ID, no issuer URL and no secret is ever compiled into the bundle — a property worth
stating in the README, since it is the main practical argument for the BFF pattern. This also
means the app introduces no `VITE_*` convention, of which it currently has none.

### Where the secrets live

A root `.env`, created by `portal-demo.sh setup` and `chmod 600` on **every** run, holding only
the three client secrets and the three client IDs. It is **parsed, never sourced** — using `.`
or `source` would execute its contents as shell code, which is the first finding the existing
`demo.sh` review fixed. A committed `.env.example` documents every variable above with its
default and an empty value for each secret.

> **The same trap the bridge variant has with `BRIDGE_API_KEY` applies here.** `.env` must match
> what is registered in the Console. If `.env` is lost, the next `setup` cannot re-mint client
> secrets — they are issued by IS — so the recovery is to regenerate each secret in the Console
> and paste it back. `MANUAL-STEPS.md` gets a row for this in its "when you have to redo the
> manual steps" table.

### IS-side configuration

Already handled: the `[session.timeout]` block in `deployment.toml` is appended by
`setup-without-bridge/demo.sh`. Optionally added by this work:
`[[authentication_context.method_refs]]`, to remap `amr` from internal authenticator names to
RFC 8176 values — cosmetic, and only if the ACR hardening path is taken.

---

## Phasing

**Build this in six milestones, not one pass.** Each ends in a state you can actually run and
show, and each retires a specific class of risk. Rationale in the reply that accompanied this
plan; the short version is that the unknowns are concentrated in one place — the real IS round
trip — and everything else is ordinary work that only becomes hard if it is debugged at the
same time.

TDD applies to the Go packages throughout: tests first.

| # | Milestone | Ends when | Retires |
|---|---|---|---|
| **M1** | **Walking skeleton.** Register **only** `Citizen Portal` in IS. Build the BFF's config, session store, login-transaction store, the full OIDC round trip and the security middleware. No SPA changes, no resource server — `/bff/portal/session` returns JSON to a browser | a real citizen logs in through IS → MOSIP eSignet → OTP → and the BFF prints the real ID-token claims | **The riskiest ~80%**: are the Console settings right, does the JWT/PKCE/logout config behave, and — critically — **what do `amr`, `acr` and `sid` actually contain**, which the assurance mapping depends on |
| **M2** | **Three clients, SSO and single logout.** Register Applications A and B, add their RP configs, implement RP-initiated and back-channel logout with `sid`-linked session destruction | opening app B's login URL after app A's login completes with **no prompt**, and one logout kills all three sessions | The SSO and logout mechanics, still with zero UI in the way |
| **M3** | **Resource server.** `gov-services-api` with per-audience/scope validation, the `sub`-keyed citizen registry, and the BFF's typed upstream client and data endpoints | `curl` proves App A's token is accepted by `/driving-licence/*` and **rejected** by `/vehicle-registry/*` | The token-audience story |
| **M4** | **SPA routing migration.** `react-router`, the `Screen → path` map, route trees. **Mock services still in place, behaviour unchanged.** A separate commit on its own | `npm run build` passes and every screen is reachable by URL | Nothing about auth — which is exactly the point. A large mechanical diff stays separable from a behavioural one |
| **M5** | **SPA goes real.** `AuthContext` bootstrap and redirects, `http.ts`, service bodies swapped, `clients.ts` registry, then the session inspector | the full demo runs end to end, including the two micro apps and the inspector's side-by-side claim comparison | The integration itself |
| **M6** | **Orchestration, docs, security pass.** `portal-demo.sh`, the `MANUAL-STEPS.md` / `RUNBOOK-DELTA.md` rewrite, root `README.md`, `.gitignore` fix, `gosec` + `govulncheck` green | `./portal-demo.sh preflight` ends `failed=0` from a cold start | Reproducibility — the thing this repo cares most about |

**This inverts the build order a first reading suggests.** Starting with `gov-services-api`
because it is self-contained and testable without IS is the comfortable choice and the wrong
one: it is the *lowest*-risk component, so building it first defers every real unknown and
risks discovering at M3 that an assumption about IS was wrong after a week of work was built on
it. M1 is deliberately the least comfortable thing to build first.

---

## Security requirements (WSO2 secure engineering guidelines)

Non-negotiable, and each gets a test:

- **No token ever reaches the browser.** Session cookies are `HttpOnly`, `SameSite=Lax`,
  `Secure` when served over TLS; the session projection struct has no token field.
- **§1.22 unvalidated redirects.** `returnTo` must match `^/[A-Za-z0-9._~/-]*$` *and* fall
  within the requesting app's own route prefix. `post_logout_redirect_uri` comes from a fixed
  allowlist, never from the request.
- **§1.7 log forging.** Every externally supplied string passes a `SanitizeForLog` helper
  (CR/LF/tab stripped, length capped) before any log or error — mirroring `LogSanitizer.clean()`
  in the Java authenticator and `clean()` in the bridge.
- **CSRF.** Double-submit token on every state-changing route, compared in constant time.
- **Token validation.** ID token: signature via JWKS with rotation, `iss`, `aud`, `exp`,
  `nonce`. Logout token: signature, `iss`, `aud`, the back-channel `events` claim, `sid`/`sub`,
  and `jti` replay rejection. Access token at the resource server: signature, `iss`,
  `exp`/`nbf`, required audience, required scope.
- **Secrets.** Client secrets via environment or a `0600` file — never `argv` (`ps` is world
  readable), never a committed file.
- **Hardening.** Request body caps, server read/write/idle timeouts, rate limiting on the auth
  routes, `crypto/rand` 256-bit session IDs, a bounded session store with TTL eviction,
  security headers + CSP on served HTML, `X-Frame-Options: DENY`.
- **Tooling.** `gosec` and `govulncheck` wired and green (guidelines §8.2 static analysis and
  §1.17 known-vulnerable components); `go vet` clean.

---

## Verification

**Automated**

```bash
cd gov-services-api    && go test ./... && go vet ./... && gosec ./... && govulncheck ./...
cd citizen-portal-bff  && go test ./... && go vet ./... && gosec ./... && govulncheck ./...
cd citizen-portal-demo-app && npm run build      # tsc -b must pass
./portal-demo.sh preflight                        # must end failed=0
```

Go tests cover: session TTL and eviction, `returnTo` rejection (absolute URLs,
protocol-relative `//evil`, `..` traversal, cross-app prefixes), log sanitiser against CRLF,
CSRF mismatch, ID-token verification against a bad signature / wrong `aud` / wrong `nonce`,
logout-token replay, and audience enforcement (App A's token rejected by App B's router).

**Manual, end to end** — the demo itself, in a fresh incognito window:

1. `cd setup-without-bridge && ./demo.sh start` → `./demo.sh preflight` green.
2. `./portal-demo.sh start` → `http://localhost:8090`.
3. Click **Sign in** (top right) → IS login page shows **both** options → choose **MOSIP
   eSignet** → eSignet at `:3000` → individual ID `8267411571`, **Get OTP**, `111111`, approve
   → back at the portal, header shows the real name from the eSignet claims.
4. `./demo.sh logs is` shows no `ESIGNET-650xx`; Console → **Users** shows the JIT-provisioned
   user whose username is the eSignet `sub`.
5. Start the **driving licence** application → full-page hop through
   `9443/oauth2/authorize` and straight back, **no login prompt** — SSO proven. The verified
   identity panel is populated from the real token.
6. Open **Renew vehicle revenue licence** → again no prompt. Open the **session inspector** on
   both micro apps: same `sub`/`sid`/`idp`/`acr`/`amr`, different `aud`, different claims.
7. **Cold entry:** fresh incognito window straight to
   `http://localhost:8090/apps/driving-licence/step/1` → redirected to IS → sign in → landed
   in the micro app. Repeat with an existing IS session → silent.
8. **Logout:** sign out from any app → IS `/oidc/logout` → back-channel logout fires → all
   three sessions dead; re-entering any app re-prompts. Verify in the BFF log that three
   sessions were destroyed by one logout token.
9. **Negative:** call a `/vehicle-registry/*` endpoint with Application A's token → `403`.
10. Confirm with devtools that **no `Authorization` header and no token** is ever present in a
    browser-originated request, and that each `*_sid` cookie is `HttpOnly` and sent only to its
    own `/bff/{app}` path.

**Known gaps to state openly during the demo** (carried forward from the existing docs, still
true): eSignet has no logout endpoint, so sign-out ends the IS and app sessions but does not
propagate to eSignet; eSignet's `sub` is a pairwise PSUT and is never shown as a national ID;
one browser origin means the micro-app boundary is path-scoped, not origin-scoped.

---

## Appendix — IS 7.3.0 facts this plan rests on

Verified against the component versions `product-is` v7.3.0 pins
(`identity-inbound-auth-oauth` v7.4.99, `carbon-identity-framework` v7.10.156, Console
`@wso2is/console@3.3.29`) — not recalled. Do not re-derive these during implementation.

| Fact | Value |
|---|---|
| Issuer / discovery | `https://localhost:9443/oauth2/token` (+ `/.well-known/openid-configuration`) |
| authorize / token / userinfo / jwks | `/oauth2/authorize`, `/oauth2/token`, `/oauth2/userinfo`, `/oauth2/jwks` |
| end_session / introspect / revoke | `/oidc/logout`, `/oauth2/introspect`, `/oauth2/revoke` |
| RP-initiated logout params | `client_id` (recommended), `id_token_hint`, `post_logout_redirect_uri` (only with `id_token_hint`), `state` |
| Back-channel logout | Supported and advertised; logout token carries `iss`, `sub`, `aud`, `iat`, `exp`, `jti`, `sid`, `events`. Front-channel also supported |
| `sid` in ID token | Present by default for `/authorize` and the `authorization_code`/`refresh_token` grants; absent for client_credentials/password |
| Traditional Web App defaults | Client secret issued; grant `authorization_code`; **PKCE not mandatory**; **access token opaque**; auth methods `client_secret_basic`/`_post`, `private_key_jwt`, `tls_client_auth` |
| JWT access token claims | `iss` = the issuer above; `aud` = `[client_id] + configured audiences`; plus `client_id`, `azp`, `scope` (space-separated) |
| `acr` / `amr` | `amr` emitted when authenticator refs exist (internal names by default); `acr` emitted **only if resolved**. `acr_values` on `/authorize`; ACR adaptive auth via `context.requestedAcr` |
| `prompt` / `max_age` | Both honoured; `prompt=login` forces re-authentication |
| CORS | Not needed — the browser never calls IS's APIs directly under the BFF pattern |
| Session API | `GET /api/users/v1/me/sessions`, needs `internal_login`; returns `applications[]` |
