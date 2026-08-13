# MOSIP eSignet ↔ WSO2 Identity Server — Federation & SSO Demo

Federates **WSO2 Identity Server 7.3.0** to **MOSIP eSignet v1.8.0**, then proves single sign-on across two applications: a citizen logs in once with their national ID, and every subsequent service signs them in from the WSO2 session without touching eSignet again.

> **This is a demo environment, not a product.** The bridge service is a translating middlebox — correct for a demo, wrong for production. See [Known limitations](#known-limitations) before you show it to anyone.

---

## Table of contents

- [Start here](#start-here)
- [What problem this solves](#what-problem-this-solves)
- [Architecture](#architecture)
- [Repository layout](#repository-layout)
- [Prerequisites](#prerequisites)
- [Quick start](#quick-start)
- [First-time setup](#first-time-setup)
- [Running the demo](#running-the-demo)
- [The bridge in detail](#the-bridge-in-detail)
- [Configuration reference](#configuration-reference)
- [Troubleshooting](#troubleshooting)
- [Known limitations](#known-limitations)
- [Security notes](#security-notes)

---

## Start here

| Document | What it is |
|---|---|
| [`esignet-wso2-is-federation-runbook.md`](./esignet-wso2-is-federation-runbook.md) | The authoritative spec — every step, port, credential and source citation |
| [`demo.sh`](./demo.sh) | Everything scriptable: `setup`, `start`, `stop`, `restart`, `status`, `preflight`, `logs`, `apikey`, `clean` |
| [`MANUAL-STEPS.md`](./MANUAL-STEPS.md) | The WSO2 Console work `demo.sh` cannot do, plus failure recovery |

**[`esignet-wso2-is-federation-runbook.md`](./esignet-wso2-is-federation-runbook.md) is the authoritative document.** Every step, port, credential, troubleshooting entry and source citation lives there, and it is what a presenter follows live.

This README is the orientation layer: what the pieces are, why they exist, and how to get running fast. When the two disagree, **the runbook wins** — and please fix this file.

| I want to… | Go to |
|---|---|
| Understand the design before touching anything | [What problem this solves](#what-problem-this-solves) |
| Get it running on a machine that's already set up | [Quick start](#quick-start) → `./demo.sh start` |
| Set it up from scratch the first time | `./demo.sh setup`, then [`MANUAL-STEPS.md`](./MANUAL-STEPS.md) |
| Do the WSO2 Console configuration | [`MANUAL-STEPS.md`](./MANUAL-STEPS.md) — the part no script can do |
| Present the demo | Runbook §16 (demo script, ~4 minutes) |
| Fix something that broke | [Troubleshooting](#troubleshooting) → runbook §17 |
| Edit the bridge code | [The bridge in detail](#the-bridge-in-detail) + `CLAUDE.md` |

---

## What problem this solves

You would expect to federate WSO2 IS to eSignet using the stock **Standard-Based IdP → OpenID Connect** connection in the WSO2 Console. **You cannot.** There are two hard blockers, both confirmed by reading the source:

| # | Blocker | Why it's fatal |
|---|---|---|
| 1 | eSignet **mandates `private_key_jwt`** at the token endpoint. | `client_assertion` is `@NotBlank` on eSignet's request DTO and there is **no `client_secret` code path at all**. WSO2's outbound OIDC connector implements only `client_secret_basic` and `client_secret_post` — the string `client_assertion` does not appear anywhere in that repository. |
| 2 | eSignet returns UserInfo as a **signed JWT**, always. | WSO2's connector calls `JSONUtils.parseJSON()` on the UserInfo response and throws. Even plain JSON from a plugin gets signed into a JWS before return. |

Two dead ends worth naming, so nobody burns a day on them:

- **You cannot skip UserInfo.** eSignet puts *no user claims in the ID token*, so omitting the UserInfo endpoint leaves you with no attributes.
- **You cannot enable client secrets via config.** Changing `mosip.esignet.supported.client.auth.methods` only alters what the discovery document *advertises*. It does not create a code path that accepts a secret.

### The solution: a small bridge service

WSO2 IS 7.1+ ships a **service-based custom authenticator** — you register an HTTP endpoint in the Console and IS calls it with a JSON contract. **No Java, no OSGi bundle, no JAR, no server restart.** The contract supports a `redirect` operation, which is exactly what OIDC federation needs.

The bridge (`demo-setup/esignet-bridge/server.js`, ~240 lines) implements that contract on the IS side and speaks proper OIDC — `private_key_jwt` + PKCE + JWS UserInfo verification — on the eSignet side.

---

## Architecture

```
Browser
  │  1. open App A
  ▼
WSO2 Identity Server 7.3.0 ─────────────── https://localhost:9443
  │  2. POST /authenticate {flowId, event, allowedOperations:[{op:"redirect"}]}
  ▼
Bridge (Node.js) ───────────────────────── http://localhost:4000
  │  3. reply {actionStatus:"INCOMPLETE", operations:[{op:"redirect", url:<eSignet /authorize>}]}
  ▼
eSignet UI ─────────────────────────────── http://localhost:3000
  │     citizen enters national ID → receives OTP → consents
  ▼
eSignet service ────────────────────────── http://localhost:8088
  │  4. redirect to http://localhost:4000/callback?code=..&state=..
  ▼
Bridge
  │  5. POST /oauth/v2/token   (private_key_jwt client assertion + PKCE)
  │  6. GET  /oidc/userinfo    (signed JWT — verified against eSignet's JWKS)
  │  7. redirect to https://localhost:9443/t/carbon.super/commonauth?flowId=..
  ▼
WSO2 Identity Server
  │  8. POST /authenticate again, same flowId
  │     bridge replies {actionStatus:"SUCCESS", data:{user:{id, claims:[...]}}}
  │  9. JIT-provisions the user, creates the SSO session (commonauthId cookie)
  ▼
App A logged in.   App B then logs in from the IS session — eSignet is never called again.
```

**Step 9 is the demo.** eSignet authenticates once; WSO2 IS holds the session and every subsequent service federates against it.

### Port map

| Component | Port | URL |
|---|---|---|
| eSignet UI | 3000 | `http://localhost:3000` |
| Bridge | 4000 | `http://localhost:4000` |
| PostgreSQL | 5455 | (internal) |
| Redis | 6379 | (internal) |
| Mock identity system | 8082 | `http://localhost:8082/v1/mock-identity-system` |
| eSignet service | 8088 | `http://localhost:8088/v1/esignet` |
| WSO2 IS | 9443 | `https://localhost:9443` |

> ⚠️ **Everything must run on one host.** The browser and all services have to resolve `localhost` to the same machine. Do not split components across machines.

---

## Repository layout

| Path | What it is |
|---|---|
| `esignet-wso2-is-federation-runbook.md` | **The runbook.** Primary artifact — every step, credential and source citation. |
| `demo-setup/esignet-bridge/` | **The only hand-written code.** Node.js ESM, `express` + `jose`. |
| `demo-setup/esignet/` | Vendored upstream clone of `mosip/esignet` at tag `v1.8.0`. **Read-only.** |
| `demo-setup/wso2is-7.3.0/` | Unpacked WSO2 IS distribution. Config in `repository/conf/deployment.toml`. |
| `CLAUDE.md` | Guidance for Claude Code sessions in this repo. |

> 🚫 **Never edit `demo-setup/esignet/`.** It is upstream source, present only as evidence — the runbook's Appendix A maps each technical claim to a specific file in it (`TokenRequest.java`, `UserInfoResponseHelper.java`, `application-*.properties`).

Inside `demo-setup/esignet-bridge/`:

| File | Role |
|---|---|
| `server.js` | The bridge. The whole service, one file. |
| `genKeys.js` | One-shot RSA keypair generator. Run once, during setup. |
| `private.jwk.json` | Signing key. **Never leaves this directory.** |
| `public.jwk.json` | Registered with eSignet. |
| `preflight.sh` | Pre-demo health check. **Not yet checked in** — source is in runbook §15. |

---

## Prerequisites

```bash
docker --version          # Docker Engine
docker compose version    # Compose v2
node --version            # v22 — demo.sh hard-fails below it
java -version             # JDK 11–21 (21 recommended)
curl --version
python3 --version         # JSON pretty-printing in checks
```

Also required:

- Ports **3000, 4000, 5455, 6379, 8082, 8088, 9443** free
- ~**6 GB RAM**, ~**10 GB disk**
- Outbound internet on first run (Docker image pulls, `npm install`, IS download)

---

## Quick start

Everything scriptable is in **[`demo.sh`](./demo.sh)** at the repository root — one script, one terminal, health-gated waits included. The Console work it cannot do is in **[`MANUAL-STEPS.md`](./MANUAL-STEPS.md)**.

**Node 22 is required** — `demo.sh` refuses to start the bridge on anything older, and `express@5` / `jose@6` need ≥18 regardless:

```bash
nvm use 22        # or however you manage Node
node -v           # must print v22.x
```

### From a fresh clone

```bash
./demo.sh setup        # runbook Steps 1–6: clone eSignet, fetch IS, keys, citizen, client
./demo.sh start        # eSignet + WSO2 IS + bridge, in order, with waits
./demo.sh status       # all five services should read OK
./demo.sh apikey       # the Console asks for this
```

Then work through **[`MANUAL-STEPS.md`](./MANUAL-STEPS.md)** — the custom authenticator, JIT provisioning, both applications, and the first login. Those are one-time and persist in the IS H2 databases.

`setup` is **idempotent**: every phase skips work already done, so if it fails part-way, just run it again. The keypair is deliberately never regenerated once it exists, because a new key invalidates the registered eSignet client.

### Every demo day after that

```bash
./demo.sh start
./demo.sh preflight    # must end failed=0
```

**Every preflight line must read `OK`.** On success it also restarts the bridge, so the `PREFLIGHT` entry leaves its in-memory flow map and the log is clean for the demo. If anything fails, fix it before presenting.

Then open a **fresh incognito window** — a stale `commonauthId` cookie makes the SSO proof look like it already happened.

### The rest of the commands

```bash
./demo.sh stop         # all three services, all data kept
./demo.sh restart
./demo.sh logs bridge  # or: is | esignet
./demo.sh clean        # logs, pid files, the IS zip — keeps keys, data, Console config
./demo.sh clean --all  # back to a fresh clone (asks YES; destroys citizen + client)
./demo.sh reset-wso2   # only the IS config and users (asks YES)
```

`stop` uses `docker compose stop`, never `down -v` — that would destroy the test citizen **and** the registered OIDC client.

> **The four-terminal manual equivalent** — eSignet compose, `wso2server.sh`, `node server.js`, preflight — is in runbook §19. Worth knowing when you need to watch one service closely or something refuses to start.

### Test citizen

| | |
|---|---|
| Individual ID | `8267411571` |
| OTP | `111111` — hardcoded in the mock identity system |
| Password (password ACR) | `Mosip@123` |

The citizen **persists in the Postgres volume** across restarts. Running `docker compose down -v` destroys the volume and forces a re-run of runbook Steps 2 and 4.

---

## First-time setup

Full detail is in the runbook — this is the map so you know what you're walking into. Steps 1–6 and 11.2 are automated by `./demo.sh setup`; the Console steps are yours, in [`MANUAL-STEPS.md`](./MANUAL-STEPS.md).

| Step | What happens | Who does it | Runbook |
|---|---|---|---|
| 1 | Start the eSignet stack; verify the discovery document advertises `private_key_jwt` | `demo.sh setup` | §4 |
| 2 | Create the test citizen via the mock identity API | `demo.sh setup` | §5 |
| 3 | Generate the RSA keypair (`node genKeys.js`) | `demo.sh setup` | §6 |
| 4 | Register the OIDC client with eSignet (CSRF token, then create client) | `demo.sh setup` | §7 |
| 5 | Install and start WSO2 IS 7.3.0 | `demo.sh setup` | §8 |
| 6 | Run the bridge; generate an API key with `openssl rand -base64 32` | `demo.sh setup` | §9 |
| 7 | Register the custom authenticator in the IS Console | **you** — [`MANUAL-STEPS.md`](./MANUAL-STEPS.md) §1 | §10 |
| 8 | Create application A and add **Sign in with eSignet** to login step 1 | **you** — §3 | §11 |
| 9 | First end-to-end login | **you** — §5 | §12 |
| 10 | Create application B — **this is what makes it an SSO demo** | **you** — §4 | §13 |
| 11 | Enable JIT provisioning; raise the session idle timeout to 60m | JIT: **you** — §2 · timeout: `demo.sh setup` | §14 |

### Three traps that will cost you time

1. **Stale `requestTime` (Steps 2 and 4).** eSignet rejects a request whose `requestTime` is not close to now. The runbook deliberately keeps **both** a `not working` block (hardcoded timestamp) and a `working` block (live `$(date -u +%Y-%m-%dT%H:%M:%S.000Z)`). **Keep both** — the failing one documents a real trap. Always use the `working` variant.

2. **Save the API key immediately.** You paste it in two places: `BRIDGE_API_KEY` and the IS Console. **The Console will not show it again after you save.**

3. **CSRF cookie extraction (Step 4.1) is fragile** and was never testable against a live instance. If it misbehaves, use the Postman collection fallback in runbook §7.2 — that is the officially documented path.

> **Keys already exist in this tree.** `private.jwk.json` and `public.jwk.json` are present, so you can normally skip Step 3. Re-running `genKeys.js` overwrites them and **every token request will fail with `invalid_client` until you re-register the client with eSignet** (Step 4).

---

## Running the demo

Runbook §16 has the full four-minute script. The beats that matter:

1. **Frame it (20s)** — eSignet is the national ID authentication layer; WSO2 IS is the broker in front of government services. Complementary, not competing.
2. **Log in to app A (90s)** — pause on the eSignet screen (it's MOSIP's UI, not WSO2's), then pause on the **consent screen**. The citizen sees each attribute by name and chooses what to release. That is eSignet's genuine differentiator for government work.
3. **Show what arrived (40s)** — attributes in app A, then IS Console → **Users**: the citizen was created automatically on first login. No pre-registration, no batch import.
4. **The SSO proof (60s)** — open app B. **Already logged in, instantly.** Then point at the bridge terminal: **exactly one `[login-ok]` line.** The bridge was never called a second time. That is the federation boundary made visible.
5. **Close (30s)** — eSignet supplies the credential, inclusive auth factors and consented data sharing; IS supplies the session, SSO, MFA chaining and app-level authorization.

**Before the audience arrives:** run preflight (all `OK`) → restart the bridge → open a **fresh incognito window** (no stale `commonauthId` cookie) → both app URLs in separate tabs.

### Confirming a login worked

| Where | What you should see |
|---|---|
| Bridge terminal | `[login-ok] flow=<id> sub=<PSUT> claims=6` |
| IS Console → Users | A new user whose username is the eSignet `sub` |
| That user's profile | Email, first name, last name, mobile populated |

---

## The bridge in detail

### `genKeys.js` — one-shot key generation

Generates an **RSA-2048 / RS256** keypair and writes both halves as JWKs. Not a free choice: eSignet accepts only RSA keys for client registration, and its signing algorithm list is `{RS256, PS256, ES256}`.

- **Both halves share one `kid`** (a random UUID). The bridge puts that `kid` in the header of every client assertion; eSignet uses it to select the registered key. Mismatched `kid`s mean failed verification.
- `extractable: true` is required, or `exportJWK()` cannot serialize the private key.
- It prints the public JWK to stdout, ready to paste into the client-registration call.

`preflight.sh` compares `kid` and modulus `n` across the two files precisely to catch a half-finished regeneration before a live demo.

### `server.js` — the bridge

Stateful across two IS calls, keyed by `flowId`:

**① IS `POST /authenticate`** — API-key gate first. For a new `flowId`, mint `state` (256 bits), `nonce`, and a PKCE verifier; store `{tenant, verifier}` under `flowId` and `state → flowId` alongside; reply `INCOMPLETE` with a `redirect` operation pointing at eSignet's `/authorize`. **The verifier never goes over the wire** — that is the point of PKCE.

**② The citizen authenticates at eSignet.** The bridge is idle.

**③ eSignet redirects to `GET /callback?code&state`** — the real work:

- `state → flowId`, then **deleted immediately**. Single use; a forged or replayed `state` gets a flat `400`.
- Mint a **fresh client assertion** and `POST /oauth/v2/token` with the code, verifier and matching `redirect_uri`. *This is the call WSO2's stock connector cannot make.*
- `GET /oidc/userinfo`, read the body **as text**, and verify the JWS against eSignet's remote JWKS. *This is the response WSO2's stock connector cannot parse.*
- Map claims to WSO2 URIs, then redirect the browser to `${IS}/t/${tenant}/commonauth?flowId=..`.

Anything that throws is caught, recorded as `FAILED` with a sanitized reason, and the browser is redirected back to IS anyway — so the user sees a proper IS error page instead of a hung tab.

**④ IS `POST /authenticate` again**, same `flowId` — return the stored `SUCCESS` + user or `FAILED` + reason, and delete the entry. IS then JIT-provisions the user and issues the `commonauthId` SSO cookie.

#### Claim mapping

| eSignet | WSO2 |
|---|---|
| `sub` | `.../username` (and the user `id`) |
| `email` | `.../emailaddress` |
| `given_name` ‖ `name` | `.../givenname` |
| `family_name` | `.../lastname` |
| `phone_number` ‖ `phone` | `.../mobile` |
| `birthdate` | `.../dob` |
| `gender` | `.../gender` |

Empty values are skipped, so IS never receives blank attributes. All URIs are under `http://wso2.org/claims/` and all exist in a clean 7.3.0.

#### Client assertion invariants — eSignet enforces these strictly

| Claim | Must be |
|---|---|
| `iss` and `sub` | **Both** exactly the `client_id` |
| `aud` | The token endpoint URL, **byte-exact** |
| `jti` | **Unique per request** — eSignet runs `unique.jti.required=true` |

`redirect_uri` must be **byte-identical** between `/authorize` and `/token`, and registered on the client.

#### Invariants to preserve when editing

The code was abuse-tested against these — 16 checks including forged state, state replay, missing API key and CRLF injection. **Do not weaken them:**

- API key compared with **`timingSafeEqual`**, never `===` (`===` leaks the key a byte at a time through timing).
- The IS redirect is built from **config plus a tenant validated against `^[A-Za-z0-9._-]{1,120}$`**, falling back to `carbon.super`. Never interpolate user input into it.
- Every externally supplied string passes through **`clean()`** before logging (strips CR/LF/tab) — blocks log forging.
- **`redact()`** strips `access_token`, `id_token`, `refresh_token`, `client_assertion` and `code` from anything logged or returned.
- **`TtlMap`** bounds both stores (10-minute TTL, 5000 entries); JSON body limit 64 kB.
- `state` is **256 bits** from `randomBytes`; PKCE **`S256`** is always sent.
- Response headers: `nosniff`, `X-Frame-Options: DENY`, `Cache-Control: no-store`, `x-powered-by` disabled.

---

## Configuration reference

All bridge configuration is environment variables with localhost defaults (`cfg` at the top of `server.js`):

| Variable | Default | Notes |
|---|---|---|
| `PORT` | `4000` | |
| `BRIDGE_URL` | `http://localhost:4000` | Callback is `${BRIDGE_URL}/callback` — must match the registered `redirectUris` |
| `IS_BASE_URL` | `https://localhost:9443` | |
| `ESIGNET_UI` | `http://localhost:3000` | |
| `ESIGNET_API` | `http://localhost:8088/v1/esignet` | |
| `CLIENT_ID` | `wso2-is-bridge` | Must match the client registered with eSignet exactly |
| `ACR` | `mosip:idp:acr:generated-code` | OTP flow; `mosip:idp:acr:password` for password |
| `SCOPE` | `openid profile` | |
| `BRIDGE_API_KEY` | *(empty)* | **Empty disables the API-key check.** Always set it. |
| `FLOW_TTL_MS` | `600000` (10 min) | |

> ⚠️ **Run the bridge from its own directory.** `private.jwk.json` is read from the **process CWD**, not from a path relative to `server.js`.

There is **no build, no lint and no test script** — `npm test` exits 1. Verification is `preflight.sh` plus a manual end-to-end login.

WSO2 IS config lives in `demo-setup/wso2is-7.3.0/repository/conf/deployment.toml`. The session idle timeout defaults to **15 minutes**, short enough to expire between rehearsal and the real demo:

```toml
[session.timeout]
idle_session_timeout = "60m"
remember_me_session_timeout = "14d"
```

Restart IS for this to take effect.

---

## Troubleshooting

Full tables in runbook §17. The ones you'll hit most:

| Symptom | Cause | Fix |
|---|---|---|
| `invalid_assertion` | `aud` isn't the exact token endpoint URL | Must be `http://localhost:8088/v1/esignet/oauth/v2/token`, character for character |
| `invalid_assertion` | Reused `jti` | Generate a fresh UUID per request |
| `invalid_client` | Registered public key ≠ your private key | Re-run `genKeys.js` **and re-register the client** |
| `invalid_redirect_uri` | `redirect_uri` differs between `/authorize` and `/token` | Must be byte-identical and registered |
| `pkce_failed` | Verifier missing or mismatched | The flow record expired — check `FLOW_TTL_MS` |
| IS shows a generic auth error immediately | Bridge rejected the call | API key must match in both the Console and `BRIDGE_API_KEY` |
| Browser stalls after eSignet | Wrong `commonauth` path | Root tenant is `/t/carbon.super/commonauth`; sub-orgs use `/o/{orgId}/commonauth` |
| Second app prompts for login again | SSO session expired | Raise `idle_session_timeout`, restart IS |
| `Please Enter Valid Individual ID` | ID absent or has whitespace | eSignet's validator is `\S*` — **no whitespace anywhere**. Pasting from a PDF often adds a trailing space |
| CAPTCHA appears | Not running the `local` Spring profile | Container needs `active_profile_env=default,local` |

```bash
docker compose ps                 # all containers healthy?
docker compose logs -f esignet    # eSignet service logs
curl -s localhost:4000/health     # bridge reachable?
curl -s -o /dev/null -w '%{http_code}\n' -X POST localhost:4000/authenticate   # expect 401
```

To reset only the WSO2 side, delete the JIT-provisioned user in Console → **Users**. The next login recreates it.

---

## Known limitations

**State these openly during the demo.** Don't let someone else discover them mid-presentation.

**1 — The bridge is a translating middlebox.** It terminates eSignet's tokens and then asserts the user's identity to WSO2 IS on its own authority. IS never sees eSignet's tokens. Correct for a demo, **wrong for production**.

> **The production-correct answer** is a custom federated authenticator JAR extending `OpenIDConnectAuthenticator` and overriding `getAccessTokenRequest()` and `getSubjectAttributes()` — both `protected`, so this is the intended extension seam and WSO2 documents the pattern. Budget 2–4 days for the OSGi build. Given the WSO2 / IIIT-B eSignet collaboration, that connector is worth contributing upstream.

**2 — Logout is one-sided.** Signing out of IS destroys the `commonauthId` session, so both apps re-authenticate. It does **not** propagate to eSignet, because **eSignet exposes no logout endpoint at all** — its OpenAPI spec has no `end_session`, no front-channel and no back-channel logout, only `/oauth/introspect`. This is a current gap in eSignet, not something omitted from this build.

**3 — `sub` is a pairwise pseudonymous identifier.** eSignet declares `subject_types_supported: pairwise`; `sub` is a Partner-Specific User Token, different for every relying party. **Never present it as a national ID number.** It cannot correlate the citizen across services — a privacy feature, not a defect.

**4 — eSignet's "SSO" is not a session.** It means "one credential across many services," not a browser session. There is no session cookie and no cross-RP session state. **WSO2 IS provides the actual SSO session.** That division of labour is the point of the demo.

**5 — What was and wasn't tested.**
- *Tested:* the bridge's contract handling and cryptography, end to end, against a simulated eSignet that verifies the client assertion, enforces PKCE and returns UserInfo as `application/jwt`. Sixteen checks passed, including forged state, state replay, missing API key and CRLF injection.
- *Not tested:* the exact error strings the real eSignet container returns, and the CSRF cookie extraction in Step 4.1.

---

## Security notes

> 🔴 **`private.jwk.json`, `public.jwk.json` and `cookies.txt` are present in `demo-setup/esignet-bridge/`, and this tree has no `.gitignore`.** Nothing is exposed today — the tree is not a git repository. **If it is ever initialised as one, add a `.gitignore` first** (runbook §6.3):
>
> ```bash
> echo -e "private.jwk.json\npublic.jwk.json\ncookies.txt\nnode_modules/\n.env" > .gitignore
> ```

- The private JWK **must never leave the bridge**.
- When registering the authenticator in the IS Console, **do not select "No Authentication."** It works, but it leaves an unauthenticated identity-assertion endpoint open on your machine. Use **API Key** with header `x-api-key`.
- `BRIDGE_API_KEY` empty means the key check is skipped entirely. The bridge prints `api key : DISABLED (dev only)` at startup — **if you see that before a demo, stop and fix it.**

---

## References

Runbook **Appendix A** maps every technical claim above to its source file — eSignet v1.8.0, `identity-outbound-auth-oidc`, the official WSO2 custom-authenticator sample, and the MOSIP community announcements. If you need to defend a statement in this README, that table is where the evidence is.
