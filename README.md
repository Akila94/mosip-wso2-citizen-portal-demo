# MOSIP eSignet ↔ WSO2 Identity Server — federation & SSO demo

A **demo environment**, not a product. A citizen logs in with a national ID through
[MOSIP eSignet](https://github.com/mosip/esignet), WSO2 Identity Server 7.3.0 federates to it,
and that one login carries an SSO session across two applications.

Everything here runs locally: eSignet in Docker, IS from its release zip.

## Why there are two variants

The stock WSO2 "Standard-Based OIDC" connection **cannot** talk to eSignet. Two hard
incompatibilities, both confirmed in source:

1. eSignet authenticates clients at the token endpoint with `private_key_jwt` only —
   `client_assertion` is mandatory on its request DTO and no `client_secret` code path exists.
   WSO2's **outbound** OIDC connector implements only `client_secret_basic` and
   `client_secret_post`. (IS supports `private_key_jwt` fully *inbound*, for clients
   authenticating *to* IS — a separate implementation, in a separate repository.)
2. eSignet always returns UserInfo as a **signed JWT**; the outbound connector parses UserInfo
   as plain JSON. eSignet puts no user claims in the ID token, so skipping UserInfo is not a
   workaround.

Each folder closes those two gaps a different way, and both are demoable:

| Folder | Approach | Use it when |
|---|---|---|
| [`esignet-bridge/`](esignet-bridge/) | A small Node.js service implementing IS 7.1+'s **service-based custom authenticator** contract, translating between IS and eSignet's OIDC. No Java, no restart. | You want the quickest path, or to show the contract-based extension point. It is a translating middlebox that asserts identity to IS on its own authority — right for a demo, wrong for production. |
| [`setup-without-bridge/`](setup-without-bridge/) | A **custom federated authenticator JAR** extending the product's `OpenIDConnectAuthenticator`, overriding only `getAccessTokenRequest()` (private_key_jwt) and `getSubjectAttributes()` (signed UserInfo). | You want the production-correct answer. No extra process, no shared secret; the client assertion is signed with IS's own keystore key. |

The two are independent. Read the README in whichever folder you pick.

## Start here

```bash
cd esignet-bridge          # or: cd setup-without-bridge
./demo.sh setup            # clone eSignet, fetch IS, build, create test citizen, register client
./demo.sh start
./demo.sh preflight        # must end failed=0
```

Then work through that folder's **`MANUAL-STEPS.md`** — the WSO2 Console configuration, which no
script can do. `./demo.sh` with no arguments lists every subcommand (`status`, `stop`, `logs`,
`clean`, …).

First run downloads ~730 MB of IS plus several GB of Docker images; budget ~20 GB of disk.

**Prerequisites:** Docker with Compose v2, `git`, `curl`, `unzip`, `python3`, plus
Node 22 for `esignet-bridge/`, or JDK 11/17/21 and Maven for `setup-without-bridge/`.

**Only one variant can run at a time** — they use the same ports, and both drive the same
Docker Compose project, so `clean --all` or `docker compose down -v` in one destroys the other's
eSignet data too.

## Ports

| Component | URL |
|---|---|
| eSignet UI (serves `/authorize` — browser-facing) | `http://localhost:3000` |
| eSignet API (token, UserInfo, JWKS — server-to-server) | `http://localhost:8088/v1/esignet` |
| Mock identity system | `http://localhost:8082/v1/mock-identity-system` |
| WSO2 IS console (`admin` / `admin`) | `https://localhost:9443/console` |
| Bridge (`esignet-bridge/` only) | `http://localhost:4000` |

eSignet's two base URLs are **not** interchangeable; mixing them yields a 404 that looks like a
client-registration failure.

## Test citizen

Individual ID `8267411571`, OTP `111111` (hardcoded in the mock identity system). It lives in a
Postgres volume and survives restarts — `docker compose down -v` destroys it, and `./demo.sh
setup` recreates it.

## What is and is not committed

Only the files that cannot be regenerated: the two `demo.sh` scripts, the docs, the bridge's
`server.js`/`genKeys.js`, and the connector's Maven sources. Everything else —
`esignet/` (a `--depth 1` clone of v1.8.0), `wso2is-7.3.0/`, build output, keys, `.env`,
`.run/` — is produced by `./demo.sh setup` and is gitignored.

`esignet-bridge/esignet-wso2-is-federation-runbook.md` is the **authoritative spec**:
architecture, every setup step, troubleshooting, and an appendix of source-verified claims.
Where any README disagrees with it, the runbook wins.
`setup-without-bridge/RUNBOOK-DELTA.md` is the diff against it for the JAR variant.

## Limitations to state openly

- **Logout is one-sided.** Signing out of IS ends the IS session; eSignet exposes no logout or
  `end_session` endpoint at all, so nothing propagates to it.
- **eSignet's `sub` is a pairwise pseudonymous identifier** — a different opaque value per
  relying party. Never present or store it as a national ID number.
- **eSignet's "SSO" is not a browser session** — it is one credential across services. WSO2 IS
  owns the actual SSO session, which is the point of this demo.
