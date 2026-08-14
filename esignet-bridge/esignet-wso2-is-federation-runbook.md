# MOSIP eSignet + WSO2 Identity Server — Federation & SSO Demo Runbook

**Goal:** Run MOSIP eSignet locally, federate WSO2 Identity Server 7.3.0 to it, and demonstrate identity federation plus single sign-on across two applications.

**Time:** ~90 minutes first run, ~10 minutes to restart afterwards.

**Everything below was verified against source.** Version numbers, config keys, API payloads and the bridge code were checked against the eSignet v1.8.0 repository, the WSO2 `identity-outbound-auth-oidc` repository, and the official WSO2 custom-authenticator sample. The bridge code was executed end-to-end against a simulated eSignet; 16 functional and abuse-case checks passed.

---

## Table of contents

1. [Read this first — the blocker](#1-read-this-first--the-blocker)
2. [Architecture](#2-architecture)
3. [Prerequisites](#3-prerequisites)
4. [Step 1 — Run eSignet locally](#4-step-1--run-esignet-locally)
5. [Step 2 — Create a test citizen](#5-step-2--create-a-test-citizen)
6. [Step 3 — Generate the client keypair](#6-step-3--generate-the-client-keypair)
7. [Step 4 — Register the OIDC client with eSignet](#7-step-4--register-the-oidc-client-with-esignet)
8. [Step 5 — Install WSO2 Identity Server 7.3.0](#8-step-5--install-wso2-identity-server-730)
9. [Step 6 — Build and run the bridge](#9-step-6--build-and-run-the-bridge)
10. [Step 7 — Register the custom authenticator in WSO2 IS](#10-step-7--register-the-custom-authenticator-in-wso2-is)
11. [Step 8 — Create application A and wire the login flow](#11-step-8--create-application-a-and-wire-the-login-flow)
12. [Step 9 — First end-to-end login](#12-step-9--first-end-to-end-login)
13. [Step 10 — Create application B for the SSO proof](#13-step-10--create-application-b-for-the-sso-proof)
14. [Step 11 — Configure JIT provisioning and session timeout](#14-step-11--configure-jit-provisioning-and-session-timeout)
15. [Preflight check](#15-preflight-check)
16. [Demo script](#16-demo-script)
17. [Troubleshooting](#17-troubleshooting)
18. [Known limitations — state these openly](#18-known-limitations--state-these-openly)
19. [Restarting the environment](#19-restarting-the-environment)
20. [Appendix A — Verified facts and where they came from](#20-appendix-a--verified-facts-and-where-they-came-from)

---

## 1. Read this first — the blocker

**The stock WSO2 "Standard-Based IdP → OpenID Connect" connection cannot talk to eSignet. Do not attempt it.** Two hard incompatibilities, both confirmed by reading the source:

| # | Incompatibility | Evidence |
|---|---|---|
| 1 | eSignet requires `private_key_jwt` at the token endpoint. WSO2's outbound OIDC connector only implements `client_secret_basic` and `client_secret_post`. | `TokenRequest.java` declares `@NotBlank private String client_assertion` and has no `client_secret` field at all. `OpenIDConnectAuthenticator.getAccessTokenRequest()` has exactly two branches — HTTP Basic and credentials-in-body. The string `client_assertion` does not appear anywhere in the `identity-outbound-auth-oidc` repository. |
| 2 | eSignet returns UserInfo as a signed JWT. WSO2's connector parses UserInfo as plain JSON. | `UserInfoResponseHelper.processUserInfoResponse()` always returns JWS or JWE — even plain JSON from a plugin is signed into a JWS first. WSO2's `getSubjectAttributes()` calls `JSONUtils.parseJSON()`, which throws on a JWT. |

**This is a limitation of the *outbound* connector only.** WSO2 IS fully supports `private_key_jwt` **inbound** — for OAuth clients authenticating *to* IS, where it is a selectable token-endpoint auth method and the default for FAPI apps. The two directions are separate implementations in separate repositories, so inbound support says nothing about outbound. If this comes up during the demo, that distinction is the honest answer; see §20 for the source citations on both sides.

Additional constraint: **eSignet does not put user claims in the ID token**, so you cannot work around #2 by omitting the UserInfo endpoint URL.

> **Do not** try to fix this by editing `mosip.esignet.supported.client.auth.methods`. That property only controls what the discovery document advertises. `client_assertion` is a mandatory field on the request DTO — there is no code path that accepts a client secret.

### The solution

WSO2 IS 7.1 and later ship a **service-based custom authenticator**: you register an HTTP endpoint in the Console and IS calls it with a JSON contract. No Java, no OSGi bundle, no JAR, no server restart. The contract supports a `redirect` operation, which is exactly what OIDC federation needs.

This runbook builds a small Node.js service ("the bridge") that implements that contract on one side and speaks proper OIDC to eSignet on the other.

---

## 2. Architecture

```
Browser
  │
  │ 1. open App A
  ▼
WSO2 Identity Server 7.3.0  ── https://localhost:9443
  │
  │ 2. POST /authenticate  {flowId, event, allowedOperations:[{op:"redirect"}]}
  ▼
Bridge (Node.js)  ── http://localhost:4000
  │
  │ 3. reply {actionStatus:"INCOMPLETE", operations:[{op:"redirect", url:<eSignet /authorize>}]}
  │    IS redirects the browser to eSignet
  ▼
eSignet UI  ── http://localhost:3000
  │  user enters national ID, receives OTP, consents
  ▼
eSignet service  ── http://localhost:8088
  │  4. redirects browser to  http://localhost:4000/callback?code=..&state=..
  ▼
Bridge
  │  5. POST /oauth/v2/token   (private_key_jwt client assertion + PKCE)
  │  6. GET  /oidc/userinfo    (returns a signed JWT; bridge verifies against JWKS)
  │  7. redirect browser to  https://localhost:9443/t/carbon.super/commonauth?flowId=..
  ▼
WSO2 Identity Server
  │  8. POST /authenticate again with the same flowId
  │     bridge replies {actionStatus:"SUCCESS", data:{user:{id, claims:[...]}}}
  │  9. JIT-provisions the user, creates the SSO session (commonauthId cookie)
  ▼
App A logged in.   App B then logs in from the IS session — eSignet is never called again.
```

### Port map

| Component | Port | URL |
|---|---|---|
| eSignet UI | 3000 | `http://localhost:3000` |
| eSignet service | 8088 | `http://localhost:8088/v1/esignet` |
| Mock identity system | 8082 | `http://localhost:8082/v1/mock-identity-system` |
| PostgreSQL | 5455 | (internal) |
| Redis | 6379 | (internal) |
| Bridge | 4000 | `http://localhost:4000` |
| WSO2 IS | 9443 | `https://localhost:9443` |

---

## 3. Prerequisites

Install and verify each before starting:

```bash
docker --version          # Docker Engine
docker compose version    # Compose v2
node --version            # v22 (use Node 22 — the version the bridge is run on)
java -version             # JDK 11–21 (21 recommended)
curl --version
python3 --version         # used only for JSON pretty-printing in checks
```

**Requirements:**

- Everything runs on **one machine**. Both the browser and the servers must resolve `localhost` to the same host. Do not split components across machines for this demo.
- Ports 3000, 4000, 5455, 6379, 8082, 8088 and 9443 must be free.
- Roughly 6 GB of free RAM and 10 GB of disk.
- Outbound internet access for the first run (Docker image pulls, npm install, IS download).

### Where everything lives

Clone the `mosip-wso2-citizen-portal-demo` repository wherever you like. **The `esignet-bridge/` directory inside that clone is the workspace root for this runbook**, and every file it creates goes inside it — nothing is written to your home directory or anywhere else:

```
<clone>/
├── esignet-bridge/                 # the workspace root for this runbook
│   ├── server.js  genKeys.js  package.json   # in the repo
│   ├── demo.sh                               # automates Steps 1-6, 15
│   ├── private.jwk.json   public.jwk.json    # Step 3 writes these here
│   ├── esignet/                              # Step 1 clones this here
│   └── wso2is-7.3.0/                         # Step 5 unpacks this here
└── setup-without-bridge/           # the same demo without the bridge (a JAR instead)
```

Every path below is written relative to that root as `$REPO`. **In each new terminal, point `$REPO` at the bridge directory first:**

```bash
export REPO=~/mosip-wso2-citizen-portal-demo/esignet-bridge   # adjust to your clone
cd "$REPO"
```

The `esignet/`, `wso2is-7.3.0/` and generated-key paths are already in the repo's `.gitignore`, so keeping them here does not make them committable.

---

## 4. Step 1 — Run eSignet locally

### 4.1 Clone and start

```bash
cd "$REPO"
git clone --depth 1 --branch v1.8.0 https://github.com/mosip/esignet.git
cd "$REPO"/esignet/docker-compose
docker compose --file docker-compose.yml up
```

Leave this terminal running. It starts five containers:

| Image | Host port |
|---|---|
| `postgres:bookworm` | 5455 |
| `redis:6.0` | 6379 |
| `mosipid/mock-identity-system:0.13.0` | 8082 |
| `mosipid/esignet-with-plugins:1.8.0` | 8088 |
| `mosipid/oidc-ui:1.8.0` | 3000 |

First run pulls several GB of images. Wait until the eSignet container logs `Started EsignetServiceApplication`.

### 4.2 Verify before continuing

```bash
curl -s http://localhost:8088/v1/esignet/oidc/.well-known/openid-configuration | python3 -m json.tool
```

**Confirm all four of these in the output:**

- `"token_endpoint_auth_methods_supported": ["private_key_jwt"]`
- `"grant_types_supported"` contains `authorization_code`
- `"response_types_supported": ["code"]`
- `"acr_values_supported"` contains `mosip:idp:acr:generated-code`

Then check the UI loads:

```bash
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:3000/    # expect 200
```

### 4.3 Why local beats the MOSIP Collab sandbox right now

Two things make the local stack far more reliable for a demo, both verified in source:

- **Static OTP `111111` still works locally.** `AuthenticationServiceImpl.java` in the mock identity system contains `private static final String OTP_VALUE = "111111"`. MOSIP removed static OTP from the Collab sandbox on 19 June 2026; the published docs still say `111111` and have not been updated.
- **No CAPTCHA and no admin auth locally.** `application-local.properties` sets `mosip.esignet.captcha.required=` (empty) and leaves the client-management endpoints unsecured.

Collab's self-registration → IDA pipeline has also failed twice in five months (March–April 2026 and again 6–7 August 2026, with the August thread still showing an unresolved reply). Do not build a demo on it.

---

## 5. Step 2 — Create a test citizen

```bash
curl -s -X POST http://localhost:8082/v1/mock-identity-system/identity \
  -H 'Content-Type: application/json' -d "{
  \"requestTime\": \"$(date -u +%Y-%m-%dT%H:%M:%S.000Z)\",
  \"request\": {
    \"individualId\": \"8267411571\",
    \"pin\": \"545411\",
    \"password\": \"Mosip@123\",
    \"email\": \"demo@example.com\",
    \"phone\": \"+919427357934\",
    \"fullName\":   [{\"language\":\"eng\",\"value\":\"Siddharth K Mansour\"}],
    \"givenName\":  [{\"language\":\"eng\",\"value\":\"Siddharth\"}],
    \"familyName\": [{\"language\":\"eng\",\"value\":\"Mansour\"}],
    \"gender\":     [{\"language\":\"eng\",\"value\":\"Male\"}],
    \"dateOfBirth\": \"1987/11/25\",
    \"streetAddress\": [{\"language\":\"eng\",\"value\":\"Slung\"}],
    \"locality\":   [{\"language\":\"eng\",\"value\":\"Bengaluru\"}],
    \"region\":     [{\"language\":\"eng\",\"value\":\"Karnataka\"}],
    \"country\":    [{\"language\":\"eng\",\"value\":\"India\"}],
    \"postalCode\": \"45009\",
    \"encodedPhoto\": \"data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/wAALCAABAAEBAREA/8QAFAABAAAAAAAAAAAAAAAAAAAACf/EABQQAQAAAAAAAAAAAAAAAAAAAAD/2gAIAQEAAD8AKp//2Q==\",
    \"preferredLang\": \"eng\",
    \"locale\": \"en\",
    \"zoneInfo\": \"test zone\"
  }}" | python3 -m json.tool
```

**Write down the login credentials:**

- Individual ID: `8267411571`
- OTP: `111111`
- Password (if you use the password ACR instead): `Mosip@123`

Expect an HTTP 200 with a success response. If you get a duplicate error, the user already exists — that is fine, continue.

---

## 6. Step 3 — Generate the client keypair

eSignet only supports **RSA** keys for client registration, and only `private_key_jwt` for client authentication. You must generate a keypair and register the public half.

### 6.1 Create the project

```bash
mkdir -p "$REPO"/esignet-bridge && cd "$REPO"/esignet-bridge
npm init -y                # skip: the repo already ships package.json
npm pkg set type=module     # skip: already set
npm install jose express    # always run — node_modules/ is gitignored
```

### 6.2 The key generator

`genKeys.js` in this repository does the whole job — read it there rather than retyping it. It generates a 2048-bit RS256 keypair, tags both halves with a shared random `kid` plus `alg: RS256` / `use: sig`, writes `private.jwk.json` and `public.jwk.json` **into the current working directory**, and prints the public JWK ready to paste into Step 4.

That cwd behaviour is why Step 3.3 runs it from `$REPO/esignet-bridge`: the bridge later reads `private.jwk.json` the same way, so the pair must land beside `server.js`.

### 6.3 Run it

```bash
node genKeys.js
```

**Result:** two files. `private.jwk.json` stays on the bridge and is never shared. `public.jwk.json` goes to eSignet in the next step.

> Never commit `private.jwk.json`. The repository's `.gitignore` at `$REPO/.gitignore` already covers both key files, `node_modules/` and `.env` — confirm rather than overwrite it:
> ```bash
> cd "$REPO" && git check-ignore -v private.jwk.json
> ```
> If that prints nothing the file is *not* ignored; append the rule instead of redirecting over the file:
> ```bash
> printf '%s\n' private.jwk.json public.jwk.json node_modules/ .env >> "$REPO"/.gitignore
> ```

---

## 7. Step 4 — Register the OIDC client with eSignet

eSignet uses Spring's cookie-based CSRF protection, so you must fetch a CSRF token first and send it back in both the cookie and the header.

### 7.1 Register via curl

not working
```bash
cd "$REPO"/esignet-bridge
curl -s -c cookies.txt -o /dev/null http://localhost:8088/v1/esignet/csrf/token
CSRF=$(awk '$6=="XSRF-TOKEN"{print $7}' cookies.txt)
PUBKEY=$(cat public.jwk.json)

curl -s -b cookies.txt -X POST http://localhost:8088/v1/esignet/client-mgmt/client \
  -H 'Content-Type: application/json' -H "X-XSRF-TOKEN: $CSRF" -d "{
  \"requestTime\": \"$(date -u +%Y-%m-%dT%H:%M:%S.000Z)\",
  \"request\": {
    \"clientId\": \"wso2-is-bridge\",
    \"clientName\": \"WSO2 IS Bridge\",
    \"relyingPartyId\": \"mock-relying-party-id\",
    \"publicKey\": $PUBKEY,
    \"logoUri\": \"https://wso2.com/favicon.ico\",
    \"userClaims\": [\"name\",\"email\",\"gender\",\"phone_number\",\"birthdate\",\"picture\"],
    \"authContextRefs\": [\"mosip:idp:acr:generated-code\",\"mosip:idp:acr:password\"],
    \"redirectUris\": [\"http://localhost:4000/callback\"],
    \"grantTypes\": [\"authorization_code\"],
    \"clientAuthMethods\": [\"private_key_jwt\"],
    \"additionalConfig\": { \"userinfo_response_type\": \"JWS\" }
  }}" | python3 -m json.tool
```

**Expected result:** a response containing `"clientId": "wso2-is-bridge"` and `"status": "ACTIVE"`.

### 7.2 Fallback if the curl approach fails

The CSRF cookie extraction is the fragile part and was **not** testable against a live instance during preparation. If it fails, use the Postman collection, which is the officially documented path:

1. Import `$REPO/esignet/postman-collection/eSignet.postman_collection.json`
2. Import `$REPO/esignet/postman-collection/eSignet-with-mock.postman_environment.json` and select it
3. Run **OIDC Client Mgmt → Mock → Get CSRF token**
4. Set the environment variables `client_id` = `wso2-is-bridge`, `client_public_key` = the contents of `public.jwk.json`, `redirection_url` = `http://localhost:4000/callback`
5. Run **OIDC Client Mgmt → Mock → Create OIDC client**

### 7.3 Field notes

| Field | Why this value |
|---|---|
| `clientId` | Free choice, but must match `CLIENT_ID` in the bridge exactly. |
| `relyingPartyId` | `mock-relying-party-id` is what the mock plugin expects. Do not change it. |
| `publicKey` | Must be the JWK object itself, not a string. Note there are no escaping quotes around `$PUBKEY` in the payload. |
| `redirectUris` | Must contain the bridge callback. eSignet matches this against the value sent in both `/authorize` and `/token`. |
| `clientAuthMethods` | `private_key_jwt` is the only accepted value. |
| `userinfo_response_type` | Keep `JWS`. `JWE` requires registering a separate encryption certificate and adds a decryption step with no demo benefit. |

---

## 8. Step 5 — Install WSO2 Identity Server 7.3.0

7.3.0 is the current GA release (confirmed against the release tags on `wso2/product-is`).

```bash
cd "$REPO"
curl -LO https://github.com/wso2/product-is/releases/download/v7.3.0/wso2is-7.3.0.zip
unzip wso2is-7.3.0.zip
cd "$REPO"/wso2is-7.3.0
./bin/wso2server.sh            # Windows: bin\wso2server.bat
```

Leave this terminal running. Startup takes 1–3 minutes.

**Verify:** open `https://localhost:9443/console` and log in with `admin` / `admin`. Accept the self-signed certificate warning.

---

## 9. Step 6 — Build and run the bridge

### 9.1 The bridge

`server.js` in this repository **is** the bridge — it is already written; do not recreate it. Read the file for the details; the shape is:

| Part | What it does |
|---|---|
| config block | Every setting comes from an environment variable with a localhost default: `PORT`, `BRIDGE_URL`, `IS_BASE_URL`, `ESIGNET_UI`, `ESIGNET_API`, `CLIENT_ID`, `ACR`, `SCOPE`, `BRIDGE_API_KEY`, `FLOW_TTL_MS`. There is no config file. |
| startup | Loads `private.jwk.json` from the cwd (so run it from `$REPO/esignet-bridge`) and opens a remote JWKS reference to `$ESIGNET_API/oauth/.well-known/jwks.json`. |
| `POST /authenticate` | The endpoint WSO2 IS calls, guarded by the `x-api-key` check. First call per `flowId`: mints `state`, `nonce` and a PKCE verifier, then replies `INCOMPLETE` with a `redirect` operation to eSignet `/authorize`. Second call: replies `SUCCESS` with the mapped claims, or `FAILED` with a sanitised reason. |
| `GET /callback` | Where eSignet sends the browser. Redeems the single-use `state`, exchanges the code at `/oauth/v2/token` with a `private_key_jwt` assertion plus `code_verifier`, fetches `/oidc/userinfo`, verifies that signed JWT against the JWKS, records the outcome on the flow, and redirects to `$IS_BASE_URL/t/<tenant>/commonauth?flowId=…`. |
| `toWso2User()` | Maps eSignet claims onto `http://wso2.org/claims/*` — `username` (from `sub`), `emailaddress`, `givenname`, `lastname`, `mobile`, `dob`, `gender`. |
| `GET /health` | Unauthenticated liveness check used by the preflight script. |

Install its two dependencies (`express`, `jose`) as shown in Step 3.1 before running it.

### 9.2 Choose and record an API key

```bash
openssl rand -base64 32
```

Copy the output. You will paste it in two places: the `BRIDGE_API_KEY` environment variable below, and the WSO2 IS Console in Step 7. **Save it somewhere now** — the Console will not show it again after you save.

### 9.3 Run the bridge

```bash
cd "$REPO"/esignet-bridge
BRIDGE_API_KEY='<paste-your-key-here>' node server.js
```

Leave this terminal running and visible — you will point at it during the demo.

**Expected output:**

```
bridge listening on http://localhost:4000
  eSignet UI  : http://localhost:3000
  eSignet API : http://localhost:8088/v1/esignet
  WSO2 IS     : https://localhost:9443
  client_id   : wso2-is-bridge
  api key     : enabled
```

### 9.4 Security properties built into this code

These are not optional extras — they are the reason the code looks the way it does:

| Control | Implementation |
|---|---|
| Endpoint authentication | API key compared with `timingSafeEqual`, not `===` |
| Redirect safety | The IS redirect is assembled from config plus a tenant validated against `^[A-Za-z0-9._-]{1,120}$`; anything else falls back to `carbon.super` |
| Session unpredictability | `state` is 256 bits from `crypto.randomBytes`, well above the 128-bit floor |
| Replay protection | `state` is deleted on first use; PKCE `S256` is always sent |
| Log forging | All externally supplied strings pass through `clean()`, which strips CR, LF and tab |
| Token leakage | `redact()` removes `access_token`, `id_token`, `refresh_token`, `client_assertion` and `code` before any error is logged or returned |
| Resource exhaustion | `TtlMap` expires entries after 10 minutes and caps at 5000; JSON body limit is 64 kB |
| Response headers | `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Cache-Control: no-store`, `x-powered-by` disabled |

---

## 10. Step 7 — Register the custom authenticator in WSO2 IS

1. Open `https://localhost:9443/console` and sign in as `admin` / `admin`.
2. Go to **Connections** in the left navigation.
3. Click **New Connection**.
4. Select the **Custom Authenticator** template. Click **Create**.
5. For authenticator type, select **External (Federated) User Authentication**. Click **Next**.
6. Under **General Settings**:
   - **Identifier:** `esignet`
   - **Display name:** `Sign in with eSignet`
7. Under **Configuration**:
   - **Endpoint:** `http://localhost:4000/authenticate`
   - **Authentication:** select **API Key**
     - **Header name:** `x-api-key`
     - **Value:** the key you generated in Step 6.2
8. Click **Finish**.

> **Do not select "No Authentication".** It works, but it leaves an unauthenticated identity-assertion endpoint open on your machine.

---

## 11. Step 8 — Create application A and wire the login flow

1. In the Console, go to **Applications** → **New Application**.
2. Choose the **Single-Page Application** template.
3. Name it `Citizen Portal A`.
4. Set **Authorized redirect URL** to `http://localhost:5173` (or whatever your test app uses). Click **Create**.
5. Open the application and go to the **Login Flow** tab.
6. Click **Add Sign-In Option** on **Step 1**.
7. Select **Sign in with eSignet**. Click **Add**.
8. Click **Update**.

> Keep the eSignet option in **Step 1**. WSO2 recommends putting identifying connections in the first step, because that step establishes who the user is.

---

## 12. Step 9 — First end-to-end login

### 12.1 Run the login

1. Start a login from `Citizen Portal A`. If you do not have a client app yet, use the **Try it** button on the application's Quick Start tab. To build the request by hand instead, it **must include PKCE** — the SPA template enables *Mandatory PKCE*, so a request without `code_challenge` is rejected with `PKCE is mandatory for this application`. See `MANUAL-STEPS.md` step 5 for the two commands that generate the challenge; the URL is then:

   ```
   https://localhost:9443/oauth2/authorize?response_type=code&client_id=<CLIENT_ID>&redirect_uri=http://localhost:5173&scope=openid%20profile&code_challenge=<CHALLENGE>&code_challenge_method=S256
   ```
2. On the WSO2 IS login page, click **Sign in with eSignet**.
3. You land on the eSignet UI at `localhost:3000`.
4. Enter individual ID `8267411571`.
5. Click **Get OTP** (or **Send OTP**).
6. Enter `111111`.
7. Review the consent screen and approve.
8. You are returned through the bridge to the application, logged in.

### 12.2 Confirm each hop worked

| Where to look | What you should see |
|---|---|
| Bridge terminal | `[login-ok] flow=<id> sub=<PSUT> claims=6` |
| IS Console → **Users** | A new user whose username is the eSignet `sub` value |
| That user's profile | Email, first name, last name, mobile populated |

If any of these are missing, go to [Troubleshooting](#17-troubleshooting) before continuing.

---

## 13. Step 10 — Create application B for the SSO proof

Repeat Step 8 exactly, changing only:

- **Name:** `Citizen Portal B`
- **Authorized redirect URL:** a different port, e.g. `http://localhost:5174`

Add the same **Sign in with eSignet** option to its Step 1 and click **Update**.

**This second application is the whole point of the demo.** It is what turns a login demo into an SSO demo.

---

## 14. Step 11 — Configure JIT provisioning and session timeout

### 14.1 JIT provisioning

1. **Connections** → **Sign in with eSignet** → **Just-in-Time Provisioning** tab.
2. Confirm **Just-in-Time (JIT) User Provisioning** is checked.
3. Set **JIT provisioning scheme** to **Provision silently**. This is the default and is the only scheme that does not add an extra prompt to your demo.
4. Set **Attribute synchronization method** to **Override All**, so attributes refresh from eSignet on every login.
5. Click **Update**.

### 14.2 Session timeout

The WSO2 IS SSO session idle timeout defaults to **15 minutes**. That is short enough to expire between rehearsal and the real demo. Raise it.

Edit `$REPO/wso2is-7.3.0/repository/conf/deployment.toml` (referred to below as `<IS_HOME>`) and add:

```toml
[session.timeout]
idle_session_timeout = "60m"
remember_me_session_timeout = "14d"
```

Restart WSO2 IS for this to take effect.

> **Why this matters:** WSO2 IS creates the SSO session and sets a `commonauthId` cookie in the browser holding the session identifier. That cookie is what lets application B skip eSignet entirely. eSignet itself has no session mechanism at all — see [Known limitations](#18-known-limitations--state-these-openly).

---

## 15. Preflight check

**These checks are implemented as `./demo.sh preflight`** — run that immediately before every demo and skip the rest of this section. On success it also restarts the bridge, clearing the `PREFLIGHT` entry from its in-memory map. The equivalent standalone script, if you want it as a file in `$REPO/esignet-bridge`:

```bash
#!/usr/bin/env bash
set -u
ESIGNET_API=${ESIGNET_API:-http://localhost:8088/v1/esignet}
ESIGNET_UI=${ESIGNET_UI:-http://localhost:3000}
MOCK_ID=${MOCK_ID:-http://localhost:8082/v1/mock-identity-system}
BRIDGE=${BRIDGE:-http://localhost:4000}
IS=${IS:-https://localhost:9443}
CLIENT_ID=${CLIENT_ID:-wso2-is-bridge}
API_KEY=${BRIDGE_API_KEY:-}

pass=0; fail=0
http() { code=$(curl -sk -o /dev/null -w '%{http_code}' --max-time 8 "$1")
         [ "$code" = "$2" ] || { echo "got HTTP $code, want $2"; return 1; }; }
chk()  { local l="$1"; shift
         if out=$("$@" 2>&1); then echo "  OK    $l"; pass=$((pass+1));
         else echo "  FAIL  $l"; echo "        $out" | head -2; fail=$((fail+1)); fi; }

echo "eSignet"
chk "discovery reachable"  http "$ESIGNET_API/oidc/.well-known/openid-configuration" 200
chk "JWKS reachable"       http "$ESIGNET_API/oauth/.well-known/jwks.json" 200
chk "eSignet UI serving"   http "$ESIGNET_UI/" 200
chk "mock identity up"     http "$MOCK_ID/actuator/health" 200   # /swagger-ui.html 302-redirects

disc=$(curl -s --max-time 8 "$ESIGNET_API/oidc/.well-known/openid-configuration")
echo "$disc" | grep -q private_key_jwt \
  && { echo "  OK    advertises private_key_jwt"; pass=$((pass+1)); } \
  || { echo "  FAIL  private_key_jwt not advertised"; fail=$((fail+1)); }

echo "Bridge"
chk "bridge /health" http "$BRIDGE/health" 200
resp=$(curl -s --max-time 8 -X POST "$BRIDGE/authenticate" \
  -H 'Content-Type: application/json' -H "x-api-key: $API_KEY" \
  -d '{"actionType":"AUTHENTICATION","flowId":"PREFLIGHT","requestId":"P","event":{"tenant":{"id":"-1234","name":"carbon.super"},"application":{"id":"a","name":"Preflight"},"currentStepIndex":1},"allowedOperations":[{"op":"redirect"}]}')
echo "$resp" | grep -q '"actionStatus":"INCOMPLETE"' \
  && { echo "  OK    returns INCOMPLETE + redirect"; pass=$((pass+1)); } \
  || { echo "  FAIL  bad response: $resp"; fail=$((fail+1)); }
echo "$resp" | grep -q "client_id=$CLIENT_ID" \
  && { echo "  OK    redirect carries client_id"; pass=$((pass+1)); } \
  || { echo "  FAIL  wrong or missing client_id"; fail=$((fail+1)); }

echo "Keys"
python3 -c "
import json,sys
a=json.load(open('private.jwk.json')); b=json.load(open('public.jwk.json'))
sys.exit(0 if a.get('kid')==b.get('kid') and a.get('n')==b.get('n') else 1)" \
  && { echo "  OK    keypair consistent"; pass=$((pass+1)); } \
  || { echo "  FAIL  kid or modulus mismatch"; fail=$((fail+1)); }

echo "WSO2 IS"
chk "console reachable" http "$IS/console" 200

echo; echo "passed=$pass failed=$fail"
[ "$fail" -eq 0 ] || exit 1
```

Run it:

```bash
chmod +x preflight.sh
BRIDGE_API_KEY='<your-key>' ./preflight.sh
```

**Every line must say `OK` and the exit code must be 0.** If anything fails, fix it before presenting.

> After preflight, restart the bridge so the `PREFLIGHT` entry is cleared from its in-memory map and the log is clean.

---

## 16. Demo script

**Total time: about four minutes.** Have three windows visible: the browser, the bridge terminal, and the IS Console.

### Before the audience arrives

- [ ] Run `preflight.sh` — all `OK`
- [ ] Restart the bridge for a clean log
- [ ] Open a **fresh incognito browser window** (no stale `commonauthId` cookie)
- [ ] Have both application URLs ready in separate tabs
- [ ] Optionally open `docker logs -f` on the eSignet container in a fourth pane

### The script

**1. Frame it (20 seconds)**

> "MOSIP eSignet is the national ID authentication layer. WSO2 Identity Server is the broker sitting in front of government services. These are complementary, not competing — I'll show you exactly where the boundary is."

**2. Log in to application A (90 seconds)**

- Click **Sign in with eSignet**
- Pause on the eSignet screen. Call out: the branding, the choice of authentication factors, and the fact that this is MOSIP's UI, not WSO2's.
- Enter `8267411571`, request OTP, enter `111111`.
- Pause on the **consent screen**. Call out that the citizen sees each attribute by name and chooses what to release. That is eSignet's contribution and it is a genuine differentiator for government work.

**3. Show what arrived (40 seconds)**

- Show the profile attributes in application A.
- Switch to the IS Console → **Users**. The citizen was just created, automatically, on first login. No pre-registration, no batch import.

**4. The SSO proof (60 seconds)**

- Open application B in a new tab. **You are already logged in. Instantly.**
- Now switch to the bridge terminal and point at it: **there is exactly one `[login-ok]` line.** The bridge was never called a second time.
- If you have the eSignet container log open, show it is idle too.

> "That is the federation boundary made visible. eSignet authenticated the citizen once. WSO2 Identity Server holds the session and every subsequent service federates against it."

**5. Close (30 seconds)**

> "eSignet gives you the national credential, inclusive authentication factors, and consented data sharing. It has no cross-service session and no logout endpoint. Identity Server supplies the session, single sign-on, MFA chaining and application-level authorization on top. Together that is a complete citizen access layer."

---

## 17. Troubleshooting

### Errors at the eSignet token endpoint

| Error | Cause | Fix |
|---|---|---|
| `invalid_assertion` | `aud` is not the exact token endpoint URL string | Must be `http://localhost:8088/v1/esignet/oauth/v2/token`, character for character |
| `invalid_assertion` | `iss` or `sub` is not the client ID | Both must equal `wso2-is-bridge` |
| `invalid_assertion` | Reused `jti` | eSignet sets `mosip.esignet.client-assertion.unique.jti.required=true`. Generate a fresh `randomUUID()` per request |
| `invalid_client` | Registered public key does not match your private key | Re-run `genKeys.js` and re-register the client |
| `invalid_redirect_uri` | `redirect_uri` differs between `/authorize` and `/token` | They must be byte-identical and must match a registered URI |
| `pkce_failed` | `code_verifier` missing or does not match the challenge | Confirm the flow record survived; check `FLOW_TTL_MS` has not expired |

### Errors in the WSO2 IS flow

| Symptom | Cause | Fix |
|---|---|---|
| IS shows a generic authentication error immediately | Bridge rejected the call | Check the API key matches in both the Console and `BRIDGE_API_KEY` |
| Browser stalls after eSignet, never returns to the app | Wrong `commonauth` path | Root tenant is `/t/carbon.super/commonauth`. Sub-organisations use `/o/{orgId}/commonauth` |
| Login succeeds but user has no attributes | Claims arriving but not mapped | Verify the `http://wso2.org/claims/*` URIs exist in your claim dialect. They are all present in a clean 7.3.0 |
| Second app prompts for login again | SSO session expired | Raise `idle_session_timeout` (Step 11.2) and restart IS |
| Self-signed certificate error calling the bridge | Only applies if you put the bridge behind HTTPS | Import the certificate into `<IS_HOME>/repository/resources/security/client-truststore.jks` and set `[actions.http_client] use_carbon_truststore = true` in `deployment.toml` |

### Errors at eSignet login

| Symptom | Cause | Fix |
|---|---|---|
| `Please Enter Valid Individual ID` | The ID was never created, or has whitespace | Re-run Step 2. eSignet's backend validator is `\S*` — no whitespace is permitted anywhere, and pasting from a PDF often adds a trailing space |
| OTP rejected | Wrong OTP | Locally the value is `111111`. This is hardcoded in the mock identity system |
| CAPTCHA appears | Not running the `local` Spring profile | Confirm the eSignet container has `active_profile_env=default,local` |

### General diagnostics

```bash
# Are all containers healthy?
docker compose ps

# eSignet service logs
docker compose logs -f esignet

# Is the bridge reachable and does it require the key?
curl -s localhost:4000/health
curl -s -o /dev/null -w '%{http_code}\n' -X POST localhost:4000/authenticate   # expect 401
```

Enable WSO2 IS diagnostic logs from the Console (**Logs** section) to trace the authenticator invocation in detail.

---

## 18. Known limitations — state these openly

Do not let someone else discover these mid-demo. Say them yourself.

### 18.1 The bridge is a translating middlebox

It terminates eSignet's tokens and then asserts the user's identity to WSO2 IS on its own authority. That is correct for a demo and wrong for production.

**The production-correct answer** is a custom federated authenticator JAR that extends `OpenIDConnectAuthenticator` and overrides `getAccessTokenRequest()` and `getSubjectAttributes()`. Both methods are `protected`, so this is the intended extension seam and WSO2 documents the pattern. Budget 2–4 days for the OSGi build. Given the WSO2 / IIIT-B eSignet collaboration announced in February 2026, that connector is an artifact worth contributing upstream rather than shelving.

### 18.2 Logout is one-sided

Signing out of WSO2 IS destroys the `commonauthId` session, so both applications require re-authentication. It does **not** propagate to eSignet, because **eSignet exposes no logout endpoint at all**. Its OpenAPI specification contains no `end_session`, no front-channel logout and no back-channel logout — only `/oauth/introspect`.

On a fresh browser this is invisible. If asked, the honest answer is that RP-initiated logout is a current gap in eSignet, not something omitted from this build.

### 18.3 `sub` is a pairwise pseudonymous identifier

eSignet's discovery document declares `subject_types_supported: pairwise`. The `sub` value is a Partner-Specific User Token — a different opaque value for every relying party. **Never present it as a national ID number.** It cannot be used to correlate the citizen across services, which is a privacy feature, not a defect.

### 18.4 eSignet's "SSO" is not a session

eSignet's marketing describes single sign-on, and that is accurate in the sense of "one credential across many services." It is not a browser session. There is no session cookie and no cross-relying-party session state. In this architecture, WSO2 IS provides the actual SSO session. That division of labour is the point of the demo.

### 18.5 What was and was not tested

- **Tested:** the bridge's contract handling and cryptography, end to end, against a simulated eSignet that verifies the client assertion (`iss`/`sub` equal `client_id`, correct `aud`, `jti` present), enforces PKCE, and returns UserInfo as `application/jwt`. Sixteen checks passed, including forged state, state replay, missing API key, and CRLF injection into the failure description.
- **Not tested:** the exact error strings the real eSignet container returns, and the CSRF cookie extraction in Step 4.1. Use the Postman fallback if that step misbehaves.

---

## 19. Restarting the environment

After the first setup, bringing everything back up takes about five minutes.

**Normally just run `./demo.sh start`** (or `restart` / `stop` / `status`) from the repository root — one terminal, all three services, waits included. The four-terminal form below is what it automates, and is worth knowing when you need to watch a specific service or something refuses to start.

Run `export REPO=~/mosip-wso2-citizen-portal-demo` (or wherever your clone is) in each of the four terminals first.

```bash
# Terminal 1 — eSignet
cd "$REPO"/esignet/docker-compose && docker compose up

# Terminal 2 — WSO2 Identity Server
cd "$REPO"/wso2is-7.3.0 && ./bin/wso2server.sh

# Terminal 3 — bridge
cd "$REPO"/esignet-bridge && BRIDGE_API_KEY='<your-key>' node server.js

# Terminal 4 — preflight, then leave free for logs
cd "$REPO"/esignet-bridge && BRIDGE_API_KEY='<your-key>' ./preflight.sh
```

**The test citizen persists** in the Postgres volume across restarts. If you ever run `docker compose down -v`, the volume is destroyed and you must repeat Step 2 and Step 4.

To reset only the WSO2 side, delete the JIT-provisioned user from the Console → **Users**. The next login recreates it.

---

## 20. Appendix A — Verified facts and where they came from

| Claim | Source |
|---|---|
| eSignet requires `private_key_jwt`; no `client_secret` path exists | `esignet-core/.../dto/TokenRequest.java` — `client_assertion` is `@NotBlank`; no `client_secret` field |
| Only `private_key_jwt` is configured | `esignet-service/src/main/resources/application-default.properties` — `mosip.esignet.supported.client.auth.methods={'private_key_jwt'}` |
| Client assertion signing algorithms | Same file — `{'RS256','PS256','ES256'}` |
| UserInfo is always JWS or JWE | `oidc-service-impl/.../UserInfoResponseHelper.java` — plain JSON is signed into a JWS before return |
| IS 7.3.0 ships outbound OIDC connector `5.15.2` | `product-is@support-7.3.0.x-full:pom.xml:2742` — `<identity.outbound.auth.oidc.version>5.15.2</identity.outbound.auth.oidc.version>` |
| WSO2 **outbound** OIDC has no `private_key_jwt` | `identity-outbound-auth-oidc@v5.15.2:.../OpenIDConnectAuthenticator.java:1470-1536` — `getAccessTokenRequest()` branches only on the `IsBasicAuthEnabled` property: Basic header, else client id/secret in the body. `git grep -i "client_assertion\|private_key_jwt"` over the whole repo returns zero matches at `v5.15.2` and on `master`. |
| WSO2 **inbound** OIDC *does* support `private_key_jwt` (opposite direction — not a workaround) | `identity-inbound-auth-oauth@support-7.4.99.x-full:.../oauth/common/OAuthConstants.java:298` defines the constant; `.../oauth/OAuthAdminServiceImpl.java:2913` handles it at app registration; `docs-is:.../7.3.0/docs/_data/configuration_catalog.yaml:1375` makes it a FAPI default and `:2317` documents the assertion-replay cache |
| WSO2 outbound OIDC parses UserInfo as JSON | `OIDCCommonUtil.extractUserClaimsFromJsonPayload()` calls `JSONUtils.parseJSON()` |
| eSignet has no logout endpoint | `docs/esignet-openapi.yaml` — no `end_session`, `logout`, `revoke`, `check_session` or back-channel path; only `/oauth/introspect` |
| Local static OTP is `111111` | `mock-identity-system/.../AuthenticationServiceImpl.java` — `private static final String OTP_VALUE = "111111"` |
| No CAPTCHA locally | `esignet-service/src/main/resources/application-local.properties` — `mosip.esignet.captcha.required=` (empty) |
| Static OTP removed from Collab | MOSIP community announcement, 19 June 2026 |
| WSO2 IS 7.3.0 is current GA | Release tags on `wso2/product-is` |
| Custom authenticator contract | `wso2/docs-is` — `custom-authentication-action-v1.yaml` and the official `pin-based-authentication-service-express` sample |
| Redirect-back URL format | Official WSO2 sample — `${IS}/t/${tenant}/commonauth?flowId=${flowId}`, or `/o/${orgId}/commonauth` for sub-organisations |
| SSO session cookie is `commonauthId`, default idle timeout 15 minutes | WSO2 IS session persistence and session timeout documentation |
| `sub` is pairwise | eSignet discovery document — `subject_types_supported: pairwise` |
