# Manual steps

`./demo.sh` automates everything in `esignet-wso2-is-federation-runbook.md` that can be
automated. What is left is **WSO2 Identity Server Console work** — clicking through screens in
`https://localhost:9443/console`. There is no supported API-free way around it for a demo, so it
stays manual.

**Prerequisite: Node 22.** Run `nvm use 22` (or equivalent) in any terminal you use for `demo.sh` —
`setup` and the bridge both refuse to run on an older Node.

Do these **once**, after `./demo.sh setup && ./demo.sh start`. They survive restarts, because they
live in the IS H2 databases under `wso2is-7.3.0/repository/database/`.

---

## What the script already did

So you don't repeat it:

| Runbook step | Automated by |
|---|---|
| 1 — clone eSignet, start the five containers | `setup`, `start` |
| 2 — create the test citizen `8267411571` | `setup` |
| 3 — npm install, generate the RS256 keypair | `setup` |
| 4 — register the OIDC client `wso2-is-bridge` | `setup` |
| 5 — download and unpack WSO2 IS 7.3.0, start it | `setup`, `start` |
| 6 — API key, run the bridge | `setup` (key → `.env`), `start` |
| 11.2 — `[session.timeout]` in `deployment.toml` | `setup` |
| 15 — preflight check | `./demo.sh preflight` |

Before starting, confirm all five services are up:

```bash
./demo.sh status
```

---

## 1. Register the custom authenticator (runbook Step 7)

You need the bridge API key. Print it with:

```bash
./demo.sh apikey
```

1. Open `https://localhost:9443/console`, sign in as `admin` / `admin`, accept the self-signed
   certificate warning.
2. **Connections** → **New Connection** → **Custom Authenticator** → **Create**.
3. Authenticator type: **External (Federated) User Authentication** → **Next**.
4. General Settings:
   - **Identifier:** `esignet`
   - **Display name:** `Sign in with eSignet`
5. Configuration:
   - **Endpoint:** `http://localhost:4000/authenticate`
   - **Authentication:** **API Key**
     - **Header name:** `x-api-key`
     - **Value:** the output of `./demo.sh apikey`
6. **Finish**.

> Do **not** pick "No Authentication". It works, but it leaves an unauthenticated
> identity-assertion endpoint open on your machine.

The Console will not show the key again after saving. If you lose it, `./demo.sh apikey` still has
it — it is stored in `.env`, which is gitignored.

## 2. Configure JIT provisioning (runbook Step 11.1)

**Connections** → **Sign in with eSignet** → **Just-in-Time Provisioning**:

1. **Just-in-Time (JIT) User Provisioning** — checked.
2. **JIT provisioning scheme** — **Provision silently** (the default; anything else adds a prompt
   mid-demo).
3. **Attribute synchronization method** — **Override All**, so attributes refresh from eSignet on
   every login.
4. **Update**.

Without this the login succeeds but no user appears under **Users**, which is the part of the demo
that shows there is no pre-registration.

## 3. Create application A (runbook Step 8)

1. **Applications** → **New Application** → **Single-Page Application**.
2. **Name:** `Service A`
3. **Authorized redirect URL:** `http://localhost:5173` (or whatever your test app uses) →
   **Create**.
4. Open it → **Login Flow** tab → **Add Sign-In Option** on **Step 1**.
5. Select **Sign in with eSignet** → **Add** → **Update**.

> Keep eSignet in **Step 1**. That step is what establishes who the user is.

Note the application's **Client ID** from the Protocol tab — step 5 below needs it. That same tab has
**Mandatory PKCE** enabled by default; leave it on and let step 5 supply the challenge. Turning it
off would make a hand-built login URL simpler but the demo less representative of a real public
client.

## 4. Create application B (runbook Step 10)

Repeat step 3 exactly, changing only:

- **Name:** `Service B`
- **Authorized redirect URL:** a different port, e.g. `http://localhost:5174`

Add the same **Sign in with eSignet** option to its Step 1 and **Update**.

**This second application is the whole point of the demo** — it is what turns a login demo into an
SSO demo.

