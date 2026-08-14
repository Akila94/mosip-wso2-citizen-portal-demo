#!/usr/bin/env bash
#
# demo.sh — bootstrap, start, stop and check the MOSIP eSignet <-> WSO2 Identity
#           Server federation demo, in the bridge-free variant: IS talks OIDC to
#           eSignet directly through a custom federated authenticator JAR.
#
# It automates every step of ../esignet-bridge/esignet-wso2-is-federation-runbook.md
# that can be automated, with the deltas listed in RUNBOOK-DELTA.md. What remains is WSO2
# Console work; it is in MANUAL-STEPS.md and this script tells you when to do it.
#
#   ./demo.sh setup       one-time bootstrap (eSignet, IS, JAR, citizen, client)
#   ./demo.sh build       rebuild the authenticator JAR and deploy it to dropins
#   ./demo.sh start       start eSignet and WSO2 IS
#   ./demo.sh stop        stop both, keeping all data
#   ./demo.sh restart     stop then start
#   ./demo.sh status      what is up, what is down
#   ./demo.sh preflight   the pre-demo check
#   ./demo.sh jwk         print the public JWK registered with eSignet
#   ./demo.sh logs [svc]  tail is | esignet
#   ./demo.sh clean       delete logs and the IS download cache
#   ./demo.sh clean --all delete downloads, build output, IS, eSignet and volumes
#   ./demo.sh reset-wso2  delete IS config+users, forcing a redo of MANUAL-STEPS.md
#
set -euo pipefail

REPO="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$REPO"

CONNECTOR_DIR="$REPO/esignet-oidc-authenticator"
CONNECTOR_MODULE="$CONNECTOR_DIR/components/org.wso2.carbon.identity.application.authenticator.esignet"
ESIGNET_DIR="$REPO/esignet"
COMPOSE_DIR="$ESIGNET_DIR/docker-compose"
IS_DIR="$REPO/wso2is-7.3.0"
RUN_DIR="$REPO/.run"

IS_VERSION="7.3.0"
IS_ZIP="wso2is-${IS_VERSION}.zip"
IS_URL="https://github.com/wso2/product-is/releases/download/v${IS_VERSION}/${IS_ZIP}"
ESIGNET_BRANCH="v1.8.0"
ESIGNET_GIT="https://github.com/mosip/esignet.git"

# Fully-qualified name of the standalone JWK exporter shipped in the connector JAR.
JWK_EXPORTER_CLASS="org.wso2.carbon.identity.application.authenticator.esignet.tools.JwkExporter"

# Endpoints. eSignet has TWO base URLs and they are not interchangeable:
# ESIGNET_UI (:3000) serves /authorize, the browser-facing hop; ESIGNET_API
# (:8088/v1/esignet) serves /oauth/v2/token, /oidc/userinfo and the JWKS.
ESIGNET_API="${ESIGNET_API:-http://localhost:8088/v1/esignet}"
ESIGNET_UI="${ESIGNET_UI:-http://localhost:3000}"
MOCK_ID="${MOCK_ID:-http://localhost:8082/v1/mock-identity-system}"
IS_URL_BASE="${IS_BASE_URL:-https://localhost:9443}"
CLIENT_ID="${CLIENT_ID:-wso2-is-esignet}"
# Where eSignet sends the browser back to: IS's own commonauth endpoint. No bridge.
IS_CALLBACK="$IS_URL_BASE/commonauth"

INDIVIDUAL_ID="8267411571"

# # Console/REST admin credentials. Overridable, defaulting to the shipped pack values.
# IS_ADMIN_USER="${IS_ADMIN_USER:-admin}"
# IS_ADMIN_PASSWORD="${IS_ADMIN_PASSWORD:-admin}"

# # The connection ('identity provider') the login flow selects, and the authenticator
# # inside it. IS addresses a federated authenticator by base64 of its getName() value:
# #   printf 'EsignetOIDCAuthenticator' | base64  ->  RXNpZ25ldE9JRENBdXRoZW50aWNhdG9y
# CONNECTION_NAME="${CONNECTION_NAME:-MOSIP eSignet}"
# AUTHENTICATOR_ID="RXNpZ25ldE9JRENBdXRoZW50aWNhdG9y"

# Set by a setup step that could not finish but must not abort the rest of setup.
SETUP_BLOCKER=""

# ---------------------------------------------------------------- output ----
if [ -t 1 ]; then
  R=$'\033[31m'; G=$'\033[32m'; Y=$'\033[33m'; B=$'\033[1m'; N=$'\033[0m'
