# eSignet ↔ WSO2 IS federation, without the bridge

The same demo as the parent folder — a MOSIP eSignet national-ID login driving SSO across
two applications in WSO2 Identity Server 7.3.0 — built the way
`../esignet-bridge/esignet-wso2-is-federation-runbook.md` §18.1 calls production-correct: **a custom
federated authenticator JAR that extends the product's `OpenIDConnectAuthenticator`**,
rather than a Node.js middlebox between IS and eSignet.

When this README and the runbook disagree, the runbook wins. `RUNBOOK-DELTA.md` lists every
place this variant deliberately departs from it.

## Why a JAR rather than the stock OIDC connection

The stock "Standard-Based OIDC" connection cannot talk to eSignet, for two reasons, both
verified in source:

1. eSignet authenticates clients at the token endpoint with `private_key_jwt` only —
   `client_assertion` is `@NotBlank` on its `TokenRequest` and there is no `client_secret`
   code path. WSO2's **outbound** OIDC connector implements only `client_secret_basic` and
   `client_secret_post`.
2. eSignet always returns UserInfo as a signed JWT; the outbound connector parses UserInfo
   as plain JSON. eSignet also puts no user claims in the ID token, so skipping UserInfo is
   not a workaround.

This is a limitation of the **outbound** direction only. IS supports `private_key_jwt` fully
*inbound*, for clients authenticating *to* IS, where it is a selectable token-endpoint auth
method and the FAPI default. The two directions are separate implementations in separate
repositories.

Both methods are `protected`, which is the intended extension seam — so this variant
overrides exactly those two and inherits everything else.

## Architecture

```
Browser
  │ 1. open App A
  ▼
WSO2 Identity Server 7.3.0  ── https://localhost:9443
  │  EsignetOIDCAuthenticator (JAR in repository/components/dropins)
  │  2. redirect to eSignet /authorize  (+ nonce, PKCE S256, acr_values)
  ▼
eSignet UI  ── http://localhost:3000
  │  user enters national ID, receives OTP, consents
  ▼
eSignet service  ── http://localhost:8088/v1/esignet
  │  3. redirects browser to  https://localhost:9443/commonauth?code=..&state=..
  ▼
WSO2 Identity Server
  │  4. POST /oauth/v2/token   (private_key_jwt assertion signed with IS's own key + PKCE verifier)
  │  5. GET  /oidc/userinfo    (signed JWT, verified against eSignet's JWKS)
  │  6. JIT-provisions the user, creates the SSO session (commonauthId cookie)
  ▼
App A logged in.   App B then logs in from the IS session — eSignet is never called again.
```

Compared with the bridge variant: one fewer network hop, one fewer process, no shared API
key, and no component asserting identity to IS on its own authority.

### Port map

| Component | Port | URL |
|---|---|---|
| eSignet UI | 3000 | `http://localhost:3000` |
| eSignet service | 8088 | `http://localhost:8088/v1/esignet` |
| Mock identity system | 8082 | `http://localhost:8082/v1/mock-identity-system` |
| PostgreSQL | 5455 | (internal) |
| Redis | 6379 | (internal) |
| WSO2 IS | 9443 | `https://localhost:9443` |

## Quick start

Prerequisites: Docker (with compose v2), a JDK 11/17/21, Maven, `git`, `curl`, `unzip`,
`python3`. No Node.

```bash
./demo.sh setup       # eSignet clone + containers, IS download/unpack, JAR build+deploy,
                      # test citizen, eSignet client registration
./demo.sh start       # eSignet stack + WSO2 IS
# → then work through MANUAL-STEPS.md in the Console
./demo.sh preflight   # pre-demo checks
```

Everything `setup` does is idempotent; re-running it skips completed work. Other
subcommands: `build`, `stop`, `restart`, `status`, `jwk`, `logs is|esignet`,
`clean [--all]`, `reset-wso2`. `./demo.sh` with no arguments prints the list.

The security gates the WSO2 secure engineering guidelines ask for live in a Maven profile,
outside the normal build:

```bash
cd esignet-oidc-authenticator
mvn -Psecurity-checks verify -Ddependency-check.skip=true   # Find Security Bugs (offline)
mvn -Psecurity-checks verify -DnvdApiKey=$NVD_API_KEY       # + OWASP Dependency Check
```

`RUNBOOK-DELTA.md` records the full guideline-by-guideline review.

`stop` uses `docker compose stop`, never `down -v` — the latter destroys the test citizen
and the registered eSignet client.

## Repository layout

| Path | Role |
|---|---|
| `esignet-oidc-authenticator/` | Maven project for the connector: an aggregator pom and one OSGi bundle module. |
| `…/esignet/EsignetOIDCAuthenticator.java` | The connector. Two overrides, both documented in place. |
| `…/esignet/ClientAssertionBuilder.java` | Builds and signs the `private_key_jwt` assertion. |
| `…/esignet/JwkUtil.java` | RFC 7638 thumbprint and RFC 7517 JWK serialisation, plain JDK only. |
| `…/esignet/tools/JwkExporter.java` | Standalone `main` that prints the keystore's public JWK for eSignet client registration. |
| `…/esignet/internal/EsignetAuthenticatorServiceComponent.java` | Registers the authenticator as an OSGi `ApplicationAuthenticator`. |
| `demo.sh` | The operational entry point. Prefer extending it over adding scripts. |
| `MANUAL-STEPS.md` | The Console work `demo.sh` cannot do, plus the Postman fallback. |
| `RUNBOOK-DELTA.md` | What changes relative to the runbook, and the live verification checklist. |

Regenerated, never committed: `esignet/`, `wso2is-7.3.0/`, `**/target/`, `.run/`,
`wso2is-7.3.0.zip`.