## 5. First end-to-end login (runbook Step 9)

Easiest path is **Try it** on the application's Quick Start tab — it builds a compliant request for
you, PKCE included.

To drive it by hand instead, the URL **must carry PKCE**. The Single-Page Application template makes
the app a public client with **Mandatory PKCE** on, so a plain `/authorize` is rejected with
`PKCE is mandatory for this application` before you ever reach the eSignet screen.

This is IS-side PKCE, between your browser and Identity Server. It is unrelated to the PKCE the
bridge already does against eSignet — that one is internal to `server.js` and needs nothing from
you.

Generate a verifier and its S256 challenge:

```bash
VERIFIER=$(openssl rand -base64 96 | tr -d '\n=' | tr '/+' '_-' | cut -c1-64)
CHALLENGE=$(printf %s "$VERIFIER" | openssl dgst -binary -sha256 | openssl base64 | tr -d '\n=' | tr '/+' '_-')
echo "verifier  = $VERIFIER"      # keep this if you want to redeem the code
echo "challenge = $CHALLENGE"
```

Build the authorize URL with it (substitute the Client ID from step 3 and the challenge above):

```
https://localhost:9443/oauth2/authorize?response_type=code&client_id=_HZFXSjCzTjXQclpY2CGkaqZgjka&redirect_uri=http://localhost:5173&scope=openid%20profile


https://localhost:9443/oauth2/authorize?response_type=code&client_id=FKmlVb9sUsYHAxlEcw3fuz_HHrwa&redirect_uri=http://localhost:5173&scope=openid%20profile
```

Or have the shell print the whole thing:

```bash
CLIENT_ID='PASTE_THE_CLIENT_ID_HERE'   # from the Protocol tab
echo "https://localhost:9443/oauth2/authorize?response_type=code&client_id=$CLIENT_ID&redirect_uri=http://localhost:5173&scope=openid%20profile&code_challenge=$CHALLENGE&code_challenge_method=S256"
```

Open that in the browser, then:

1. Click **Sign in with eSignet** → you land on `localhost:3000`.
2. Individual ID: `8267411571`
3. **Get OTP** → enter `111111`
4. Approve the consent screen.

The browser then lands on `http://localhost:5173/?code=…` — a connection error is expected if
nothing is serving that port. **The login has already succeeded at that point**, and the
`commonauthId` SSO cookie is set, which is all the demo needs. If you want to prove the code is
redeemable, exchange it with the same verifier (a public client sends no secret):

```bash
curl -sk -X POST https://localhost:9443/oauth2/token \
  -d grant_type=authorization_code \
  -d client_id="$CLIENT_ID" \
  -d redirect_uri=http://localhost:5173 \
  -d code='<code-from-the-url>' \
  -d code_verifier="$VERIFIER" | python3 -m json.tool
```

Omitting `code_verifier` here fails with `invalid_grant` — the same PKCE requirement, at the token
endpoint this time.

Confirm each hop:

| Where | What you should see |
|---|---|
| `./demo.sh logs bridge` | `[login-ok] flow=<id> sub=<PSUT> claims=6` |
| Console → **Users** | a new user whose username is the eSignet `sub` |
| that user's profile | email, first name, last name, mobile populated |

Anything missing → runbook §17 (Troubleshooting).

---

## If `setup` fails part-way

**Just run `./demo.sh setup` again.** Every phase checks whether its work is already done and skips
it, so a re-run resumes rather than restarting: the clone, the IS download and unpack, `npm
install`, the `[session.timeout]` patch and the API key are all no-ops the second time. The keypair
is deliberately *never* regenerated once it exists, because a new key invalidates the registered
eSignet client.

Two failures do not stop setup, and are reported at the end as a blocker instead:

- **client registration** — everything else is complete; use the Postman fallback below.
- **citizen creation** — a duplicate is the normal outcome on every run after the first, and is
  harmless.

Anything else exits immediately with an `error:` line and is safe to retry.

## Fallback: if client registration failed during setup

