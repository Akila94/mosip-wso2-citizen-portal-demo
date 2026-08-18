# Getting started — from a fresh clone to a running demo

One path, start to finish, for someone who has never touched this repo. It builds the
**production-correct** federation (`setup-without-bridge/` — a custom authenticator JAR, not the
Node.js bridge) and then the full citizen portal on top of it (`citizen-portal-integration/`).

Budget **60–90 minutes** the first time (mostly waiting on downloads and one Console
walk-through) and **~20 GB of free disk**. Every later `stop` / `start` cycle takes seconds.

If any step's exact rationale is missing here, the deep-dive docs are one link away — this page
just sequences them. Two rules that make the rest of this page make sense:

- **Two scripts, two jobs.** `setup-without-bridge/demo.sh` owns WSO2 IS and eSignet. This
  repo's own `citizen-portal-integration/portal-demo.sh` owns the two Go services and the SPA,
  and **never** starts IS or eSignet itself.
- **Two kinds of work.** Scripted steps are a command. Console steps are clicking around
  `https://localhost:9443/console` by hand — no script can do them, because they're the thing
  `demo.sh setup` genuinely cannot automate (WSO2 IS has no declarative-config import for this).

---

## 0. Prerequisites

| Tool | Why | Check |
|---|---|---|
| Docker Desktop / Rancher Desktop, **Compose v2** | runs the 5-container eSignet stack | `docker compose version` |
| A JDK — **11, 17, or 21** (not 22+) | builds and runs WSO2 IS 7.3.0 and the authenticator JAR | `java -version` |
| Maven 3.6+ | builds the authenticator JAR | `mvn -v` |
| **Go 1.25+** | builds `citizen-portal-bff` and `gov-services-api` | `go version` |
| **Node 22** | builds the React SPA (Vite 5 needs ≥18; this repo standardizes on 22) | `node -v` |
| `git`, `curl`, `unzip`, `python3`, `keytool` (ships with the JDK) | used throughout by both `demo.sh` scripts | — |

If your default `node` is older (Vite 5 fails on Node 16 with an opaque
`crypto$2.getRandomValues is not a function` — a version problem wearing a confusing error),
switch before building the SPA:

```bash
nvm install 22 && nvm use 22
```

**Only one federation variant can be running at a time** — `esignet-bridge/` and
`setup-without-bridge/` share ports and the same eSignet Docker Compose project. This guide uses
`setup-without-bridge/` only; don't run `esignet-bridge/demo.sh` alongside it.

---

## 1. Bring up WSO2 IS + eSignet (the base federation)

Everything in this step lives in **`setup-without-bridge/`**.

```bash
cd setup-without-bridge
./demo.sh setup
```

This clones eSignet (`v1.8.0`), starts its 5 containers, downloads and unpacks WSO2 IS 7.3.0,
builds the authenticator JAR and drops it into `wso2is-7.3.0/repository/components/dropins/`,
creates the test citizen (individual ID `8267411571`, OTP `111111`), and registers the eSignet
OIDC client. It's idempotent — safe to re-run if it's interrupted.

```bash
./demo.sh start
```

Starts the eSignet stack and WSO2 IS (`https://localhost:9443`, `admin` / `admin` — accept the
self-signed cert warning once). Wait for it to report both up.

### 1a. Console work — the MOSIP eSignet connection (required)

No script can do this part. Open `setup-without-bridge/MANUAL-STEPS.md` and do **§1 and §2 only**
right now:

- **§1** — **Connections → New Connection → Custom Authenticator (Plugin Based)**, name it
  `MOSIP eSignet`, then **Settings → New Authenticator → MOSIP eSignet** and fill in the fields
  the page lists (client id, the four eSignet URLs, scopes, PKCE on).
- **§2** — map the seven eSignet claims to local claims, and turn on **Just-in-Time (JIT) User
  Provisioning** (**Provision silently**, **Override All**) — without this, login succeeds but no
  user ever appears under **Users**.

Skip §3–§5 in that file for now — those build a second, standalone demo application directly in
`setup-without-bridge/` that this guide doesn't need. (Come back to them later only if you want to
smoke-test the base federation in isolation before adding the portal on top.)

```bash
./demo.sh preflight
```

Must end `passed=N failed=0` before moving on. If the eSignet connection above isn't showing up
correctly, this is where it'll tell you.

---

## 2. Export IS's certificate (needed by the BFF)

The two Go services in the next step talk to IS over TLS and pin its self-signed certificate
rather than skip verification. Export it once:

