# Manual steps — citizen-portal-integration

Console work `citizen-portal-bff` cannot do for you. Console: `https://localhost:9443/console`,
`admin` / `admin`. This assumes `setup-without-bridge/./demo.sh setup && ./demo.sh start` has
already run and `./demo.sh preflight` is green — the MOSIP eSignet connection and its
`EsignetOIDCAuthenticator` bundle must already exist before you add a sign-in option that uses
it.

Work through this page in order. It currently covers **M1 only**: one application, "Citizen
Portal". Applications A (Driving Licence) and B (Vehicle Revenue Licence) are added in the same
style once M2 starts.

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

## 3. Fill in `.env`

Copy `citizen-portal-bff/.env.example` to `citizen-portal-bff/.env` and set:

```
PORTAL_CLIENT_ID=<Client ID from step 1.7>
PORTAL_CLIENT_SECRET=<Client Secret from step 1.7>
```

Everything else in `.env.example` has a working default for a local `setup-without-bridge`
environment on the default ports.

## 4. Run it

```bash
cd citizen-portal-integration/citizen-portal-bff
go run ./cmd/bff
```

Then open `http://localhost:8090/bff/portal/login?returnTo=/` in a browser. You should land on
IS's login page with both **Username & Password** and **MOSIP eSignet** offered. Choosing MOSIP
eSignet redirects to `localhost:3000`; individual ID `8267411571`, **Get OTP**, `111111`,
approve consent; you land back on `http://localhost:8090/` and
`GET /bff/portal/session` returns the real ID-token claims as JSON — no SPA involved yet, that
is M4/M5.

---

## When you have to redo this page

| You did | What breaks | Redo |
|---|---|---|
| Anything in `setup-without-bridge/MANUAL-STEPS.md`'s own "when to redo" table | The eSignet connection or the underlying client | That page's table first, then confirm the **MOSIP eSignet** option is still listed in step 2 above |
| Regenerated the Citizen Portal application's client secret in the Console | `.env`'s `PORTAL_CLIENT_SECRET` no longer matches | Update `.env` with the new secret — this is the same trap `esignet-bridge/.env`/`BRIDGE_API_KEY` has, except IS issues the secret so it cannot be re-derived; you must copy the new value from the Console |
| Deleted the "Citizen Portal" application | Nothing else — it is only referenced by `.env`'s `PORTAL_CLIENT_ID`/`_SECRET` | Repeat §1–§3 |
