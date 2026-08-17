# M1 session notes — walking skeleton

Record of what was built, what broke during live verification, and how each issue was
diagnosed and resolved. Companion to `PORTAL-INTEGRATION-PLAN.md` (the design) and
`MANUAL-STEPS.md` (the Console steps this session's fixes are now folded into).

**Status at the end of this session: M1 code-complete and live-verified.** A real citizen
logged in through WSO2 IS → MOSIP eSignet → OTP, and the BFF held genuine ID-token claims
(`sub`, `sid`, `amr`, `name`, `email`, correctly derived `assuranceLevel`) across multiple fresh
logins. One known limitation remains, documented at the end, with a root cause in WSO2 IS
itself rather than in this BFF.

---

## What was built

`citizen-portal-bff/` (Go), one WSO2 IS application ("Citizen Portal"), TDD throughout:

| Package | Responsibility | Tests |
|---|---|---|
| `internal/security` | Log-forging prevention (`SanitizeForLog`), open-redirect guard (`ValidateReturnTo`), PKCE S256, CSRF constant-time compare, security response headers | 25 |
| `internal/session` | Generic TTL-bounded store (oldest-evicted, background sweep) used for both login transactions and authenticated sessions; `amr`→assurance-level derivation | 15 |
| `internal/config` | Env-var-only configuration, fails fast with an aggregated error on missing/invalid values | 10 |
| `internal/oidcrp` | OIDC discovery against IS (pinned to IS's own cert, never `InsecureSkipVerify`), the authorization-code + PKCE round trip, ID-token verification, RP-initiated logout URL, back-channel logout-token verification | 12 |
| `internal/httpapi` | chi routes: login / callback / session / logout / backchannel-logout, cookie management, CSRF, request-body limits | 17 |
| `cmd/bff` | Wiring, graceful shutdown | — |

`certs/wso2is-local.pem` — IS's default shipped self-signed cert (`CN=localhost`), exported so
the BFF can pin its trust store rather than disabling TLS verification.

`MANUAL-STEPS.md` — the Console registration steps (superseded/extended by the findings below).

**Verification tooling, all green:**
```
go build ./... && go vet ./... && go test ./... -race   # 79 tests, all packages
gosec ./...          # 0 issues (3 justified #nosec suppressions — see below)
govulncheck ./...     # 0 vulnerabilities (after a toolchain bump, see below)
```

---

## Issues hit during setup and live verification

Each of these was a real finding, diagnosed from first principles (logs, source, API
inspection) rather than guessed at. Ordered as encountered.

### 1. `source .env` silently truncated a multi-word value

**Symptom:** the BFF started fine, but `PORTAL_CLIENT_SCOPES` — meant to be
`openid profile email` — was effectively just `openid`.

**Cause:** the very first run used `set -a; source .env; set +a` to load environment variables
for a quick `go run`. Bash's `source`/`.` *executes* each line as a shell command. An unquoted
value containing spaces (`KEY=value with spaces`) is parsed as an assignment followed by a
command invocation: `PORTAL_CLIENT_SCOPES=openid profile email` became "set
`PORTAL_CLIENT_SCOPES=openid`, then try to run a command named `profile` with argument
`email`" — which bash duly reported as `command not found: profile`. The assignment silently
kept only the first word.

**This is exactly why the rest of this repo's `demo.sh` scripts parse `.env` instead of
sourcing it** (`esignet-bridge`'s `load_env()` uses `grep`/parameter expansion, never `.`) — a
rule stated in this project's own conventions, and rediscovered here empirically rather than
just taken on faith.

**Fix:** wrote `run-bff.sh`, a small launcher that reads `.env` line by line
(`grep -E '^[A-Za-z_][A-Za-z0-9_]*=' .env`, `read -r key value` with `IFS='='`, `export
"$key=$value"`) and only then `exec go run ./cmd/bff` — no shell evaluation of the file's
right-hand side ever occurs. `.env`'s value is treated as an opaque string, not code.

### 2. Redirect to `/` 404'd after a successful login

**Symptom:** after approving consent in the eSignet UI, the browser landed on
`http://localhost:8090/` and got a 404.

**Diagnosis:** not a bug. M1 deliberately serves no page at `/` — there is no SPA integration
yet (that's M4/M5) and no static file server wired up. The 404 is chi's default response for an
unregistered route. What actually mattered — whether `handleCallback` completed successfully
before issuing that redirect — was confirmed by (a) no error logged by the BFF and (b) the
session cookie having been set, verified in the next step.

**Resolution:** none needed. Documented so it isn't mistaken for a failure during future runs.

### 3. Missing `name`/`email` in the first successful login's claims

**Symptom:** first successful login returned `{"sub": "...", "assuranceLevel": "substantial",
...}` with no `name`, `email`, or other profile claims.

**Diagnosis:** inspected the "mosip" connection via
`GET /api/server/v1/identity-providers/{id}` (read-only, admin-authenticated) and found:
```json
"claims": { "mappings": [] },
"provisioning": { "jit": { "isEnabled": false } }
```
`setup-without-bridge/MANUAL-STEPS.md` §2 (attribute mapping + JIT provisioning) had not been
completed for this connection.

**Fix:** completed §2 in the Console — added the 7 attribute mappings (`email`, `given_name`,
`family_name`, `phone_number`, `birthdate`, `gender`, `name`) and enabled JIT ("Provision
silently", "Override All"). Verified via the same read-only API call before asking for another
login attempt, rather than trusting the Console UI's "saved" state at face value — which turned
out to be the right instinct (see #4).

### 4. Two Console "Update" clicks that didn't persist on the first attempt

**Symptom:** after the fix in #3 was reportedly applied, `GET
.../identity-providers/{id}/claims` still showed `"mappings": []`. Separately, after adding
requested attributes to the Citizen Portal application's User Attributes tab, `GET
.../applications/{id}` still showed `claimConfiguration.requestedClaims` containing only
`http://wso2.org/claims/username`.

**Diagnosis:** rather than guess why the Console silently didn't save, each change was
independently re-verified via the read-only Server Configuration Management API immediately
after being told it was done, before proceeding. Both times, a second attempt at the same
Console edit did persist correctly, confirmed the same way.

**Fix:** no code change — this was purely an operator/Console-UI interaction issue (the exact
mechanism was not root-caused: candidates are clicking away before the tab's own save action
fired, or a slow Console-to-backend round trip). The general lesson applied throughout the rest
of the session: **verify every Console change via the read-only API before proceeding to the
next step**, rather than taking a "done" report at face value. This caught both misses
immediately instead of after a wasted login attempt.

### 5. JIT-provisioned user never appeared under Users, despite JIT being enabled

**Symptom:** with JIT enabled and claim mappings correct (confirmed via API), and login
returning full, correct claims (`name`, `email`, `sub`, `sid`, `amr`), `GET /scim2/Users` still
listed only the two pre-existing users (`admin`, `johndoe@gmail.com`) — no third user for the
eSignet-authenticated citizen.

**Diagnosis path:**
1. Ruled out a domain/userstore-scoping issue: `GET /api/server/v1/userstores` showed only the
   default `PRIMARY` store plus an unrelated `AGENT` store; `filter=userName+co+"<sub-fragment>"`
   found nothing anywhere.
2. Read `wso2is-7.3.0/repository/logs/wso2carbon.log` directly (not a documented API, but the
   ground truth) and found a `User provisioning failed!` `FrameworkException` on every login,
   deep inside the post-authentication JIT provisioning handler
   (`JITProvisioningPostAuthenticationHandler` → `DefaultProvisioningHandler` →
   `AbstractUserStoreManager.addUser`).
3. Followed the full `Caused by:` chain to its root:
   ```
   Caused by: org.wso2.carbon.user.core.UserStoreClientException:
     Date of Birth is not in the correct format of YYYY-MM-DD
       at ...SCIMUserOperationListener.validateClaimValueForRegex
   ```
   eSignet's mock identity system returns `birthdate` as `1987/11/25` (slash-separated); IS's
   SCIM2 layer rejects this for the standard Birth Date claim, and the entire user-creation
   transaction is aborted by that single field failing validation — nothing else about the
   user record is wrong.

**First attempted fix (did not work): loosening the Birth Date claim's regex.** Edited
**Attributes → Attribute Dialects → `http://wso2.org/claims` → Birth Date → Regular
expression** to `^\d{4}[-/]\d{2}[-/]\d{2}$`. Re-tested: identical failure, same exact log
message. Re-reading `SCIMUserOperationListener.validateClaimValueForRegex`'s call site
(`doPreAddUserWithID` → `validateClaimValue`) shows this specific error string is not driven by
the claim's configurable regex at all for the Date of Birth claim — WSO2 IS's SCIM2 listener
applies its own hardcoded `YYYY-MM-DD` format check for DOB regardless of what regex is
configured on the local claim. **This is a WSO2 IS behavior, not a configuration mistake** —
loosening the regex has no effect on this specific validator.

**Second attempted fix (also did not work, but clarified the actual architecture): removing
the `birthdate → Birth Date` mapping from the connection's Attributes tab.** Verified via API
that the mapping really was removed (6 entries remained, no `dob`). A fresh login still
returned `birthdate` in the session claims, **and** the same provisioning failure still
appeared in the logs.

This is the most useful finding of the session, because it corrects an assumption both
`MANUAL-STEPS.md` files carry: **the per-connection Attributes-tab mapping does not govern
which claims are released into the ID token, and does not gate what the JIT provisioning
handler attempts to write.** `setup-without-bridge/MANUAL-STEPS.md` §2's own preamble says so,
read literally: *"The authenticator reports claims in the OIDC dialect
(`http://wso2.org/oidc/claim`), so IS maps them to local claims itself — but the connection
still has to declare which ones it wants."* In practice:
- **ID-token claim release** is governed by the requesting **application's** own
  `claimConfiguration.requestedClaims` (Console: Applications → Citizen Portal → User
  Attributes tab) — this is what actually put `name`/`email`/`birthdate` into the BFF's session
  view, and removing the connection-level mapping had no effect on it.
- **JIT provisioning** writes whatever the framework's automatic OIDC-dialect-to-local-claim
  mapping resolves from the connector's raw response — also independent of the per-connection
  Attributes table, which is why the failure persisted after removing that row.

**Conclusion:** there is no Console-level fix available for the provisioning failure. A real
fix requires the `EsignetOIDCAuthenticator`'s `getSubjectAttributes()` (in
`setup-without-bridge/esignet-oidc-authenticator/...`) to reformat MOSIP eSignet's
`YYYY/MM/DD` into `YYYY-MM-DD` before returning it — a Java source change, out of scope for this
BFF milestone, and one that should be considered alongside the `LogSanitizer`/claim-handling
code already in that JAR rather than as a one-off patch.

**Decision (user's call):** accept the gap. Login, claim release, session establishment and
`assuranceLevel` derivation are all correct and fully verified; only local-user persistence is
affected, and it fails safe (no partial/corrupt user record — the whole provisioning
transaction is rejected). JIT stays enabled as configured.

---

## Toolchain and static-analysis fixes (not IS-related)

- **`govulncheck` initially reported 15 Go standard-library CVEs**, all already fixed in later
  1.25.x patch releases; the environment's `go` was 1.25.6. Fixed by `go mod edit
  -toolchain=go1.25.13` — `go build` auto-downloads the pinned toolchain locally (no system Go
  install touched), and a second `govulncheck` pass came back clean. Re-run this whenever a
  newer 1.25.x patch ships; nothing about the code itself was at fault.
- **`gosec` flagged 8 findings**, resolved as follows:
  - 5× `G104` (unhandled `json.Encoder.Encode` error) — genuine gap, not a false positive.
    Fixed by routing all JSON responses through a `Server.writeJSON` helper that logs (rather
    than silently drops) an encode failure.
  - 1× `G304` (path traversal via variable, on `IS_CA_FILE`) — justified suppression: this is
    deployment-time configuration, never end-user input, the same reasoning
    `setup-without-bridge/.../spotbugs-exclude.xml` already gives its one `PATH_TRAVERSAL_IN`
    exclusion for the analogous JwkExporter CLI path. Suppressed with `#nosec G304` plus that
    justification inline.
  - 2× `G124` (cookie missing Secure/HttpOnly/SameSite) — false positives: both flagged cookies
    already set all three attributes; gosec's checker apparently triggers whenever any flag is
    a variable rather than a literal `true`. Suppressed with `#nosec G124` plus an inline
    explanation of why the variable is intentional (the CSRF cookie must be non-`HttpOnly` by
    design; `Secure` is correctly derived from configuration, not hardcoded).
  - Final state: `gosec ./...` → 0 issues, 3 justified `#nosec` suppressions, matching the
    style (and honesty) of the existing `spotbugs-exclude.xml` in `setup-without-bridge/`.

---

## What is proven, concretely

Three separate fresh (incognito, new IS session) logins during this session, each producing a
distinct `sid` — confirming each was a genuine new authentication, not a cached BFF session
being replayed:

| Login | `sid` | Claims received |
|---|---|---|
| 1st (before §2 fixes) | `5bcbacd4-897d-4a0b-beff-99208462268c` | `sub` only |
| 2nd (after mapping + JIT + app claims fixed) | `8cc23c75-efab-4276-9f28-3c45e01ed523` | `sub`, `name`, `email`, `birthdate` |
| 3rd (after DOB regex attempt, no effect) | `b318aa4c-6378-464e-959f-294ad4cac89f` | same full claim set — confirms claim release is unaffected by the connection Attributes-tab edits |

Every login: `amr: ["EsignetOIDCAuthenticator"]` → `assuranceLevel` correctly derived as
`substantial`; `expiresAt` present and sane; no token of any kind ever appeared in a
browser-facing response (`GET /bff/portal/session` was inspected directly each time).

## Follow-ups worth carrying into M2+

- **Fix the DOB format at the source** — reformat `birthdate` in
  `EsignetOIDCAuthenticator.getSubjectAttributes()` before it reaches the framework, so JIT
  provisioning succeeds and a persisted user record becomes part of the demo again.
- **Console-change verification discipline**: given two silent Console-save misses this
  session, any future Console configuration step should be followed by a read-only API check
  before the next dependent step is attempted — this cost two unnecessary login round trips
  before the pattern was recognized.
- **`run-bff.sh`** is a dev convenience for this session only — it is not part of the plan's
  `portal-demo.sh` (M6) and should not be treated as the final orchestration story; M6 will
  fold proper `.env` parsing (not sourcing) into that script directly, following the same
  discipline `esignet-bridge/demo.sh` and `setup-without-bridge/demo.sh` already use.
