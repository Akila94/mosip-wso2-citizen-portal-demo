# Manual steps — citizen-portal-integration

Console work `citizen-portal-bff` cannot do for you. Console: `https://localhost:9443/console`,
`admin` / `admin`. This assumes `setup-without-bridge/./demo.sh setup && ./demo.sh start` has
already run and `./demo.sh preflight` is green — the MOSIP eSignet connection and its
`EsignetOIDCAuthenticator` bundle must already exist before you add a sign-in option that uses
it.

Work through this page in order. §1–§2 cover **M1**: the "Citizen Portal" application. §3–§4
cover **M2**: Applications A (Driving Licence Service) and B (Vehicle Revenue Licence), added in
the same style, so all three apps share one WSO2 IS SSO session.

---

## 1. Register the Citizen Portal application

1. **Applications** → **New Application** → **Traditional Web Application**.
2. **Name:** `Citizen Portal` → **Create**.
3. Open it → **Protocol** tab:
   - **Authorized redirect URLs** — add **both**:
     ```
     http://localhost:8090/bff/portal/callback
     http://localhost:8090/
     ```
     (The second one is not a mistake: IS validates `post_logout_redirect_uri` against this
     same list, not a separate field — see PORTAL-INTEGRATION-PLAN.md Component 4.)
   - **Allowed grant types** — leave as **Code** only (the template default).
   - **PKCE** — **Mandatory** must be turned **on**. The `oidc-web-application` template ships
     this *off*, unlike the Single-Page Application template — verified against the exact
     component versions IS 7.3.0 ships (see the plan's appendix). Leaving it off means a stolen
     authorization code is redeemable with no additional secret.
   - **Public client** — leave **off** (this is a confidential client; the BFF holds the
     client secret).
4. Same **Protocol** tab, **Access Token** section:
   - **Token type** → **JWT**. The template default is **Opaque**, and the resource server
     built in M3 has nothing to validate against an opaque token. Do this even though M1 has no
     resource server yet — the setting belongs to the client, not to any particular login.
5. Same **Protocol** tab, **Logout URLs** section:
   - **Back channel logout URL:**
     ```
     http://localhost:8090/bff/portal/backchannel-logout
     ```
   - Front-channel logout URL: leave blank (not used by this design).
6. Same **Protocol** tab, **ID Token** section:
   - **User attributes** — no change needed; the default `profile`/`email` claims are enough
     for M1.
7. **Note the Client ID and Client Secret** — Protocol tab, top of the page. Both are needed for
   `.env` below.

## 2. Add MOSIP eSignet as a sign-in option

1. Same application → **Login Flow** tab → Visual Editor.
2. On **Step 1**, click **+ Add Sign In Option**.
3. Choose the **MOSIP eSignet** connection (already registered per
   `setup-without-bridge/MANUAL-STEPS.md` §1 — if it is not listed, `preflight` is not green
   yet; fix that first, this page cannot substitute for it).
4. Also add **Username & Password** to the **same** Step 1, so the citizen genuinely chooses
   between the two. (The one documented exclusion: Identifier First cannot share a step with
   Username & Password — not relevant here, since Identifier First is not used.)
5. **Update**.
6. You will need at least one local user for the Username & Password path to be worth
   demonstrating: **User Management** → **Users** → **Add User**, any username/password —
   this is a throwaway demo account, not a citizen record.
      - Username: johndoe@gmail.com
      -  Password: Wso2.123

## 3. Register the Driving Licence Service application (Application A)

Same shape as §1, with its own routes and cookies. Per
`PORTAL-INTEGRATION-PLAN.md`'s Component 4 table, this is a **Traditional Web Application** too
— not the Single-Page Application template `setup-without-bridge/MANUAL-STEPS.md` §3–§4 uses
for its own standalone "Service A"/"Service B" — because this app's BFF namespace holds a client
secret, which only a confidential client can use.

1. **Applications** → **New Application** → **Traditional Web Application**.
2. **Name:** `Driving Licence Service` → **Create**.
3. Open it → **Protocol** tab:
   - **Authorized redirect URLs** — add **both**:
     ```
     http://localhost:8090/bff/driving-licence/callback
     http://localhost:8090/apps/driving-licence
     ```
     (As in §1.3, the second URL is not a mistake — it is both the SPA's own route for this
     micro app and the `post_logout_redirect_uri` IS will validate against this same list.)
   - **Allowed grant types** — leave as **Code** only.
   - **PKCE** — **Mandatory** must be turned **on** (same reasoning as §1.3).
   - **Public client** — leave **off**.
4. Same **Protocol** tab, **Access Token** section:
   - **Token type** → **JWT** (same reasoning as §1.4 — the resource server built in M3 has
     nothing to validate against an opaque token).
5. Same **Protocol** tab, **Logout URLs** section:
   - **Back channel logout URL:**
     ```
     http://localhost:8090/bff/driving-licence/backchannel-logout
     ```
6. Same application → **Login Flow** tab → Visual Editor → **Step 1** → **+ Add Sign In
   Option** → choose **MOSIP eSignet**, then also add **Username & Password** to the same Step
   1 (mirrors §2.2–§2.4) → **Update**.
7. **Note the Client ID and Client Secret** (Protocol tab) — needed for `.env` in §5.

## 4. Register the Vehicle Revenue Licence application (Application B)

Repeat §3 exactly, changing only:

- **Name:** `Vehicle Revenue Licence`
- **Authorized redirect URLs:**
  ```
  http://localhost:8090/bff/revenue-licence/callback
  http://localhost:8090/apps/revenue-licence
  ```
- **Back channel logout URL:**
  ```
  http://localhost:8090/bff/revenue-licence/backchannel-logout
  ```

Note this application's **Client ID** and **Client Secret** as well.

## 5. No custom scopes needed — audience alone gates each app's router

An earlier draft of this plan gave Application A the scope `driving_licence.write` and
Application B `vehicle_registry.read`, so `gov-services-api` (M3) could reject one app's token on
the other's router. **That approach was tried live, found to add real friction with no benefit,
and removed** — `gov-services-api`'s `/driving-licence/*` and `/vehicle-registry/*` routers now
require only that the token's audience match that app's own client ID, exactly like `/portal/*`
already did. The reasoning: a citizen only ever holds an access token whose audience is the
specific app they authenticated to, so the audience check alone already proves "this citizen has
a validly-authenticated session with this app" — there is no scenario in this project where two
different citizens should get different access to the *same* app's endpoints, so a scope on top
of the audience check was solving a problem this demo doesn't have. `M3-SESSION-NOTES.md` has
the full history, including what the scope-based approach actually required (a WSO2 IS Console
"Authorization Policy" setting that defaults to a Role-Based Access Control policy no citizen
would ever satisfy) and why it was abandoned rather than worked around further.

**Nothing to do here** — no API Resource, no Authorize-API binding, no Console work at all for
this. `citizen-portal-bff/.env.example`'s `DL_CLIENT_SCOPES`/`VRL_CLIENT_SCOPES` defaults no
longer include the removed custom scopes.

## 6. Fill in `.env`

Copy `citizen-portal-bff/.env.example` to `citizen-portal-bff/.env` and set all three apps'
credentials:

```
PORTAL_CLIENT_ID=<Client ID from step 1.7>
PORTAL_CLIENT_SECRET=<Client Secret from step 1.7>
DL_CLIENT_ID=<Client ID from step 3.7>
DL_CLIENT_SECRET=<Client Secret from step 3.7>
VRL_CLIENT_ID=<Client ID from step 4>
VRL_CLIENT_SECRET=<Client Secret from step 4>
```

Everything else in `.env.example` has a working default for a local `setup-without-bridge`
environment on the default ports.

## 7. Run it

```bash
cd citizen-portal-integration/citizen-portal-bff
go run ./cmd/bff
```

**M1 check** — open `http://localhost:8090/bff/portal/login?returnTo=/` in a browser. You
should land on IS's login page with both **Username & Password** and **MOSIP eSignet** offered.
Choosing MOSIP eSignet redirects to `localhost:3000`; individual ID `8267411571`, **Get OTP**,
`111111`, approve consent; you land back on `http://localhost:8090/` and
`GET /bff/portal/session` returns the real ID-token claims as JSON — no SPA involved yet, that
is M4/M5.

**M2 checks — SSO and single logout, still with no SPA:**

1. With the portal session from the check above still live, open
   `http://localhost:8090/bff/driving-licence/login?returnTo=/apps/driving-licence` in the
   **same browser**. It should redirect through IS and land back on
   `http://localhost:8090/apps/driving-licence` with **no eSignet screen and no credential
   prompt** — IS answers instantly from the existing `commonauthId` SSO session. Confirm with
   `GET /bff/driving-licence/session`: same `sub`/`sid` as the portal's, different `clientId`.
2. Repeat for `http://localhost:8090/bff/revenue-licence/login?returnTo=/apps/revenue-licence`
   — also silent.
3. **Cold entry:** in a fresh incognito window, go straight to
   `http://localhost:8090/bff/driving-licence/login?returnTo=/apps/driving-licence` with no
   prior portal visit. This one **does** prompt (fresh IS session) — sign in, land in the app.
4. **Single logout:** from the original window/session,
   `POST http://localhost:8090/bff/portal/logout` with the CSRF header (or drive it from a
   browser once the SPA exists in M5) → follow the returned IS logout URL. Then confirm all
   three of `GET /bff/portal/session`, `GET /bff/driving-licence/session`, `GET
   /bff/revenue-licence/session` return `401`, and check the BFF's own log for one
   back-channel-logout call reporting **3** sessions destroyed by a single `sid`.
5. Re-enter any app after that — it must re-prompt (no leftover session).

## 8. M3 — run `gov-services-api` and prove audience/scope separation

M3 adds `gov-services-api`, a resource server the BFF calls on the citizen's behalf using the
OAuth2 access token captured at login. It never talks to a browser and holds no secrets — it
only validates the JWT access token it's given (signature via IS's JWKS, `iss`, `exp`, then a
**per-router required audience and scope**), so App A's token is genuinely rejected by App B's
router and vice versa.