else
  R=''; G=''; Y=''; B=''; N=''
fi
say()  { printf '%s==>%s %s\n' "$B" "$N" "$*"; }
ok()   { printf '  %sOK%s    %s\n' "$G" "$N" "$*"; }
warn() { printf '  %sWARN%s  %s\n' "$Y" "$N" "$*"; }
bad()  { printf '  %sFAIL%s  %s\n' "$R" "$N" "$*"; }
die()  { printf '%serror:%s %s\n' "$R" "$N" "$*" >&2; exit 1; }

# Strip control characters from anything that came off the network, out of a
# container log, or off disk before printing it. Without this, escape sequences
# in a remote response can rewrite the terminal or forge output lines.
sanitize() { LC_ALL=C tr -d '\000-\010\013\014\016-\037\177' | LC_ALL=C tr -s '[:space:]' ' '; }

# ---------------------------------------------------------------- helpers ----
# TLS verification is disabled ONLY for WSO2 IS, which ships a self-signed
# certificate. Every other endpoint is verified normally.
http_code() {
  local insecure="" code
  case "$1" in "$IS_URL_BASE"*) insecure="--insecure" ;; esac
  # curl already writes 000 on a transport failure; do not append a second one.
  code="$(curl -s ${insecure:+"$insecure"} -o /dev/null -w '%{http_code}' \
            --max-time 8 "$1" 2>/dev/null || true)"
  case "$code" in ''|*[!0-9]*) code=000 ;; esac
  printf '%s' "$code"
}
is_up() { [ "$(http_code "$1")" = "${2:-200}" ]; }

# wait_for <label> <url> <expected-code> <timeout-seconds>
wait_for() {
  local label="$1" url="$2" want="$3" timeout="$4" waited=0
  printf '  waiting for %s ' "$label"
  while [ "$waited" -lt "$timeout" ]; do
    if is_up "$url" "$want"; then printf ' up (%ss)\n' "$waited"; return 0; fi
    printf '.'; sleep 3; waited=$((waited + 3))
  done
  printf ' timed out after %ss\n' "$timeout"
  return 1
}

need() { command -v "$1" >/dev/null 2>&1 || die "$1 is required but not installed"; }

# The connector is compiled for Java 11 but builds and runs on 11, 17 or 21 —
# the JDKs WSO2 IS 7.3.0 supports.
need_jdk() {
  detect_java_home
  local major
  major="$("$JAVA_HOME/bin/java" -version 2>&1 | head -1 \
            | sed -n 's/.*version "\([0-9][0-9]*\).*/\1/p')"
  case "$major" in
    11|17|21) ok "JDK $major ($JAVA_HOME)" ;;
    "")       warn "could not determine the JDK version of $JAVA_HOME" ;;
    *)        warn "JDK $major is outside the 11/17/21 range WSO2 IS 7.3.0 supports" ;;
  esac
}

detect_java_home() {
  [ -n "${JAVA_HOME:-}" ] && return 0
  if [ "$(uname -s)" = "Darwin" ] && [ -x /usr/libexec/java_home ]; then
    JAVA_HOME="$(/usr/libexec/java_home 2>/dev/null || true)"
    [ -n "$JAVA_HOME" ] && export JAVA_HOME && return 0
  fi
  die "JAVA_HOME is not set and could not be detected; WSO2 IS needs a JDK (11-21)"
}

compose() { (cd "$COMPOSE_DIR" && docker compose --file docker-compose.yml "$@"); }

