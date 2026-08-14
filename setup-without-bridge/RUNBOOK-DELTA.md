# Runbook delta — the bridge-free build

`../esignet-bridge/esignet-wso2-is-federation-runbook.md` remains the authoritative spec.
This page lists only where this folder departs from it, and carries the live verification
checklist.

## Section by section

| Runbook section | Status here |
|---|---|
| §1 the blocker | **Unchanged and still true.** The *stock* outbound OIDC connector cannot talk to eSignet. What changes is the answer: §1 offers the service-based custom authenticator (the bridge); this folder takes the other route the runbook itself names in §18.1. |
| §2 architecture | One hop shorter — see the diagram in `README.md`. eSignet redirects to `https://localhost:9443/commonauth`, not to a bridge. |
| §3 prerequisites | Node is **not** needed. A JDK (11/17/21) and Maven are. |
| §4 run eSignet | Unchanged (`./demo.sh setup`/`start`). |
| §5 test citizen | Unchanged, same payload including `encodedPhoto` and a generated `requestTime`. |
| §6 generate the client keypair | **Removed.** No `genKeys.js`, no `private.jwk.json`. The assertion is signed with IS's own OAuth signing key from its keystore; `./demo.sh jwk` exports the public half. |
| §7 register the OIDC client | Same call, three fields differ: `publicKey` is the JWK of the IS signing key, `redirectUris` is `["https://localhost:9443/commonauth"]`, `clientName` is `WSO2 Identity Server`. `clientAuthMethods`, `authContextRefs` and `additionalConfig.userinfo_response_type: JWS` are unchanged. |
| §8 install WSO2 IS | Unchanged, plus: the built JAR is copied into `repository/components/dropins/`. |
| §9 build and run the bridge | **Removed entirely.** Replaced by `./demo.sh build` — a Maven OSGi bundle, no process to run, no API key, no `.env`. |
| §10 register the custom authenticator | Becomes **Connections → New Connection → Custom Authenticator (Plugin Based) → Settings → New Authenticator**. The service-endpoint flow the bridge uses does not apply to a JAR. Note the 7.3.0 Console label is *Custom Authenticator (Plugin Based)*, observed live — the docs still call this tile *Custom Connector* (`docs-is:.../configure-custom-connector.md`), and the underlying template is `expert-mode-idp`, historically shown as *Expert Mode*. Fields are in `MANUAL-STEPS.md` §1. |
| §11–§13 applications, first login, app B | Unchanged; reproduced in `MANUAL-STEPS.md` §3–§5. |
| §14.1 JIT provisioning | Unchanged. §14.2 session timeout is still applied by `setup`. |
| §15 preflight | `./demo.sh preflight`, with the bridge checks replaced by JAR/bundle/signing-key checks. |
| §17 troubleshooting | Still applies for eSignet-side errors. IS-side errors now surface as `ESIGNET-650xx` codes in `wso2carbon.log` rather than as bridge log lines. |
| §18.1 the bridge is a translating middlebox | **Retired.** This is the connector §18.1 asked for. IS terminates eSignet's tokens itself; nothing asserts identity on its own authority. |
| §18.2 one-sided logout | Unchanged — eSignet has no logout endpoint. |
| §18.3 pairwise `sub` | Unchanged. |
| §18.4 eSignet SSO is not a session | Unchanged, and still the right way to describe it. |
| §18.5 what was tested | Does not apply: that simulated-eSignet harness tested the bridge. Nothing in this folder has been run against a live stack yet. |
| §19 restarting | Two terminals rather than four, or just `./demo.sh restart`. |
| §20 appendix | All citations still hold. The ones this build depends on are repeated below. |

## Source facts this build rests on

Read from the WSO2 source, not recalled:

| Claim | Source |
|---|---|
| `getAccessTokenRequest` is `protected`, branches only on `IsBasicAuthEnabled`, and builds the body with Oltu's `TokenRequestBuilder` | `identity-outbound-auth-oidc@v5.15.2:components/…/OpenIDConnectAuthenticator.java:1470-1556` |
| `getSubjectAttributes` is `protected` and parses UserInfo as JSON | same file, `:439-455`, via `OIDCCommonUtil.extractUserClaimsFromJsonPayload` |
| PKCE S256 already exists in the product connector, gated by `IsPKCEEnabled` | same file, `:596-604` (challenge) and `:1505-1535` (verifier) |
| `nonce` is always sent and checked | same file, `:519` and `:1039-1050` |
| The ID token is signature-validated only on the native-SDK path, not the browser code flow | same file, `:1198-1237` |
| `org.wso2.carbon.identity.application.authenticator.oidc.*` is an exported package, so subclassing from another bundle resolves | `identity-outbound-auth-oidc@v5.15.2:components/…/pom.xml:190-194` |
| A reusable JWKS verifier exists in the product | `identity-inbound-auth-oauth@support-7.4.99.x-full:…/oauth2/util/JWTSignatureValidationUtils.java:138` — `validateUsingJWKSUri(SignedJWT, String)` |
| Signing key resolution API | `carbon-identity-framework@support-7.10.156.x-full:…/identity/core/IdentityKeyStoreResolver.java:204` — `getPrivateKey(tenantDomain, InboundProtocol.OAUTH)` |
| JAR authenticators deploy to `dropins` and are added to a custom connection | `docs-is:en/includes/references/extend/federation/write-a-custom-federated-authenticator.md` (the tile is *Custom Authenticator (Plugin Based)* in the 7.3.0 Console; the docs still say *Custom Connector*) |
| The Console's authenticator picker is server-driven, so a missing entry is a server-side problem | `GET /api/server/v1/identity-providers/meta/federated-authenticators` — verified live: returns `EsignetOIDCAuthenticator`, `definedBy: SYSTEM`, `authenticatorId RXNpZ25ldE9JRENBdXRoZW50aWNhdG9y` (base64 of the name) |
| A bundle that imports a `java.*` package Carbon's system bundle does not export stays UNRESOLVED in dropins | Observed live: `BundleException: Could not resolve module … Unresolved requirement: Import-Package: java.nio.file`, fixed with `!java.*` in the component pom's Import-Package |
| Versions IS 7.3.0 ships | `product-is@support-7.3.0.x-full:pom.xml` — framework `7.10.156` (:2681), inbound oauth `7.4.99.7` (:2710), outbound oidc `5.15.2` (:2742), nimbus `9.41.2` (:2983) |

Two deliberate deviations from those pins: the build compiles against
`org.wso2.carbon.identity.oauth:7.4.99` (the `.7` patch is not in the public WSO2 nexus; the
API used is unchanged), and the Nimbus JOSE import carries no version floor because the
orbit build that reaches the build classpath transitively (`10.3.0.wso2v1`) is newer than
what the runtime exports.

## Security review against the WSO2 secure engineering guidelines

Reviewed against `security.docs.wso2.com` (secure coding — general recommendations, plus the
tooling sections). What applies to a federated authenticator and a bash automation script,
and where it stands:

| Guideline | Status |
|---|---|
| §1.7 log injection / log forging (CRLF) | **Fixed during review.** Claim names, endpoint URLs and tenant domains now pass through `LogSanitizer.clean()` — CR/LF/tab stripped, length capped — before reaching a log or exception message. |
| OIDC Core 5.3.2 UserInfo `sub` binding | **Fixed during review.** The UserInfo `sub` must equal the ID token `sub` or the response is discarded (`ESIGNET-65009`). Without it, a UserInfo response for one subject could contribute attributes to a session authenticated as another. |
| Response-size bound | **Added during review.** A UserInfo body over 256 kB is rejected before parsing (`ESIGNET-65010`). |
| §1.16 insecure deserialization | Signature is verified **before** any claim is read; a failed or missing verification fails the login rather than returning claims. |
| §1.25 random number generation | `jti` uses `UUID.randomUUID()` (type 4, `SecureRandom`-backed); PKCE verifiers come from the product connector. No `java.util.Random`. |
| §1.11 heap inspection | `JwkExporter` holds the keystore password in a `char[]` and zeroes it after use; the private key is never copied out of the keystore abstraction. |
| §1.22 unvalidated redirects | The connector never accepts a redirect target from a request: the callback comes from the connection config or the authorization request the connector itself issued. |
| §1.3 OS command injection | `demo.sh` builds no command from untrusted input; subcommands and `logs`/`clean` arguments are whitelisted. |
| Secrets in `argv` | The keystore password reaches `JwkExporter` through the environment; nothing secret is passed on a command line. |
| TLS | No hostname-verification or trust-store weakening anywhere in the Java code. `--insecure` in `demo.sh` is scoped to `$IS_URL_BASE` only, for IS's self-signed certificate. |
| §8.2 static analysis (Find Security Bugs) | **Wired and green** — `mvn -Psecurity-checks verify -Ddependency-check.skip=true`. Three findings were triaged: two `CRLF_INJECTION_LOGS` (the detector cannot see through `LogSanitizer`) and one `PATH_TRAVERSAL_IN` on the operator-run CLI's keystore path. All three excluded with written justifications in `spotbugs-exclude.xml`. |
| §1.17 known vulnerable components | **Wired, not run.** The profile carries `dependency-check-maven`, but NVD rejects the legacy 1.1 feeds (`HTTP 403`, verified with plugin 8.4.3), so 12.x plus a free NVD API key is the only working configuration and none was available here. Mitigating fact: every dependency is `provided` — nothing third-party ships inside the bundle. |
| §1.20 SSRF | Accepted by design, as in the product connector: the token, UserInfo and JWKS URLs are administrator configuration, so a connection admin can point the connector at internal hosts. The guideline's rule concerns URLs taken from end users. |
| §1.18 logging & monitoring | Failures carry `ESIGNET-650xx` codes and are logged by the framework, and the inherited flow emits the product's diagnostic-log events. The two overrides do not add diagnostic-log events of their own — a known, minor gap. |
| §1.1/§1.2/§1.4/§1.5/§1.19/§1.21/§1.23/§1.24/§1.26 | Not applicable: no SQL, no LDAP, no rendered output, no XML parsing, no cookies of our own, no framing, no CORS surface, no file upload. |