## How the connector works

`EsignetOIDCAuthenticator extends OpenIDConnectAuthenticator`, so the authorization
request, `nonce`, PKCE, ID token handling, claim dialect mapping, JIT provisioning and the
SSO session are all the product's own code. Two methods are replaced:

**`getAccessTokenRequest`** builds the token request with
`client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-bearer` and a signed
assertion instead of client credentials, keeping the superclass's callback-URL resolution,
`code_verifier` handling and `Origin` header. The assertion has `iss` = `sub` = client id,
`aud` = the token endpoint URL exactly as configured, a fresh `jti` per request (eSignet
enforces uniqueness), and a 5-minute default lifetime.

**`getSubjectAttributes`** fetches UserInfo and applies three gates before a single claim is
trusted: a size cap on the response, JWS signature verification against eSignet's JWKS via
the product's own `JWTSignatureValidationUtils.validateUsingJWKSUri`, and the OpenID Connect
Core 5.3.2 check that the UserInfo `sub` exactly matches the ID token `sub`. Any of them
failing fails the login — it never falls through to returning partial or unverified claims.
Plain-JSON UserInfo is delegated back to the superclass, so the connector still works
against an ordinary OIDC provider. Encrypted (JWE) UserInfo is not implemented and fails
with a clear error.

Failures surface in `wso2carbon.log` under these codes:

| Code | Meaning |
|---|---|
| `ESIGNET-65001` | The tenant's OAuth signing key could not be resolved, or is not RSA |
| `ESIGNET-65002` | Signing the client assertion failed |
| `ESIGNET-65003` | Building the token request failed |
| `ESIGNET-65004` | The UserInfo call failed at the transport level |
| `ESIGNET-65005` | No JWKS endpoint configured, so a signed UserInfo response cannot be verified |
| `ESIGNET-65006` | The UserInfo signature did not validate |
| `ESIGNET-65007` | The UserInfo response is encrypted (JWE), which is not implemented |
| `ESIGNET-65008` | The UserInfo response is not a parseable JWT |
| `ESIGNET-65009` | UserInfo `sub` ≠ ID token `sub`; the response was discarded |
| `ESIGNET-65010` | The UserInfo response exceeded the size cap |

### The signing key

The assertion is signed with **IS's own OAuth signing key**, resolved through
`IdentityKeyStoreResolver.getPrivateKey(tenantDomain, InboundProtocol.OAUTH)` — so it
honours tenant keystores and any custom keystore mapped to the OAuth protocol, and there is
no key file and no shared secret anywhere in this design. `./demo.sh jwk` exports the public
half as a JWK; that is what `setup` registers as the eSignet client's `publicKey`, and the
`kid` in both places is the same RFC 7638 thumbprint, computed by the same `JwkUtil` code.

Consequence worth knowing: **the keystore binds this deployment to the registered eSignet
client.** Replacing `wso2is-7.3.0/` (or regenerating the keystore) invalidates the client;
see the redo table in `MANUAL-STEPS.md`.

## Configuration reference

All configuration is Console-side, on the connection's authenticator (`MANUAL-STEPS.md` §1).
There are no config files and no environment variables.

| Property | Meaning |
|---|---|
| `ClientId` | Client id registered with eSignet (`wso2-is-esignet` by default) |
| `OAuth2AuthzEPUrl` | eSignet `/authorize` — on the **UI** host, `:3000` |
| `OAuth2TokenEPUrl` | eSignet `/oauth/v2/token` — on the **API** host, `:8088/v1/esignet` |
| `UserInfoUrl` | eSignet `/oidc/userinfo` — API host |
| `callbackUrl` | `https://localhost:9443/commonauth`; must match the registered `redirectUris` byte for byte |
| `EsignetJwksUrl` | eSignet JWKS, used to verify signed UserInfo |
| `AssertionValidityMinutes` | Client assertion lifetime, default 5 |
| `IsPKCEEnabled` | Must be on — eSignet requires PKCE |
| `Scopes` | `openid profile` |
| `commonAuthQueryParams` | Additional query parameters, i.e. `acr_values=…` |

`ClientSecret` and `IsBasicAuthEnabled` are deliberately removed from the property list.

## Limitations

- **Logout is one-sided.** eSignet exposes no logout, `end_session`, front-channel or
  back-channel logout endpoint at all — only `/oauth/introspect`. Signing out of IS ends the
  IS session; nothing propagates to eSignet.
- **eSignet's "SSO" is not a session.** One credential across services, not a browser
  session: no session cookie, no cross-RP session state. IS owns the actual SSO session.
- **`sub` is a pairwise PSUT.** Never present or store it as a national ID number.
- **JWE UserInfo unsupported** (`ESIGNET-65007`).
- **The ID token signature is not validated.** eSignet's ID token carries no user claims and
  the product code path only decodes it; nothing here makes that worse, but it is not a
  guarantee either. The claims that matter come from the JWS-verified UserInfo response.
- **Back-channel URLs are administrator configuration**, so a connection admin can point the
  token, UserInfo and JWKS calls at internal hosts. This is inherited from the product OIDC
  connector rather than introduced here, and the secure engineering guidelines' SSRF rule
  concerns URLs taken from end users — but it is worth knowing who can reach what.
- **Dependency vulnerability scanning is wired but unrun** — see the `security-checks` profile
  in `esignet-oidc-authenticator/pom.xml`; it needs a free NVD API key. Static analysis
  (Find Security Bugs) does run offline and is green.
- **Not yet run end-to-end against a live stack.** The build, the JWK export and every local
  branch of `demo.sh` have been exercised; the federation flow itself has not. The checklist
  in `RUNBOOK-DELTA.md` is the first live run, not a record of one.