# ------------------------------------------------------------- keystore ----
# The client assertion is signed with IS's own OAuth signing key, so there is no
# key file and no shared secret anywhere in this design. These three values are
# read from deployment.toml when it configures them, and otherwise fall back to
# the shipped defaults of the primary keystore.
keystore_conf() {   # keystore_conf <deployment.toml-key> <default>
  local toml="$IS_DIR/repository/conf/deployment.toml" value=""
  if [ -f "$toml" ]; then
    value="$(awk -v key="$1" '
      /^\[keystore\.primary\]/ { inside = 1; next }
      /^\[/                    { inside = 0 }
      inside && $0 ~ "^[[:space:]]*" key "[[:space:]]*=" {
        sub(/^[^=]*=[[:space:]]*/, ""); gsub(/"/, ""); print; exit
      }' "$toml" 2>/dev/null || true)"
  fi
  printf '%s' "${value:-$2}"
}
keystore_file()     { printf '%s/repository/resources/security/%s' \
                        "$IS_DIR" "$(keystore_conf file_name wso2carbon.jks)"; }
keystore_alias()    { keystore_conf alias wso2carbon; }
keystore_password() { keystore_conf password wso2carbon; }

connector_jar() {
  # Newest built artifact, so a version bump does not need a change here.
  ls -t "$CONNECTOR_MODULE"/target/org.wso2.carbon.identity.application.authenticator.esignet-*.jar \
    2>/dev/null | head -1
}

# Print the public JWK of the IS signing key. This is what gets registered with
# eSignet as the client's publicKey, and its kid is what the connector puts in
# the client assertion header.
export_jwk() {
  local jar keystore
  jar="$(connector_jar)"
  [ -n "$jar" ] || die "the connector JAR is not built — run './demo.sh build'"
  keystore="$(keystore_file)"
  [ -f "$keystore" ] || die "keystore not found at $keystore — run './demo.sh setup'"
  detect_java_home
  # The password goes through the environment, never argv: command lines are
  # world-readable via ps.
  ESIGNET_KEYSTORE_PASSWORD="$(keystore_password)" \
    "$JAVA_HOME/bin/java" -cp "$jar" "$JWK_EXPORTER_CLASS" "$keystore" "$(keystore_alias)"
}

# =============================================================== build ======
cmd_build() {
  say "building the eSignet federated authenticator"
  need mvn; need_jdk
  (cd "$CONNECTOR_DIR" && mvn -B -q clean install) || die "the Maven build failed"
  local jar
  jar="$(connector_jar)"
  [ -n "$jar" ] || die "the build produced no JAR under $CONNECTOR_MODULE/target"
  ok "built ${jar#"$REPO"/}"

  if [ ! -d "$IS_DIR/repository/components/dropins" ]; then
    warn "WSO2 IS is not unpacked yet — run './demo.sh setup' to deploy the JAR"
    return 0
  fi
  # Remove older copies first: two versions of the same bundle in dropins makes
  # OSGi pick one at random.
  rm -f "$IS_DIR"/repository/components/dropins/org.wso2.carbon.identity.application.authenticator.esignet-*.jar
  cp "$jar" "$IS_DIR/repository/components/dropins/"
  ok "deployed to wso2is-$IS_VERSION/repository/components/dropins/"
  if is_up "$IS_URL_BASE/console"; then
    warn "WSO2 IS is running — restart it for the new JAR to take effect ('./demo.sh restart')"
  fi
}

# =============================================================== setup ======
cmd_setup() {
  say "checking prerequisites"
  need docker; need curl; need python3; need git; need unzip; need mvn; need_jdk
  docker compose version >/dev/null 2>&1 || die "docker compose v2 is required"
  docker info >/dev/null 2>&1 || die "the Docker daemon is not running"
  ok "prerequisites present"

  mkdir -p "$RUN_DIR"

  # ---- eSignet source (only docker-compose/ and postman-collection/ are used)
  if [ -d "$ESIGNET_DIR/.git" ]; then
    ok "eSignet clone already present"
  elif [ -e "$ESIGNET_DIR" ]; then
    # A clone killed part-way can leave a directory with no .git in it.
    die "$ESIGNET_DIR exists but is not a git clone — remove it and re-run setup"
  else
    say "cloning eSignet $ESIGNET_BRANCH"
    git clone --depth 1 --branch "$ESIGNET_BRANCH" "$ESIGNET_GIT" "$ESIGNET_DIR" \
      || { rm -rf "$ESIGNET_DIR"; die "clone failed — re-run setup"; }
  fi

  # ---- WSO2 IS
  # The completeness marker is wso2server.sh, not the directory: an interrupted
  # unzip leaves a directory that looks finished but is not.
  if [ -f "$IS_DIR/bin/wso2server.sh" ]; then
    ok "WSO2 IS $IS_VERSION already unpacked"
  else
    [ -d "$IS_DIR" ] && { warn "removing a partially unpacked $IS_DIR"; rm -rf "$IS_DIR"; }
    if [ -f "$REPO/$IS_ZIP" ] && ! unzip -tq "$REPO/$IS_ZIP" >/dev/null 2>&1; then
      warn "$IS_ZIP is corrupt or truncated — downloading again"
      rm -f "$REPO/$IS_ZIP"
    fi
    if [ ! -f "$REPO/$IS_ZIP" ]; then
      say "downloading WSO2 IS $IS_VERSION (~730 MB)"
      # Download to .part and rename only on success, so an interrupted transfer
      # is never mistaken for a complete zip on the next run.
      curl -fL --progress-bar -o "$REPO/$IS_ZIP.part" "$IS_URL" \
        || { rm -f "$REPO/$IS_ZIP.part"; die "download failed — re-run setup to resume"; }
      unzip -tq "$REPO/$IS_ZIP.part" >/dev/null 2>&1 \
        || { rm -f "$REPO/$IS_ZIP.part"; die "downloaded zip is not readable — re-run setup"; }
      mv "$REPO/$IS_ZIP.part" "$REPO/$IS_ZIP"
    fi
    say "unpacking $IS_ZIP"
    unzip -q "$REPO/$IS_ZIP" -d "$REPO" || { rm -rf "$IS_DIR"; die "unzip failed — re-run setup"; }
  fi
  chmod +x "$IS_DIR"/bin/*.sh 2>/dev/null || true

  # ---- session timeout, applied before IS ever starts
  patch_session_timeout

  # ---- the authenticator JAR
  cmd_build

  # eSignet must be running before the citizen and the client can be created.
  start_esignet

  create_citizen
  register_client

  if [ -n "$SETUP_BLOCKER" ]; then
    printf '\n%ssetup finished, but one step needs you:%s %s\n' "$Y" "$N" "$SETUP_BLOCKER"
    printf 'Everything else is done. Fix that one thing, then re-run %s./demo.sh setup%s\n' "$B" "$N"
    printf '(it skips all completed work) or continue straight to %s./demo.sh start%s.\n\n' "$B" "$N"
    return 1
  fi

  say "setup complete"
  printf '\nNext: %s./demo.sh start%s, then work through %sMANUAL-STEPS.md%s (WSO2 Console).\n' \
    "$B" "$N" "$B" "$N"
  printf 'The client id to enter there is %s%s%s.\n\n' "$B" "$CLIENT_ID" "$N"
}

patch_session_timeout() {
  local toml="$IS_DIR/repository/conf/deployment.toml"
  [ -f "$toml" ] || { warn "deployment.toml not found; skipping session-timeout patch"; return 0; }
  if grep -q '^\[session\.timeout\]' "$toml"; then
    ok "session timeout already configured"
    return 0
  fi
  cp "$toml" "$toml.bak-demo"
  cat >> "$toml" <<'TOML'

[session.timeout]
idle_session_timeout = "60m"
remember_me_session_timeout = "14d"
TOML
  ok "raised the SSO idle timeout to 60m (backup at deployment.toml.bak-demo)"
}

create_citizen() {
  say "creating the test citizen $INDIVIDUAL_ID"
  local photo resp
  photo="data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/wAALCAABAAEBAREA/8QAFAABAAAAAAAAAAAAAAAAAAAACf/EABQQAQAAAAAAAAAAAAAAAAAAAAD/2gAIAQEAAD8AKp//2Q=="
  resp=$(curl -s --max-time 20 -X POST "$MOCK_ID/identity" \
    -H 'Content-Type: application/json' -d "{
    \"requestTime\": \"$(date -u +%Y-%m-%dT%H:%M:%S.000Z)\",
    \"request\": {
      \"individualId\": \"$INDIVIDUAL_ID\",
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
      \"encodedPhoto\": \"$photo\",
      \"preferredLang\": \"eng\",
      \"locale\": \"en\",
      \"zoneInfo\": \"test zone\"
    }}")
  if printf '%s' "$resp" | grep -qi 'error'; then
    # A duplicate is the expected outcome on every run after the first.
    warn "identity not created — probably already exists. Response:"
    printf '        %s\n' "$(printf '%s' "$resp" | head -c 300 | sanitize)"
  else
    ok "citizen $INDIVIDUAL_ID created (OTP 111111)"
  fi
}

register_client() {
  say "registering the OIDC client '$CLIENT_ID' with eSignet"
  local jar csrf pubkey resp
  pubkey="$(export_jwk)" || die "could not export the IS public JWK"
  # Kept in .run/ rather than /tmp so it is cleaned up even if curl aborts.
  jar="$RUN_DIR/cookies.txt"
  rm -f "$jar"
  # The jar holds a CSRF token: create it owner-only.
  ( umask 077; : > "$jar" )
  curl -s -c "$jar" -o /dev/null --max-time 15 "$ESIGNET_API/csrf/token" || true
  csrf="$(awk '$6=="XSRF-TOKEN"{print $7}' "$jar" 2>/dev/null | tail -1)"
  if [ -z "$csrf" ]; then
    rm -f "$jar"
    bad "could not read the XSRF-TOKEN cookie"
    SETUP_BLOCKER="the OIDC client is NOT registered — use the Postman fallback in MANUAL-STEPS.md"
    return 0
  fi
  # publicKey is the JWK of the IS signing key; redirectUris is IS's commonauth
  # endpoint, because with no bridge in the flow eSignet returns straight to IS.
  resp=$(curl -s --max-time 20 -b "$jar" -X POST "$ESIGNET_API/client-mgmt/client" \
    -H 'Content-Type: application/json' -H "X-XSRF-TOKEN: $csrf" -d "{
    \"requestTime\": \"$(date -u +%Y-%m-%dT%H:%M:%S.000Z)\",
    \"request\": {
      \"clientId\": \"$CLIENT_ID\",
      \"clientName\": \"WSO2 Identity Server\",
      \"relyingPartyId\": \"mock-relying-party-id\",
      \"publicKey\": $pubkey,
      \"logoUri\": \"https://wso2.com/favicon.ico\",
      \"userClaims\": [\"name\",\"email\",\"gender\",\"phone_number\",\"birthdate\",\"picture\"],
      \"authContextRefs\": [\"mosip:idp:acr:generated-code\",\"mosip:idp:acr:password\"],
      \"redirectUris\": [\"$IS_CALLBACK\"],
      \"grantTypes\": [\"authorization_code\"],
      \"clientAuthMethods\": [\"private_key_jwt\"],
      \"additionalConfig\": { \"userinfo_response_type\": \"JWS\" }
    }}")
  rm -f "$jar"
  if printf '%s' "$resp" | grep -q '"status":"ACTIVE"'; then
    ok "client $CLIENT_ID registered and ACTIVE"
  elif printf '%s' "$resp" | grep -qi 'duplicate\|already'; then
    warn "client already registered — leaving it alone"
    warn "if you have regenerated the IS keystore since, the stored publicKey is stale:"
    warn "re-register the client with a new clientId, or reset eSignet ('./demo.sh clean --all')"
  else
    bad "registration did not report ACTIVE. Response:"
    printf '        %s\n' "$(printf '%s' "$resp" | head -c 400 | sanitize)"
    SETUP_BLOCKER="the OIDC client is NOT registered — see the Postman fallback in MANUAL-STEPS.md"
  fi
}

# =============================================================== start ======
svc_exited() { compose ps -a --status exited --services 2>/dev/null | grep -qx "$1"; }

# Explain a crashed container instead of leaving the caller to guess.
diagnose_esignet() {
  local svc="$1" log
  log="$(compose logs --tail=200 "$svc" 2>&1 || true)"
  bad "the '$svc' container exited"
  if printf '%s' "$log" | grep -q 'KER-KMA-004'; then
    cat <<'EOM'
        Cause: the Postgres data and eSignet's PKCS12 keystore have diverged.
        The database still holds a ROOT key alias, but the keystore that held the
        matching private key lived inside the container filesystem and was lost
        when the container was recreated ("No such alias").

        Fix — recreate both together. This DESTROYS the test citizen and the
        registered OIDC client, so re-run setup afterwards to recreate them:

          cd esignet/docker-compose && docker compose down -v && cd -
          ./demo.sh setup
EOM
  else
    printf '        last error from the container:\n'
    printf '%s\n' "$log" | grep -oE '"message":"[^"]{0,240}' | tail -3 \
      | while IFS= read -r l; do printf '        %s\n' "$(printf '%s' "$l" | sanitize)"; done
    printf '        full log: ./demo.sh logs %s\n' "$svc"
  fi
}

start_esignet() {
  say "starting the eSignet stack"
  [ -d "$COMPOSE_DIR" ] || die "$COMPOSE_DIR is missing — run './demo.sh setup' first"
  compose up -d
  # First run pulls several GB, so allow a long window — but stop early if a
  # container has died, rather than waiting out the whole timeout.
  local waited=0 timeout=600
  printf '  waiting for eSignet discovery '
  while [ "$waited" -lt "$timeout" ]; do
    if is_up "$ESIGNET_API/oidc/.well-known/openid-configuration"; then
      printf ' up (%ss)\n' "$waited"
      break
    fi
    for svc in esignet mock-identity-system database; do
      if svc_exited "$svc"; then
        printf '\n'
        diagnose_esignet "$svc"
        die "eSignet stack failed to start"
      fi
    done
    printf '.'; sleep 3; waited=$((waited + 3))
  done
  is_up "$ESIGNET_API/oidc/.well-known/openid-configuration" \
    || die "eSignet did not come up within ${timeout}s — check './demo.sh logs esignet'"
  is_up "$ESIGNET_UI/" && ok "eSignet UI serving on $ESIGNET_UI" || warn "eSignet UI not answering yet"
  is_up "$MOCK_ID/actuator/health" && ok "mock identity system up" || warn "mock identity system not answering yet"
}

start_is() {
  say "starting WSO2 IS"
  [ -x "$IS_DIR/bin/wso2server.sh" ] || die "$IS_DIR is missing — run './demo.sh setup' first"
  detect_java_home
  if is_up "$IS_URL_BASE/console"; then ok "WSO2 IS already running"; return 0; fi
  [ -n "$(ls "$IS_DIR"/repository/components/dropins/org.wso2.carbon.identity.application.authenticator.esignet-*.jar 2>/dev/null)" ] \
    || warn "no eSignet authenticator JAR in dropins — run './demo.sh build'"
  mkdir -p "$RUN_DIR"
  # wso2server.sh start daemonises and writes to repository/logs/wso2carbon.log
  (cd "$IS_DIR" && ./bin/wso2server.sh start >"$RUN_DIR/is-start.out" 2>&1)
  wait_for "WSO2 IS console" "$IS_URL_BASE/console" 200 300 \
    || die "WSO2 IS did not come up — check './demo.sh logs is'"
}

cmd_start() {
  start_esignet
  start_is
  echo
  cmd_status
  printf '\nIf this is a fresh WSO2 IS, do the Console steps in %sMANUAL-STEPS.md%s next.\n' "$B" "$N"
}

# ================================================================ stop ======
stop_is() {
  say "stopping WSO2 IS"
  if [ -x "$IS_DIR/bin/wso2server.sh" ]; then
    detect_java_home
    mkdir -p "$RUN_DIR"
    (cd "$IS_DIR" && ./bin/wso2server.sh stop >"$RUN_DIR/is-stop.out" 2>&1) || true
    local waited=0
    while [ "$waited" -lt 60 ] && is_up "$IS_URL_BASE/console"; do sleep 3; waited=$((waited + 3)); done
  fi
  is_up "$IS_URL_BASE/console" && warn "WSO2 IS is still answering on $IS_URL_BASE" || ok "WSO2 IS stopped"
}

stop_esignet() {
  say "stopping the eSignet stack"
  # 'stop', never 'down -v': the test citizen lives in the Postgres volume.
  if [ -d "$COMPOSE_DIR" ]; then compose stop >/dev/null 2>&1 || true; fi
  ok "eSignet containers stopped (volumes kept)"
}

cmd_stop() {
  stop_is
  stop_esignet
  say "all services stopped — data preserved"
}

cmd_restart() { cmd_stop; echo; cmd_start; }

# ============================================================== status ======
cmd_status() {
  say "status"
  is_up "$ESIGNET_API/oidc/.well-known/openid-configuration" \
    && ok "eSignet service   $ESIGNET_API" || bad "eSignet service   $ESIGNET_API"
  is_up "$ESIGNET_UI/"          && ok "eSignet UI        $ESIGNET_UI"   || bad "eSignet UI        $ESIGNET_UI"
  is_up "$MOCK_ID/actuator/health" && ok "mock identity     $MOCK_ID" || bad "mock identity     $MOCK_ID"
  is_up "$IS_URL_BASE/console"  && ok "WSO2 IS console   $IS_URL_BASE/console" || bad "WSO2 IS console   $IS_URL_BASE/console"
  if [ -n "$(ls "$IS_DIR"/repository/components/dropins/org.wso2.carbon.identity.application.authenticator.esignet-*.jar 2>/dev/null)" ]
  then ok "authenticator JAR deployed in dropins"
  else bad "authenticator JAR not in dropins (./demo.sh build)"; fi
}

# =========================================================== preflight ======
cmd_preflight() {
  local pass=0 fail=0
  chk() { # chk <label> <url> <code>
    if is_up "$2" "$3"; then ok "$1"; pass=$((pass + 1))
    else bad "$1 (HTTP $(http_code "$2"), want $3)"; fail=$((fail + 1)); fi
  }
  yes_no() { # yes_no <label> <0-or-1>
    if [ "$2" -eq 0 ]; then ok "$1"; pass=$((pass + 1)); else bad "$1"; fail=$((fail + 1)); fi
  }

  say "eSignet"
  chk "discovery reachable" "$ESIGNET_API/oidc/.well-known/openid-configuration" 200
  chk "JWKS reachable"      "$ESIGNET_API/oauth/.well-known/jwks.json" 200
  chk "eSignet UI serving"  "$ESIGNET_UI/" 200
  chk "mock identity up"    "$MOCK_ID/actuator/health" 200
  if curl -s --max-time 8 "$ESIGNET_API/oidc/.well-known/openid-configuration" \
       | grep -q private_key_jwt
  then ok "advertises private_key_jwt"; pass=$((pass + 1))
  else bad "private_key_jwt not advertised"; fail=$((fail + 1)); fi

  say "Connector"
  local jar deployed
  jar="$(connector_jar)"
  yes_no "authenticator JAR built" "$([ -n "$jar" ] && echo 0 || echo 1)"
  deployed="$(ls "$IS_DIR"/repository/components/dropins/org.wso2.carbon.identity.application.authenticator.esignet-*.jar 2>/dev/null | head -1)"
  yes_no "authenticator JAR in dropins" "$([ -n "$deployed" ] && echo 0 || echo 1)"
  if [ -n "$deployed" ] && [ -n "$jar" ] && cmp -s "$jar" "$deployed"; then
    ok "deployed JAR matches the build output"; pass=$((pass + 1))
  else
    bad "deployed JAR differs from the build output (./demo.sh build)"; fail=$((fail + 1))
  fi
  if grep -q 'MOSIP eSignet federated authenticator bundle is activated' \
       "$IS_DIR/repository/logs/wso2carbon.log" 2>/dev/null
  then ok "authenticator bundle activated in this IS run"; pass=$((pass + 1))
  else bad "no activation line in wso2carbon.log — the bundle did not start"; fail=$((fail + 1)); fi

  say "Signing key"
  # The kid eSignet has on file must be the kid the connector will send. Both come
  # from the same keystore entry, so this checks the keystore is still readable
  # and reports the kid for comparison against the registered client.
  local jwk kid
  if jwk="$(export_jwk 2>/dev/null)"; then
    kid="$(printf '%s' "$jwk" | python3 -c 'import json,sys; print(json.load(sys.stdin)["kid"])' 2>/dev/null || true)"
    if [ -n "$kid" ]; then ok "IS signing key exports as a JWK (kid $kid)"; pass=$((pass + 1))
    else bad "JWK export produced unparseable output"; fail=$((fail + 1)); fi
  else
    bad "could not export the IS signing key as a JWK"; fail=$((fail + 1))
  fi

  say "WSO2 IS"
  chk "console reachable" "$IS_URL_BASE/console" 200
  # commonauth is where eSignet sends the browser back; a 302 or 200 both mean
  # the endpoint is live, so only a transport failure counts as down.
  if [ "$(http_code "$IS_CALLBACK")" != "000" ]
  then ok "commonauth endpoint answering"; pass=$((pass + 1))
  else bad "commonauth endpoint not answering at $IS_CALLBACK"; fail=$((fail + 1)); fi

  printf '\npassed=%s failed=%s\n' "$pass" "$fail"
  [ "$fail" -eq 0 ] || die "fix the failures above before presenting (see RUNBOOK-DELTA.md)"
  say "ready to present"
  printf 'Remaining manual check: the Console lists %sMOSIP eSignet%s under the connection,\n' "$B" "$N"
  printf 'and the client id registered with eSignet is %s%s%s.\n' "$B" "$CLIENT_ID" "$N"
}

# ================================================================ misc ======
cmd_logs() {
  case "${1:-is}" in
    is)      tail -f "$IS_DIR/repository/logs/wso2carbon.log" ;;
    esignet) compose logs -f esignet ;;
    *)       die "unknown log target '$1' (is | esignet)" ;;
  esac
}

# Show what a path is and how big, so nothing is deleted sight unseen.
show_target() {
  [ -e "$1" ] || return 0
  printf '    %-46s %s\n' "${1#"$REPO"/}" "$(du -sh "$1" 2>/dev/null | cut -f1)"
}

cmd_clean() {
  local all=0
  case "${1:-}" in
    "")     all=0 ;;
    --all)  all=1 ;;
    *)      die "unknown option '$1' (use --all to also delete downloads, IS and data)" ;;
  esac

  if [ "$all" -eq 0 ]; then
    say "removing transient files (logs, cookie jars, the IS download cache)"
    show_target "$RUN_DIR"
    show_target "$REPO/$IS_ZIP"
    show_target "$IS_DIR/repository/logs"
    rm -rf "$RUN_DIR"
    rm -f "$REPO/$IS_ZIP.part"
    # Only drop the zip once IS is actually unpacked, otherwise setup re-downloads it.
    [ -f "$IS_DIR/bin/wso2server.sh" ] && rm -f "$REPO/$IS_ZIP"
    # Never unlink wso2carbon.log from under a running IS — the process keeps
    # writing to a deleted inode and './demo.sh logs is' goes blind.
    if is_up "$IS_URL_BASE/console"; then
      warn "WSO2 IS is running — keeping its logs (stop it first to clear them)"
    else
      rm -f "$IS_DIR"/repository/logs/*.log 2>/dev/null || true
    fi
    ok "transient files removed — build output, containers, volumes and Console config untouched"
    printf '\nTo go all the way back to a fresh clone: %s./demo.sh clean --all%s\n\n' "$B" "$N"
    return 0
  fi

  say "this deletes everything regenerable, returning the folder to a fresh clone"
  printf '  to be deleted:\n'
  show_target "$ESIGNET_DIR"
  show_target "$IS_DIR"
  show_target "$REPO/$IS_ZIP"
  show_target "$CONNECTOR_MODULE/target"
  show_target "$RUN_DIR"
  printf '  plus the Docker volumes: the test citizen AND the registered OIDC client.\n'
  printf '  you will redo: ./demo.sh setup and every step in MANUAL-STEPS.md\n'
  printf '\nType YES to continue: '
  read -r answer
  [ "$answer" = "YES" ] || die "aborted — nothing deleted"

  mkdir -p "$RUN_DIR"          # the stop/compose steps below log into it
  stop_is || true
  if [ -d "$COMPOSE_DIR" ]; then
    say "removing containers and volumes"
    # Report failures rather than hiding them: a volume that survives here is
    # what produces the KER-KMA-004 keystore/database mismatch later.
    compose down -v >"$RUN_DIR/compose-down.out" 2>&1 \
      || { bad "docker compose down -v failed"; sanitize < "$RUN_DIR/compose-down.out" | tail -c 400; echo; }
  fi
  say "deleting files"
  rm -rf "$ESIGNET_DIR" "$IS_DIR" "$CONNECTOR_MODULE/target" "$RUN_DIR"
  rm -f  "$REPO/$IS_ZIP" "$REPO/$IS_ZIP.part"
  ok "clean — only the committed files remain"
  printf '\nStart over with: %s./demo.sh setup%s\n\n' "$B" "$N"
}

cmd_reset_wso2() {
  printf 'This deletes all WSO2 IS Console configuration and JIT-provisioned users.\n'
  printf 'You will have to redo every step in MANUAL-STEPS.md. Type YES to continue: '
  read -r answer
  [ "$answer" = "YES" ] || die "aborted"
  is_up "$IS_URL_BASE/console" && die "stop WSO2 IS first: ./demo.sh stop"
  rm -f "$IS_DIR"/repository/database/*.mv.db "$IS_DIR"/repository/database/*.trace.db
  ok "H2 databases removed — next start is a clean IS"
  warn "the keystore is untouched, so the client registered with eSignet stays valid"
}

cmd_jwk() { export_jwk; }

# Print the header comment block (everything from line 3 up to the first non-comment).
usage() { awk 'NR>2 { if (!/^#/) exit; sub(/^# ?/, ""); print }' "${BASH_SOURCE[0]}"; }

case "${1:-}" in
  setup)      cmd_setup ;;
  build)      cmd_build ;;
  start)      cmd_start ;;
  stop)       cmd_stop ;;
  restart)    cmd_restart ;;
  status)     cmd_status ;;
  preflight)  cmd_preflight ;;
  jwk)        cmd_jwk ;;
  logs)       shift; cmd_logs "${1:-is}" ;;
  clean)      shift; cmd_clean "${1:-}" ;;
  reset-wso2) cmd_reset_wso2 ;;
  ""|-h|--help|help) usage ;;
  *)          printf 'unknown command: %s\n\n' "$1"; usage; exit 1 ;;
esac