`./demo.sh setup` prints `FAIL could not read the XSRF-TOKEN cookie` (or a non-`ACTIVE` response)
when eSignet's CSRF-cookie handshake does not behave. This is the one automated step known to be
fragile, and the runbook flags it as never having been verified against a live instance. Use the
officially documented Postman path instead:

1. Import `esignet/postman-collection/eSignet.postman_collection.json`
2. Import `esignet/postman-collection/eSignet-with-mock.postman_environment.json` and select it
3. Run **OIDC Client Mgmt → Mock → Get CSRF token**
4. Set environment variables: `client_id` = `wso2-is-bridge`,
   `client_public_key` = contents of `public.jwk.json`,
   `redirection_url` = `http://localhost:4000/callback`
5. Run **OIDC Client Mgmt → Mock → Create OIDC client**

Then `./demo.sh preflight` to confirm.

---

## Day-to-day

```bash
./demo.sh start        # bring everything up
./demo.sh preflight    # before every demo — must end passed=N failed=0
./demo.sh stop         # shut down, keeping all data
./demo.sh restart
./demo.sh logs bridge  # or: is | esignet
```

`preflight` restarts the bridge on success, so its log is clean and the `PREFLIGHT` flow entry is
gone before you present.

Before presenting, also: open a **fresh incognito window** (a stale `commonauthId` cookie will make
the SSO proof look like it already happened), and have both application URLs in separate tabs. Full
narration script: runbook §16.

---

## When you have to redo the manual steps

| You ran | What breaks | Redo |
|---|---|---|
| `./demo.sh clean` | nothing — logs, pid files and the IS zip only | nothing |
| `./demo.sh clean --all` | everything: files, keys, containers, volumes | `./demo.sh setup` **and** everything on this page |
| `./demo.sh reset-wso2` | all IS config and users | everything on this page |
| deleted `wso2is-7.3.0/` | same | everything on this page |
| `docker compose down -v` in `esignet/docker-compose` | the Postgres volume: test citizen **and** registered client | `./demo.sh setup` (recreates both), no Console work |
| anything that **recreates the esignet container** without resetting the DB volume | eSignet exits on boot with `KER-KMA-004 No such alias` — permanently | `docker compose down -v` then `./demo.sh setup` (see below) |
| `node genKeys.js` by hand | the new key does not match the registered client | re-register the client — delete it in eSignet or use the Postman fallback above |

Restarting, rebooting, and `./demo.sh stop` / `start` break nothing.

---

## The `KER-KMA-004` trap

MOSIP's `docker-compose.yml` splits eSignet's state across two different lifetimes, and this bites
hard enough to be worth knowing before it happens:

- **Key metadata** (`esignet.key_alias`) lives in Postgres, in an **anonymous volume** that Compose
  *preserves* when containers are recreated.
- **The matching private keys** live at `/home/mosip/hsm-client/ref.softhsm` **inside the esignet
  container's writable layer** — no volume, so they are destroyed on recreation.

Recreate the esignet container without resetting the DB and eSignet boots, reads an alias from the
database, fails to find its private key, and exits 1 with
`KER-KMA-004 --> No such alias: <uuid>`. It has no restart policy, so it stays down, and the
symptom looks like a slow start rather than a crash.

The only fix is to regenerate both stores together:

```bash
cd esignet/docker-compose && docker compose down -v && cd -
./demo.sh setup
```

That costs the test citizen and the registered client, which `setup` recreates. WSO2 Console config
is untouched. `./demo.sh start` detects this case and prints the same instructions instead of
waiting out its timeout.

## Known gaps to state openly during the demo

Full detail in runbook §18; the short version:

- The bridge asserts identity to IS on its own authority — right for a demo, wrong for production.
  The production answer is a custom federated authenticator JAR.
- **Logout is one-sided.** eSignet exposes no logout or `end_session` endpoint at all.
- eSignet's `sub` is a pairwise pseudonymous identifier — never present it as a national ID number.
- eSignet's "SSO" is one credential across services, not a browser session. WSO2 IS owns the
  session.