### 8.1 Configure and run it

```bash
cd citizen-portal-integration/gov-services-api
cp .env.example .env && chmod 600 .env
```

Fill in the **same three client IDs** already sitting in `citizen-portal-bff/.env` — these are
not secrets, they're the expected `aud` value for each router:

```
PORTAL_CLIENT_ID=<same value as citizen-portal-bff/.env>
DL_CLIENT_ID=<same value as citizen-portal-bff/.env>
VRL_CLIENT_ID=<same value as citizen-portal-bff/.env>
```

Then run it alongside the BFF (a fourth terminal, on top of the three from §7 — eSignet/IS,
`setup-without-bridge`, and the BFF itself):

```bash
./run-govapi.sh
```

`GET http://localhost:8091/portal/catalogue` with no `Authorization` header should return `400`
(missing bearer token) — that alone confirms the service is up and its middleware is active,
with no live IS round trip needed yet.

### 8.2 Prove the audience separation with curl

This needs one real access token, captured by hand — the BFF never exposes a citizen's access
token to anything (by design, per `PORTAL-INTEGRATION-PLAN.md`'s "tokens must never reach the
browser"), so the only way to hold one yourself is to run the same authorization-code + PKCE
exchange the BFF runs, substituting yourself for it. This mirrors
`setup-without-bridge/MANUAL-STEPS.md` §5's own hand-built-PKCE-URL pattern.

**Stop the running BFF first** (`Ctrl-C` the `./run-bff.sh` from §7) — you're about to send a
browser to `driving-licence`'s own registered callback URL, and if the BFF is still running it
will intercept and consume the authorization code itself before you can copy it.

```bash
CLIENT_ID='<Driving Licence Service Client ID, from MANUAL-STEPS.md §3.7>'
CLIENT_SECRET='<its Client Secret, from the same step>'
REDIRECT_URI='http://localhost:8090/bff/driving-licence/callback'   # already registered, §3.3

VERIFIER=$(openssl rand -base64 96 | tr -d '\n=' | tr '/+' '_-' | cut -c1-64)
CHALLENGE=$(printf %s "$VERIFIER" | openssl dgst -binary -sha256 | openssl base64 | tr -d '\n=' | tr '/+' '_-')
echo "verifier = $VERIFIER"
echo "https://localhost:9443/oauth2/authorize?response_type=code&client_id=$CLIENT_ID&redirect_uri=$REDIRECT_URI&scope=openid%20profile%20email%20address&code_challenge=$CHALLENGE&code_challenge_method=S256"
```

Open that URL in a browser (with the BFF stopped): sign in via MOSIP eSignet (or Username &
Password) same as always. With nothing listening on `:8090`, the browser lands on a connection
error — expected, per §7's own note about `/` 404s in M1 — but the address bar now holds
`...callback?code=...&state=...`. Copy the `code` value, then redeem it yourself:

```bash
curl -sk -X POST https://localhost:9443/oauth2/token \
  -d grant_type=authorization_code \
  -d client_id="$CLIENT_ID" \
  -d client_secret="$CLIENT_SECRET" \
  -d redirect_uri="$REDIRECT_URI" \
  -d code='<code-from-the-url>' \
  -d code_verifier="$VERIFIER" | python3 -m json.tool
```

Take the response's `access_token` and prove both directions:

```bash
TOKEN='<access_token from above>'

curl -s -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $TOKEN" \
  http://localhost:8091/driving-licence/config          # expect 200 — this is Application A's own router

curl -s -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $TOKEN" \
  http://localhost:8091/vehicle-registry/vehicles        # expect 403 — Application A's token, Application B's router
```

That `200` then `403` pair — the same token accepted by its own router and rejected by the
other's — is M3's whole point (`PORTAL-INTEGRATION-PLAN.md`'s M3 row: "curl proves App A's token
is accepted by `/driving-licence/*` and rejected by `/vehicle-registry/*`").

Restart the BFF (`./run-bff.sh`) once you're done — it's needed again for anything else on this
page.

### 8.3 What's still unverified after this

- The full BFF-fronted path (`GET /bff/driving-licence/api/config` through the running BFF,
  using the session cookie rather than a hand-held token) — §8.2 deliberately tests
  `gov-services-api` directly to isolate the audience/scope claim, but the BFF's own
  `/bff/{app}/api/...` routes calling through to it have not yet been exercised against a live
  stack end to end. Once the BFF is running again, `curl -b <session cookie jar>
  http://localhost:8090/bff/driving-licence/api/config` (after a normal browser login) is the
  equivalent check for the BFF-fronted path.

---

## When you have to redo this page

| You did | What breaks | Redo |
|---|---|---|
| Anything in `setup-without-bridge/MANUAL-STEPS.md`'s own "when to redo" table | The eSignet connection or the underlying client | That page's table first, then confirm the **MOSIP eSignet** option is still listed in step 2 above |
| Regenerated the Citizen Portal / Driving Licence Service / Vehicle Revenue Licence application's client secret in the Console | The matching `.env` secret (`PORTAL_CLIENT_SECRET` / `DL_CLIENT_SECRET` / `VRL_CLIENT_SECRET`) no longer matches | Update `.env` with the new secret — this is the same trap `esignet-bridge/.env`/`BRIDGE_API_KEY` has, except IS issues the secret so it cannot be re-derived; you must copy the new value from the Console |
| Deleted the "Citizen Portal" application | Nothing else — it is only referenced by `.env`'s `PORTAL_CLIENT_ID`/`_SECRET` | Repeat §1–§2 |
| Deleted the "Driving Licence Service" or "Vehicle Revenue Licence" application | Nothing else — each is only referenced by its own `.env` pair | Repeat §3 or §4 |
| Regenerated any of the three applications' client secret | `gov-services-api/.env` is unaffected (it never holds a secret) but `citizen-portal-bff/.env`'s matching `_SECRET` no longer matches | Same fix as the row above |
| Deleted or recreated any of the three applications | `gov-services-api/.env`'s matching `_CLIENT_ID` (it uses the same three IDs as the expected `aud` per router) | Update `gov-services-api/.env` with the new client ID alongside updating `citizen-portal-bff/.env` |