```bash
cd ..   # back to the repo root
mkdir -p citizen-portal-integration/certs
keytool -exportcert -alias wso2carbon \
  -keystore setup-without-bridge/wso2is-7.3.0/repository/resources/security/wso2carbon.p12 \
  -storetype PKCS12 -storepass wso2carbon -rfc \
  -file citizen-portal-integration/certs/wso2is-local.pem
```

(`citizen-portal-integration/certs/README.md` has the same command with the reasoning, if the
keystore password or alias ever look unfamiliar — they're WSO2's shipped defaults, not secrets.)

---

## 3. Register the portal's three applications in the Console

Everything from here lives in **`citizen-portal-integration/`**. This step is Console work too —
`citizen-portal-integration/MANUAL-STEPS.md` §1–§6 has the full field-by-field walkthrough and the
*why* behind each non-default setting; this is the condensed version.

Create **three** applications — **Applications → New Application → Traditional Web Application**
(not Single-Page Application: these are confidential clients, because the BFF holds a secret).
Each one needs the **same** four settings changed from the template default, plus its own name
and URLs:

| Setting | Value, all three apps |
|---|---|
| **PKCE → Mandatory** | **On** (the template ships this off) |
| **Access Token → Token type** | **JWT** (the template default is opaque — the resource server in step 5 can't validate that) |
| **Login Flow → Step 1** | add **both** `MOSIP eSignet` and `Username & Password` as sign-in options, so there's a genuine choice |
| **Public client** | off (leave as-is — confidential) |

| App | Name | Authorized redirect URLs | Back-channel logout URL |
|---|---|---|---|
| 1 | `Citizen Portal` | `http://localhost:8090/bff/portal/callback` **and** `http://localhost:8090/` | `http://localhost:8090/bff/portal/backchannel-logout` |
| 2 | `Driving Licence Service` | `http://localhost:8090/bff/driving-licence/callback` **and** `http://localhost:8090/apps/driving-licence` | `http://localhost:8090/bff/driving-licence/backchannel-logout` |
| 3 | `Vehicle Revenue Licence` | `http://localhost:8090/bff/revenue-licence/callback` **and** `http://localhost:8090/apps/revenue-licence` | `http://localhost:8090/bff/revenue-licence/backchannel-logout` |

The second redirect URL in each row isn't a typo — IS validates `post_logout_redirect_uri`
against this same list, there's no separate field for it.

**After creating each one, note its Client ID and Client Secret** (Protocol tab, top of the
page) — you'll paste these into `.env` in the next step.

**Once, on any one of the three apps' Login Flow tab:** add one local demo user so the
Username & Password path is worth demonstrating too — **User Management → Users → Add User**:

```
Username: johndoe@gmail.com
Password: Wso2.123
```

No custom OAuth scopes are needed anywhere in this step — see `MANUAL-STEPS.md` §5 if curious why.

---

## 4. Configure and build

```bash
cd citizen-portal-integration
./portal-demo.sh setup
```

Creates `citizen-portal-bff/.env` and `gov-services-api/.env` from their `.env.example` files
(safe to re-run — it never overwrites an existing `.env`), and tells you which of the six client
credentials are still blank.

**Open both files and fill in the three Client ID / Client Secret pairs from step 3:**

`citizen-portal-bff/.env`:
```
PORTAL_CLIENT_ID=...          PORTAL_CLIENT_SECRET=...
DL_CLIENT_ID=...              DL_CLIENT_SECRET=...
VRL_CLIENT_ID=...             VRL_CLIENT_SECRET=...
```

`gov-services-api/.env` — the **same three Client IDs** (no secrets — this service only
validates tokens, it never presents credentials):
```
PORTAL_CLIENT_ID=...
DL_CLIENT_ID=...
VRL_CLIENT_ID=...
```

Everything else in both files already has a working default for a local setup on the default
ports. Now build everything:

```bash
./portal-demo.sh build
```

Compiles both Go services to `bin/` and runs `npm install && npm run build` for the SPA. If this
fails on the SPA step with a cryptic Vite/crypto error, it means Node isn't ≥18 in this shell —
`nvm use 22` and re-run.

---

## 5. Start it and check it's healthy

```bash
./portal-demo.sh start
```

Starts `gov-services-api` then `citizen-portal-bff` — but first checks that WSO2 IS and eSignet
(step 1) are actually up, and tells you to go run `setup-without-bridge/demo.sh start` if not.

```bash
./portal-demo.sh preflight
```

Runs the curl-level checks (SPA serving, deep-link fallback, the public catalogue, an
unauthenticated route correctly rejected, a privilege-escalation check on the public endpoint).
**Must end `failed=0`** before you open a browser.

---

## 6. Try the demo

Open **`http://localhost:8090`** — never `http://localhost:5173`. The BFF is the browser's only
origin; loading the app straight from Vite has no session and every data call fails.

1. The landing page shows the service catalogue signed out. Click **Sign in** (top right).
2. WSO2 IS's own login page offers **both** options — choose **MOSIP eSignet**.
3. On the eSignet screen (`:3000`): individual ID `8267411571` → **Get OTP** → `111111` → approve.
   (Or go back and try **Username & Password** with `johndoe@gmail.com` / `Wso2.123` instead.)
4. Back at the portal, the header shows the real name from the eSignet claims, and the catalogue
   cards flip to READY / STEP-UP.
5. Open **Driving Licence Service** → **Start application** — a full-page hop through
   `:9443/oauth2/authorize` and straight back, **with no login prompt**. That silent round trip is
   the SSO proof.
6. Open **Renew vehicle revenue licence** the same way — also silent. Its identity panel is
   narrower (no NIC, no address) than the driving-licence one, because that app's token carries
   fewer released claims — not because a fixture says so.
7. Open the **session inspector** (footer link) on both micro apps and compare: same `sub`,
   `sid`, `idp`, `amr` — different `clientId` and released claims.
8. **Sign out** from any app. WSO2 IS's back-channel logout fires and ends all three sessions in
   one shot; re-entering any app re-prompts.

`PORTAL-INTEGRATION-PLAN.md`'s "Manual, end to end" section has the full ten-step version of this
walk-through, including the devtools checks that prove no token ever reaches the browser.

---

## Day-to-day commands, once this is all working

```bash
cd citizen-portal-integration
./portal-demo.sh status      # what's up, what's down
./portal-demo.sh stop        # stops the BFF + gov-services-api only — IS/eSignet untouched
./portal-demo.sh restart
./portal-demo.sh logs bff    # or: logs govapi
./portal-demo.sh clean       # removes build output + runtime state, keeps .env
```

```bash
cd setup-without-bridge
./demo.sh stop               # stops IS + eSignet, keeps all data (never `down -v`)
./demo.sh status
```

---

## If something goes wrong

| Symptom | Likely cause | Fix |
|---|---|---|
| SPA build fails with a Vite/`crypto` error | Node < 18 active | `nvm use 22`, re-run `./portal-demo.sh build` |
| `./portal-demo.sh start` refuses immediately | IS or eSignet not up | `cd ../setup-without-bridge && ./demo.sh start` |
| "MOSIP eSignet" doesn't appear as a sign-in option to add | the authenticator JAR didn't load into IS | `setup-without-bridge/MANUAL-STEPS.md`'s *"MOSIP eSignet" is not in the authenticator list* section |
| Login fails, IS logs `ESIGNET-650xx` | eSignet-side federation problem, not the portal | `setup-without-bridge/README.md`'s error-code table |
| Login succeeds but nothing appears under Console → Users | JIT provisioning not enabled (step 1a, §2) | turn it on, retry |
| `403` on a data call | wrong client registered as JWT-vs-opaque, or a stale `.env` value | re-check step 3's four required settings and step 4's six credentials |
| A citizen's session dies mid-demo with an upstream error | access token expired (no refresh-token support, by design) | sign in again |
| Everything was working, then eSignet won't start after a container recreate (`KER-KMA-004`) | eSignet's DB and its keys have different lifetimes — see `CLAUDE.md`'s "Known state and gaps" | `cd setup-without-bridge/esignet/docker-compose && docker compose down -v`, then `./demo.sh setup` again |

---

## Map of the docs, if you need more depth than this page

| Question | Read |
|---|---|
| Why does any of this exist — the architecture, the decisions, the six milestones | `PORTAL-INTEGRATION-PLAN.md` |
| Exact Console fields and the *why* behind each one, for the base federation | `setup-without-bridge/MANUAL-STEPS.md` |
| Exact Console fields and the *why* behind each one, for the three portal apps | `citizen-portal-integration/MANUAL-STEPS.md` |
| What `portal-demo.sh` does and doesn't do | `citizen-portal-integration/README.md` |
| What was actually verified live vs. still a gap | `citizen-portal-integration/M6-SESSION-NOTES.md` |
| Repo-wide facts, invariants, and known gaps | `CLAUDE.md` (if present locally — it's gitignored) |
