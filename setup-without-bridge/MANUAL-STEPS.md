# Manual steps — bridge-free variant

Everything on this page is WSO2 Console work that no script can do. `./demo.sh setup`
does the rest.

Console: `https://localhost:9443/console`, `admin` / `admin`. Accept the self-signed
certificate warning once.

Work through §1 → §5 in order. §3–§5 are the same application work as the bridge demo;
they are repeated here so this folder stands alone.

---

## What the script already did

| Step | Automated by |
|---|---|
| clone eSignet, start the five containers | `setup`, `start` |
| create the test citizen `8267411571` | `setup` |
| download and unpack WSO2 IS 7.3.0 | `setup` |
| build the authenticator JAR and copy it into `repository/components/dropins/` | `setup`, `build` |
| export IS's public signing key as a JWK | `setup`, `jwk` |
| register the OIDC client `wso2-is-esignet` with that JWK | `setup` |
| `[session.timeout]` in `deployment.toml` | `setup` |
| pre-demo checks | `preflight` |

There is **no bridge, no API key and no `.env`** in this variant. The client assertion is
signed with IS's own keystore key, so nothing has to be copied into the Console by hand
except the values in §1.

---

## 1. Create the eSignet connection

**Before you start, the JAR must actually be loaded.** The authenticator can only appear in
the Console if its bundle resolved at IS startup, so if you have run `./demo.sh build` since
IS last started, restart IS first and confirm:

```bash
./demo.sh restart
grep -c "MOSIP eSignet federated authenticator bundle is activated" \
  wso2is-7.3.0/repository/logs/wso2carbon.log        # want 1
```

`./demo.sh preflight` checks the same line. If it is missing, go to *"MOSIP eSignet" is not
in the authenticator list* below before clicking anything.

This is a **JAR (plugin) based** federated authenticator, not the service-endpoint kind the
bridge demo used, so pick the *plugin* option:

1. **Connections** → **New Connection** → **Custom Authenticator (Plugin Based)**.
2. **Name:** `MOSIP eSignet` → optional description → **Finish**.
3. Open the connection → **Settings** tab → **New Authenticator**.
4. Select **MOSIP eSignet** → **Next**.
5. Fill in:

   | Field | Value |
   |---|---|
   | Client Id | `wso2-is-esignet` |
   | Authorization Endpoint URL | `http://localhost:3000/authorize` |
   | Token Endpoint URL | `http://localhost:8088/v1/esignet/oauth/v2/token` |
   | Userinfo Endpoint URL | `http://localhost:8088/v1/esignet/oidc/userinfo` |
   | Callback Url | `https://localhost:9443/commonauth` |
   | eSignet JWKS Endpoint URL | `http://localhost:8088/v1/esignet/oauth/.well-known/jwks.json` |
   | Scopes | `openid profile` |
   | Additional Query Parameters | `acr_values=mosip:idp:acr:generated-code&ui_locales=en&claims_locales=en` |
   | Enable PKCE | **on** |
   | Client Assertion Validity (minutes) | `5` (default) |

6. **Finish**.

> **eSignet has two base URLs and they are not interchangeable.** `:3000` is the login UI
> and serves only `/authorize` — the browser-facing hop. `:8088/v1/esignet` is the API and
> serves `/oauth/v2/token`, `/oidc/userinfo` and the JWKS, all called server-to-server.
> Mixing them up produces a 404 that looks like a client-registration failure.

Notes on three of those fields:

- **Enable PKCE** must stay on: eSignet requires PKCE, and this is the product connector's
  own S256 implementation, which the authenticator inherits unchanged.
- **Callback Url** must be byte-identical to the `redirectUris` value registered with
  eSignet (`./demo.sh setup` registers exactly this string).
- There is deliberately **no Client Secret field**. eSignet has no `client_secret` code path
  at all; the authenticator authenticates with a `private_key_jwt` assertion signed by IS's
  OAuth signing key, whose public half is already registered as the client's `publicKey`.

### "MOSIP eSignet" is not in the authenticator list

The Console builds that list from what the server reports, so an absent entry means IS itself
does not have the authenticator — not that you picked the wrong connection type. Ask the
server directly:

```bash
curl -sk -u admin:admin \
  https://localhost:9443/api/server/v1/identity-providers/meta/federated-authenticators \
  | python3 -m json.tool | grep -i esignet
```

Expect `"name": "EsignetOIDCAuthenticator"` with `"authenticatorId":
"RXNpZ25ldE9JRENBdXRoZW50aWNhdG9y"` (that id is just base64 of the name). If nothing comes
back, the bundle did not resolve at startup:

```bash
grep -A3 "Could not resolve module" wso2is-7.3.0/repository/logs/wso2carbon.log
```

An `Unresolved requirement: Import-Package: <package>` line names exactly what is missing.
The bundle imports only packages the IS 7.3.0 runtime exports, and deliberately imports no
`java.*` package at all — see the `!java.*` comment in the component `pom.xml`, which is
there because that precise failure (`java.nio.file`) left the bundle unresolved and the
Console with nothing to list. After any fix: `./demo.sh build && ./demo.sh restart`.

A resolved-but-idle bundle would instead show up as a missing activation line in
`wso2carbon.log`; both are covered by `./demo.sh preflight`.

## 2. Map attributes and configure JIT provisioning

The authenticator reports claims in the OIDC dialect (`http://wso2.org/oidc/claim`), so IS
maps them to local claims itself — but the connection still has to declare which ones it
wants.

1. **Connections** → **MOSIP eSignet** → **Attributes**.
2. Add the attributes eSignet returns, and confirm the local claim each maps to:

   | eSignet claim | Local claim |
   |---|---|
   | `email` | Email |
   | `given_name` | First Name |
   | `family_name` | Last Name |
   | `phone_number` | Mobile |
   | `birthdate` | Birth Date |
   | `gender` | Gender |
   | `name` | Full Name |

3. Set the **Subject attribute** to the default (`sub`). eSignet's `sub` is a pairwise
   pseudonymous identifier — never present it as a national ID number.
4. **Update**.
5. Same connection → **Just-in-Time Provisioning**:
   - **Just-in-Time (JIT) User Provisioning** — checked.
   - **JIT provisioning scheme** — **Provision silently**.
   - **Attribute synchronization method** — **Override All**.
   - **Update**.

Without JIT the login succeeds but no user appears under **Users**, which is the part of the
demo that shows there is no pre-registration.

## 3. Create application A

1. **Applications** → **New Application** → **Single-Page Application**.
2. **Name:** `Service A`
3. **Authorized redirect URL:** `http://localhost:5173` → **Create**.
4. Open it → **Login Flow** → **Add Sign-In Option** on **Step 1**.
5. Select **MOSIP eSignet** → **Add** → **Update**.

> Keep eSignet in **Step 1**. That step is what establishes who the user is.

Note the application's **Client ID** from the Protocol tab — §5 needs it. That tab also has
**Mandatory PKCE** on by default; leave it on and let §5 supply the challenge.

## 4. Create application B

Repeat §3, changing only:

- **Name:** `Service B`
- **Authorized redirect URL:** a different port, e.g. `http://localhost:5174`

Add the same **MOSIP eSignet** option to its Step 1 and **Update**.

**This second application is the whole point of the demo** — it is what turns a login demo
into an SSO demo.

## 5. First end-to-end login

Easiest path is **Try it** on the application's Quick Start tab — it builds a compliant
request for you, PKCE included.

To drive it by hand, the URL **must carry PKCE**: the Single-Page Application template makes
the app a public client with **Mandatory PKCE** on, so a plain `/authorize` is rejected with
`PKCE is mandatory for this application` before you ever reach the eSignet screen. This is
IS-side PKCE, between your browser and Identity Server — unrelated to the PKCE the
authenticator performs against eSignet.

Replace the placeholder on the first line with the Client ID from §3 — keep the quotes,
and do not paste it as `<...>`: in both zsh and bash a bare `<` is an input redirection and
the line dies with `parse error near '\n'`.

```bash
CLIENT_ID='PASTE_THE_CLIENT_ID_HERE'
VERIFIER=$(openssl rand -base64 96 | tr -d '\n=' | tr '/+' '_-' | cut -c1-64)
CHALLENGE=$(printf %s "$VERIFIER" | openssl dgst -binary -sha256 | openssl base64 | tr -d '\n=' | tr '/+' '_-')
echo "verifier = $VERIFIER"   # needed only if you also want to redeem the code below
echo "https://localhost:9443/oauth2/authorize?response_type=code&client_id=$CLIENT_ID&redirect_uri=http://localhost:5173&scope=openid%20profile&code_challenge=$CHALLENGE&code_challenge_method=S256"
```

Open that URL, then:

1. Click **MOSIP eSignet** → you land on `localhost:3000`.
2. Individual ID: `8267411571`
3. **Get OTP** → enter `111111`
4. Approve the consent screen.