Nothing in the review needed an **⚠ APPROVAL REQUIRED** decision, with one judgement call
worth naming for a reviewer: the SSRF row above is an accepted design property inherited
from the product connector, not something this connector introduces.

## Verified before the live run

- `mvn clean install` succeeds; the bundle manifest imports the product packages and
  exports its own, with `javax.servlet.http` absent and Nimbus unversioned.
- `JwkExporter` run standalone against a `keytool`-generated JKS produces a JWK whose `kid`
  matches an independent RFC 7638 implementation, so the assertion's `kid` and the registered
  `publicKey` agree by construction.
- `demo.sh`: `help`, `build`, `status`, `jwk`, and the connector/signing-key sections of
  `preflight` exercised against a stand-in IS tree, including the `deployment.toml`
  keystore-config parsing.
- Find Security Bugs passes with no unexcluded findings, from both the aggregator and the
  module directory.
- **The bundle resolves and activates in a live IS 7.3.0** — `wso2carbon.log` logs
  `MOSIP eSignet federated authenticator bundle is activated`, and the federated-authenticator
  meta API lists it as `MOSIP eSignet` with all twelve configuration properties, `ClientId`,
  `OAuth2AuthzEPUrl`, `OAuth2TokenEPUrl`, `UserInfoUrl`, `EsignetJwksUrl` and
  `AssertionValidityMinutes` among them, and with no `ClientSecret` or `IsBasicAuthEnabled`.
  This was reached only after fixing the `java.*` import problem above; the first deployment
  did not resolve.
- The module compiles warning-free under `-Xlint:all,-processing -Werror`, so **any deprecated
  API usage is a build failure**. Confirmed by reintroducing the deprecated
  `ServiceURLBuilder.build()` and watching the build fail on it, then restoring the fix.

Not verified: the federation flow itself, eSignet's real error strings, and the CSRF-cookie
extraction in client registration (fall back to `esignet/postman-collection/`).

## Live checklist

Run in this order and record what actually happened, here, after the run.

1. `./demo.sh build` → JAR in `wso2is-7.3.0/repository/components/dropins/`.
2. `./demo.sh setup` → citizen created, client `ACTIVE`, JWK registered.
3. `./demo.sh start`, then
   `curl -s http://localhost:8088/v1/esignet/oidc/.well-known/openid-configuration | python3 -m json.tool`.
4. `./demo.sh preflight` — all green, including `authenticator bundle activated in this IS run`.
5. Console lists **MOSIP eSignet** under the connection's Settings tab.
6. Open App A's PKCE `/authorize` URL → IS login page shows the eSignet option.
7. Browser lands on `:3000/authorize` carrying `acr_values`, `nonce`, `code_challenge`,
   `code_challenge_method=S256`.
8. Log in as `8267411571`, OTP `111111`, approve consent.
9. `./demo.sh logs esignet` → `POST /oauth/v2/token` succeeded; no `invalid_client_assertion`,
   no `invalid_client_id`, no `invalid_redirect_uri`.
10. `./demo.sh logs is` → no `JSONUtils.parseJSON` failure and no `ESIGNET-650xx` code;
    claims mapped.
11. Console → **User Management → Users** shows the JIT-provisioned user with mapped
    attributes, username = eSignet `sub`.
12. Open App B → logged in with no eSignet screen (the SSO session belongs to IS). Then sign
    out of IS and confirm both applications re-prompt.

If step 9 fails with `invalid_client_assertion`, the first thing to check is that
`./demo.sh jwk`'s `kid` matches the `publicKey` registered on the eSignet client — a
replaced `wso2is-7.3.0/` is the usual cause.
