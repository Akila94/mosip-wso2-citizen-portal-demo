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

## 5. Custom scopes — not required for M2, needed before M3

`PORTAL-INTEGRATION-PLAN.md` gives Application A the scope `driving_licence.write` and
Application B `vehicle_registry.read`, so `gov-services-api` (M3) can reject one app's token on
the other's router. `citizen-portal-bff/.env.example`'s `DL_CLIENT_SCOPES` /
`VRL_CLIENT_SCOPES` defaults already include them, on the assumption that IS drops an
**unrecognized** scope from a request rather than rejecting the whole authorization — this has
**not been verified live** and must be confirmed during M2's own live test below. If login
fails or the scope is silently absent from the token, the workaround until M3 defines these as
real OAuth API-resource scopes (**API Resources** → **New API Resource**, then bind them on
each application's **API Authorization** tab) is to drop the custom scope from that app's
`_CLIENT_SCOPES` value in `.env` and retry.

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

---

## When you have to redo this page

| You did | What breaks | Redo |
|---|---|---|
| Anything in `setup-without-bridge/MANUAL-STEPS.md`'s own "when to redo" table | The eSignet connection or the underlying client | That page's table first, then confirm the **MOSIP eSignet** option is still listed in step 2 above |
| Regenerated the Citizen Portal / Driving Licence Service / Vehicle Revenue Licence application's client secret in the Console | The matching `.env` secret (`PORTAL_CLIENT_SECRET` / `DL_CLIENT_SECRET` / `VRL_CLIENT_SECRET`) no longer matches | Update `.env` with the new secret — this is the same trap `esignet-bridge/.env`/`BRIDGE_API_KEY` has, except IS issues the secret so it cannot be re-derived; you must copy the new value from the Console |
| Deleted the "Citizen Portal" application | Nothing else — it is only referenced by `.env`'s `PORTAL_CLIENT_ID`/`_SECRET` | Repeat §1–§2 |
| Deleted the "Driving Licence Service" or "Vehicle Revenue Licence" application | Nothing else — each is only referenced by its own `.env` pair | Repeat §3 or §4 |