The browser lands on `http://localhost:5173/?code=…` — a connection error there is expected
if nothing is serving that port. **The login has already succeeded**, and the `commonauthId`
SSO cookie is set, which is all the demo needs. To prove the code is redeemable (a public
client sends no secret):

```bash
curl -sk -X POST https://localhost:9443/oauth2/token \
  -d grant_type=authorization_code \
  -d client_id="$CLIENT_ID" \
  -d redirect_uri=http://localhost:5173 \
  -d code='<code-from-the-url>' \
  -d code_verifier="$VERIFIER" | python3 -m json.tool
```

Confirm each hop:

| Where | What you should see |
|---|---|
| the eSignet screen URL | `acr_values`, `nonce`, `code_challenge` and `code_challenge_method=S256` in the query string |
| `./demo.sh logs esignet` | a successful `POST /oauth/v2/token` — no `invalid_client_assertion`, no `invalid_client_id` |
| `./demo.sh logs is` | no `JSONUtils.parseJSON` error, no `ESIGNET-650xx` error code |
| Console → **Users** | a new user whose username is the eSignet `sub` |
| that user's profile | email, first name, last name, mobile populated |
| Service B's login URL | logs in with no eSignet screen at all |

---

## Fallback: if client registration failed during setup

`setup` prints a blocker line if it could not register the client (usually a CSRF-cookie
problem). Register it by hand with the public JWK of IS's signing key:

```bash
./demo.sh jwk        # the JWK to paste as "publicKey"
```

Then use `esignet/postman-collection/` → **Create OIDC Client**, with:

| Field | Value |
|---|---|
| `clientId` | `wso2-is-esignet` |
| `publicKey` | the `./demo.sh jwk` output, as a JSON object |
| `redirectUris` | `["https://localhost:9443/commonauth"]` |
| `clientAuthMethods` | `["private_key_jwt"]` |
| `authContextRefs` | `["mosip:idp:acr:generated-code","mosip:idp:acr:password"]` |
| `additionalConfig` | `{"userinfo_response_type":"JWS"}` |
| `userClaims` | `["name","email","gender","phone_number","birthdate","picture"]` |
| `relyingPartyId` | `mock-relying-party-id` |

`requestTime` must be a current UTC timestamp; the runbook's "working" payload variants are
the ones that substitute `$(date -u …)` for it.

---

## When you have to redo the manual steps

| You ran | What breaks | Redo |
|---|---|---|
| `./demo.sh clean` | nothing — logs and the IS zip only | nothing |
| `./demo.sh build` | nothing; restart IS to load the new JAR | nothing |
| `./demo.sh clean --all` | everything: IS, eSignet, build output, volumes | `./demo.sh setup` **and** everything on this page |
| `./demo.sh reset-wso2` | all IS config and users, but **not** the keystore, so the eSignet client stays valid | everything on this page |
| deleted `wso2is-7.3.0/` | the keystore too, so the registered `publicKey` is now stale | `./demo.sh setup`, register a client with a **new** `clientId` (or reset eSignet), then everything on this page |
| `docker compose down -v` in `esignet/docker-compose` | the Postgres volume: test citizen **and** registered client | `./demo.sh setup`, no Console work |
| anything that **recreates the esignet container** without resetting the DB volume | eSignet exits on boot with `KER-KMA-004 No such alias` — permanently | `docker compose down -v` then `./demo.sh setup` |
| regenerated the IS keystore | the registered `publicKey` no longer matches; every token call fails `invalid_client_assertion` | re-register the eSignet client with the new `./demo.sh jwk` output |

The keystore is what binds this deployment to the eSignet client. Treat
`wso2is-7.3.0/repository/resources/security/wso2carbon.jks` as the thing not to lose —
it is the counterpart of `private.jwk.json` in the bridge variant.

---

## Known gaps to state openly during the demo

- **Logout is one-sided.** Signing out of IS destroys the `commonauthId` session, so both
  applications re-authenticate. It does not propagate to eSignet, which exposes no
  logout/`end_session` endpoint at all.
- **eSignet's `sub` is a pairwise PSUT** — a different opaque value per relying party. It
  cannot correlate the citizen across services, which is a privacy feature.
- **eSignet's "SSO" is not a browser session** — no session cookie, no cross-RP session
  state. WSO2 IS owns the actual SSO session, which is the point of the demo.
- **Encrypted (JWE) UserInfo is not implemented.** The eSignet client is registered with
  `userinfo_response_type: JWS`; a JWE response fails with `ESIGNET-65007`.
