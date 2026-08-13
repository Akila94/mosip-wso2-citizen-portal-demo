#!/usr/bin/env bash
#
# demo.sh — one script to bootstrap, start, stop and check the
#           MOSIP eSignet <-> WSO2 Identity Server federation demo.
#
# It automates every step of esignet-wso2-is-federation-runbook.md that can be
# automated. The steps that remain are WSO2 Console work; they are listed in
# MANUAL-STEPS.md and this script tells you when to go do them.
#
#   ./demo.sh setup       one-time bootstrap (runbook Steps 1-6)
#   ./demo.sh start       start eSignet, WSO2 IS and the bridge
#   ./demo.sh stop        stop all three, keeping all data
#   ./demo.sh restart     stop then start
#   ./demo.sh status      what is up, what is down
#   ./demo.sh preflight   the runbook Step 15 pre-demo check
#   ./demo.sh logs [svc]  tail bridge | is | esignet
#   ./demo.sh apikey      print the bridge API key (needed by the Console)
#   ./demo.sh clean       delete logs, pid files and the IS download cache
#   ./demo.sh clean --all delete downloads, node_modules, keys, .env and volumes
#   ./demo.sh reset-wso2  delete IS config+users, forcing a redo of MANUAL-STEPS.md
#
set -euo pipefail

REPO="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$REPO"

BRIDGE_DIR="$REPO/esignet-bridge"
ESIGNET_DIR="$REPO/esignet"
COMPOSE_DIR="$ESIGNET_DIR/docker-compose"
IS_DIR="$REPO/wso2is-7.3.0"
RUN_DIR="$REPO/.run"
ENV_FILE="$REPO/.env"

IS_VERSION="7.3.0"
IS_ZIP="wso2is-${IS_VERSION}.zip"
IS_URL="https://github.com/wso2/product-is/releases/download/v${IS_VERSION}/${IS_ZIP}"
ESIGNET_BRANCH="v1.8.0"
ESIGNET_GIT="https://github.com/mosip/esignet.git"

# Endpoints — must stay in step with the bridge's own defaults in server.js.
ESIGNET_API="${ESIGNET_API:-http://localhost:8088/v1/esignet}"
ESIGNET_UI="${ESIGNET_UI:-http://localhost:3000}"
MOCK_ID="${MOCK_ID:-http://localhost:8082/v1/mock-identity-system}"
BRIDGE="${BRIDGE:-http://localhost:4000}"
IS_URL_BASE="${IS_BASE_URL:-https://localhost:9443}"
CLIENT_ID="${CLIENT_ID:-wso2-is-bridge}"

INDIVIDUAL_ID="8267411571"

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
is_up()     { [ "$(http_code "$1")" = "${2:-200}" ]; }

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

# Node 22 is the prerequisite: it is what the bridge is run and verified on.
need_node22() {
  need node
  local major
  major="$(node -p 'process.versions.node.split(".")[0]' 2>/dev/null || echo 0)"
  [ "$major" = "22" ] && { ok "node $(node -v)"; return 0; }
  [ "$major" -lt 22 ] 2>/dev/null \
    && die "node $(node -v) is too old — use Node 22 (nvm: 'nvm use 22')"
  warn "node $(node -v) — the demo is built for Node 22; switch if the bridge misbehaves"
}

# .env is DATA, never sourced: sourcing would execute whatever it contains as
# shell code. Read the one key we care about and nothing else.
load_env() {
  [ -f "$ENV_FILE" ] || return 0
  local line
  line="$(grep -m1 '^BRIDGE_API_KEY=' "$ENV_FILE" 2>/dev/null || true)"
  [ -n "$line" ] || return 0
  BRIDGE_API_KEY="${line#BRIDGE_API_KEY=}"
  # Tolerate quoted values written by hand.
  BRIDGE_API_KEY="${BRIDGE_API_KEY%\"}"; BRIDGE_API_KEY="${BRIDGE_API_KEY#\"}"
  BRIDGE_API_KEY="${BRIDGE_API_KEY%\'}"; BRIDGE_API_KEY="${BRIDGE_API_KEY#\'}"
  export BRIDGE_API_KEY
}

# .env holds a secret: owner-read/write only, whether we just created it or not.
harden_env_file() {
  [ -f "$ENV_FILE" ] || return 0
  chmod 600 "$ENV_FILE" 2>/dev/null || warn "could not chmod 600 $ENV_FILE"
}

# The bridge API key is generated once and kept in .env (gitignored) so that
# every later start, preflight and Console lookup uses the same value.
ensure_api_key() {
  load_env
  if [ -z "${BRIDGE_API_KEY:-}" ]; then
    need openssl
    BRIDGE_API_KEY="$(openssl rand -base64 32)"
    # umask scoped to this subshell so it does not leak into the rest of the run.
    ( umask 077; printf 'BRIDGE_API_KEY=%s\n' "$BRIDGE_API_KEY" >> "$ENV_FILE" )
    export BRIDGE_API_KEY
    say "generated a bridge API key and stored it in .env"
  fi
  harden_env_file
}

# Pass a secret header to curl through a 600 config file instead of argv, which
# is world-readable via ps while the request is in flight.
curl_with_api_key() {   # curl_with_api_key <curl-args...>
  local cfg rc
  cfg="$(mktemp "${TMPDIR:-/tmp}/demo-curl.XXXXXX")"
  chmod 600 "$cfg"
  printf 'header = "x-api-key: %s"\n' "$BRIDGE_API_KEY" > "$cfg"
  curl --config "$cfg" "$@"
  rc=$?
  rm -f "$cfg"
  return $rc
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

# =============================================================== setup ======
cmd_setup() {
  say "checking prerequisites"
  need docker; need curl; need python3; need git; need unzip; need_node22
  docker compose version >/dev/null 2>&1 || die "docker compose v2 is required"
  docker info >/dev/null 2>&1 || die "the Docker daemon is not running"
  ok "prerequisites present"

  mkdir -p "$RUN_DIR"

  # ---- runbook Step 1: eSignet source (only docker-compose/ and postman are used)
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

  # ---- runbook Step 5: WSO2 IS
  # The completeness marker is wso2server.sh, not the directory: an interrupted
  # unzip leaves a directory that looks finished but is not.
  if [ -f "$IS_DIR/bin/wso2server.sh" ]; then
    ok "WSO2 IS $IS_VERSION already unpacked"
  else
    [ -d "$IS_DIR" ] && { warn "removing a partially unpacked $IS_DIR"; rm -rf "$IS_DIR"; }
    # Download to .part and rename only on success, so an interrupted transfer
    # is never mistaken for a complete zip on the next run.
    if [ -f "$REPO/$IS_ZIP" ] && ! unzip -tq "$REPO/$IS_ZIP" >/dev/null 2>&1; then
      warn "$IS_ZIP is corrupt or truncated — downloading again"
      rm -f "$REPO/$IS_ZIP"
    fi
    if [ ! -f "$REPO/$IS_ZIP" ]; then
      say "downloading WSO2 IS $IS_VERSION (~730 MB)"
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

  # ---- runbook Step 11.2: session timeout, applied before IS ever starts
  patch_session_timeout

  # ---- runbook Step 3: bridge dependencies and keypair
  say "installing bridge dependencies"
  (cd "$BRIDGE_DIR" && npm install --silent)
  # Dependency-vulnerability gate: report, do not silently proceed. Upstream
  # advisories are not ours to fix, so this warns rather than aborting setup.
  if (cd "$BRIDGE_DIR" && npm audit --audit-level=high >"$RUN_DIR/npm-audit.out" 2>&1); then
    ok "npm audit: no high or critical advisories"
  else
    warn "npm audit reports high/critical advisories in the bridge dependencies:"
    sanitize < "$RUN_DIR/npm-audit.out" | tail -c 500; echo
    warn "review before any non-demo use — see 'npm audit' in $RUN_DIR/npm-audit.out"
  fi
  if [ -f "$BRIDGE_DIR/private.jwk.json" ] && [ -f "$BRIDGE_DIR/public.jwk.json" ]; then
    ok "keypair already present (not regenerating — it is bound to the eSignet client)"
  else
    say "generating the RS256 client keypair"
    (cd "$BRIDGE_DIR" && node genKeys.js >/dev/null)
    ok "wrote esignet-bridge/private.jwk.json and public.jwk.json"
  fi

  ensure_api_key

  # eSignet must be running before the citizen and client can be created.
  start_esignet

  # ---- runbook Step 2 and Step 4
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
  printf 'The Console asks for the bridge API key: %s./demo.sh apikey%s\n\n' "$B" "$N"
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
  say "creating the test citizen $INDIVIDUAL_ID (runbook Step 2)"
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
  say "registering the OIDC client '$CLIENT_ID' with eSignet (runbook Step 4)"
  local jar csrf pubkey resp
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
  pubkey="$(cat "$BRIDGE_DIR/public.jwk.json")"
  resp=$(curl -s --max-time 20 -b "$jar" -X POST "$ESIGNET_API/client-mgmt/client" \
    -H 'Content-Type: application/json' -H "X-XSRF-TOKEN: $csrf" -d "{
    \"requestTime\": \"$(date -u +%Y-%m-%dT%H:%M:%S.000Z)\",
    \"request\": {
      \"clientId\": \"$CLIENT_ID\",
      \"clientName\": \"WSO2 IS Bridge\",
      \"relyingPartyId\": \"mock-relying-party-id\",
      \"publicKey\": $pubkey,
      \"logoUri\": \"https://wso2.com/favicon.ico\",
      \"userClaims\": [\"name\",\"email\",\"gender\",\"phone_number\",\"birthdate\",\"picture\"],
      \"authContextRefs\": [\"mosip:idp:acr:generated-code\",\"mosip:idp:acr:password\"],
      \"redirectUris\": [\"$BRIDGE/callback\"],
      \"grantTypes\": [\"authorization_code\"],
      \"clientAuthMethods\": [\"private_key_jwt\"],
      \"additionalConfig\": { \"userinfo_response_type\": \"JWS\" }
    }}")
  rm -f "$jar"
  if printf '%s' "$resp" | grep -q '"status":"ACTIVE"'; then
    ok "client $CLIENT_ID registered and ACTIVE"
  elif printf '%s' "$resp" | grep -qi 'duplicate\|already'; then
    warn "client already registered — leaving it alone"
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
  mkdir -p "$RUN_DIR"
  # wso2server.sh start daemonises and writes to repository/logs/wso2carbon.log
  (cd "$IS_DIR" && ./bin/wso2server.sh start >"$RUN_DIR/is-start.out" 2>&1)
  wait_for "WSO2 IS console" "$IS_URL_BASE/console" 200 300 \
    || die "WSO2 IS did not come up — check './demo.sh logs is'"
}

start_bridge() {
  say "starting the bridge"
  [ -f "$BRIDGE_DIR/private.jwk.json" ] || die "esignet-bridge/private.jwk.json missing — run './demo.sh setup'"
  [ -d "$BRIDGE_DIR/node_modules" ] || die "bridge dependencies missing — run './demo.sh setup'"
  if is_up "$BRIDGE/health"; then ok "bridge already running"; return 0; fi
  need_node22
  ensure_api_key
  mkdir -p "$RUN_DIR"
  # Runs from the bridge directory because server.js reads private.jwk.json from cwd.
  (cd "$BRIDGE_DIR" && BRIDGE_API_KEY="$BRIDGE_API_KEY" nohup node server.js \
     >"$RUN_DIR/bridge.log" 2>&1 & echo $! > "$RUN_DIR/bridge.pid")
  wait_for "bridge /health" "$BRIDGE/health" 200 30 \
    || die "bridge did not start — check './demo.sh logs bridge'"
}

cmd_start() {
  start_esignet
  start_is
  start_bridge
  echo
  cmd_status
  printf '\nIf this is a fresh WSO2 IS, do the Console steps in %sMANUAL-STEPS.md%s next.\n' "$B" "$N"
}

# ================================================================ stop ======
# Only ever signal a process we can confirm is this bridge: a recycled PID from
# a stale pidfile would otherwise kill something unrelated.
is_our_bridge() {
  local pid="$1" cmd
  case "$pid" in ''|*[!0-9]*) return 1 ;; esac      # numeric PIDs only
  kill -0 "$pid" 2>/dev/null || return 1
  cmd="$(ps -o command= -p "$pid" 2>/dev/null || true)"
  case "$cmd" in *node*server.js*) return 0 ;; *) return 1 ;; esac
}

stop_bridge() {
  say "stopping the bridge"
  local pid="" candidate
  [ -f "$RUN_DIR/bridge.pid" ] && pid="$(head -c 32 "$RUN_DIR/bridge.pid" | tr -dc '0-9')"
  if is_our_bridge "$pid"; then
    kill "$pid" 2>/dev/null || true
  elif command -v lsof >/dev/null 2>&1; then
    # Fall back to whatever holds port 4000 (e.g. started by hand in a terminal),
    # still confirming each candidate is a node server.js before signalling it.
    for candidate in $(lsof -ti tcp:4000 2>/dev/null || true); do
      if is_our_bridge "$candidate"; then
        kill "$candidate" 2>/dev/null || true
      else
        warn "PID $candidate holds port 4000 but is not this bridge — leaving it alone"
      fi
    done
  fi
  rm -f "$RUN_DIR/bridge.pid"
  sleep 1
  is_up "$BRIDGE/health" && warn "something is still answering on $BRIDGE" || ok "bridge stopped"
}

stop_is() {
  say "stopping WSO2 IS"
  if [ -x "$IS_DIR/bin/wso2server.sh" ]; then
    detect_java_home
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
  stop_bridge
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
  is_up "$BRIDGE/health"        && ok "bridge            $BRIDGE"      || bad "bridge            $BRIDGE"
  is_up "$IS_URL_BASE/console"  && ok "WSO2 IS console   $IS_URL_BASE/console" || bad "WSO2 IS console   $IS_URL_BASE/console"
}

# =========================================================== preflight ======
# Runbook Step 15, implemented here so there is only one script to maintain.
cmd_preflight() {
  ensure_api_key
  local pass=0 fail=0
  chk() { # chk <label> <url> <code>
    if is_up "$2" "$3"; then ok "$1"; pass=$((pass + 1))
    else bad "$1 (HTTP $(http_code "$2"), want $3)"; fail=$((fail + 1)); fi
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

  say "Bridge"
  chk "bridge /health" "$BRIDGE/health" 200
  local resp
  resp=$(curl_with_api_key -s --max-time 8 -X POST "$BRIDGE/authenticate" \
    -H 'Content-Type: application/json' \
    -d '{"actionType":"AUTHENTICATION","flowId":"PREFLIGHT","requestId":"P","event":{"tenant":{"id":"-1234","name":"carbon.super"},"application":{"id":"a","name":"Preflight"},"currentStepIndex":1},"allowedOperations":[{"op":"redirect"}]}')
  if printf '%s' "$resp" | grep -q '"actionStatus":"INCOMPLETE"'
  then ok "returns INCOMPLETE + redirect"; pass=$((pass + 1))
  else bad "bad response: $(printf '%s' "$resp" | head -c 200 | sanitize)"; fail=$((fail + 1)); fi
  if printf '%s' "$resp" | grep -q "client_id=$CLIENT_ID"
  then ok "redirect carries client_id"; pass=$((pass + 1))
  else bad "wrong or missing client_id"; fail=$((fail + 1)); fi

  say "Keys"
  if (cd "$BRIDGE_DIR" && python3 -c "
import json,sys
a=json.load(open('private.jwk.json')); b=json.load(open('public.jwk.json'))
sys.exit(0 if a.get('kid')==b.get('kid') and a.get('n')==b.get('n') else 1)")
  then ok "keypair consistent"; pass=$((pass + 1))
  else bad "kid or modulus mismatch"; fail=$((fail + 1)); fi

  say "WSO2 IS"
  chk "console reachable" "$IS_URL_BASE/console" 200

  printf '\npassed=%s failed=%s\n' "$pass" "$fail"
  if [ "$fail" -eq 0 ]; then
    # Clear the PREFLIGHT flow entry and leave a clean log for the demo.
    say "restarting the bridge so the PREFLIGHT entry is cleared"
    stop_bridge >/dev/null; start_bridge >/dev/null
    ok "bridge restarted — ready to present"
  else
    die "fix the failures above before presenting (see runbook §17)"
  fi
}

# ================================================================ misc ======
cmd_logs() {
  case "${1:-bridge}" in
    bridge)  tail -f "$RUN_DIR/bridge.log" ;;
    is)      tail -f "$IS_DIR/repository/logs/wso2carbon.log" ;;
    esignet) compose logs -f esignet ;;
    *)       die "unknown log target '$1' (bridge | is | esignet)" ;;
  esac
}

cmd_apikey() {
  load_env
  [ -n "${BRIDGE_API_KEY:-}" ] || die "no API key yet — run './demo.sh setup'"
  # Printing a secret puts it in scrollback and shell history. Warn on a TTY,
  # and stay pipe-clean so './demo.sh apikey | pbcopy' still works.
  [ -t 1 ] && printf '%s(this is a secret: it is now in your scrollback)%s\n' "$Y" "$N" >&2
  printf '%s\n' "$BRIDGE_API_KEY"
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
    *)      die "unknown option '$1' (use --all to also delete downloads, keys and data)" ;;
  esac

  if [ "$all" -eq 0 ]; then
    say "removing transient files (logs, pid files, cookie jars, the IS download cache)"
    show_target "$RUN_DIR"
    show_target "$REPO/$IS_ZIP"
    show_target "$IS_DIR/repository/logs"
    show_target "$BRIDGE_DIR/cookies.txt"
    if is_up "$BRIDGE/health"; then
      warn "bridge is running — keeping .run/ (stop it first to clear the log)"
    else
      rm -rf "$RUN_DIR"
    fi
    rm -f  "$BRIDGE_DIR/cookies.txt" "$REPO/$IS_ZIP.part"
    # Only drop the zip once IS is actually unpacked, otherwise setup re-downloads it.
    [ -f "$IS_DIR/bin/wso2server.sh" ] && rm -f "$REPO/$IS_ZIP"
    # Never unlink wso2carbon.log from under a running IS — the process keeps
    # writing to a deleted inode and './demo.sh logs is' goes blind.
    if is_up "$IS_URL_BASE/console"; then
      warn "WSO2 IS is running — keeping its logs (stop it first to clear them)"
    else
      rm -f "$IS_DIR"/repository/logs/*.log 2>/dev/null || true
    fi
    ok "transient files removed — keys, .env, containers, volumes and Console config untouched"
    printf '\nTo go all the way back to a fresh clone: %s./demo.sh clean --all%s\n\n' "$B" "$N"
    return 0
  fi

  say "this deletes everything regenerable, returning the repo to a fresh clone"
  printf '  to be deleted:\n'
  show_target "$ESIGNET_DIR"
  show_target "$IS_DIR"
  show_target "$REPO/$IS_ZIP"
  show_target "$BRIDGE_DIR/node_modules"
  show_target "$BRIDGE_DIR/private.jwk.json"
  show_target "$BRIDGE_DIR/public.jwk.json"
  show_target "$ENV_FILE"
  show_target "$RUN_DIR"
  printf '  plus the Docker volumes: the test citizen AND the registered OIDC client.\n'
  printf '  you will redo: ./demo.sh setup and every step in MANUAL-STEPS.md\n'
  printf '\nType YES to continue: '
  read -r answer
  [ "$answer" = "YES" ] || die "aborted — nothing deleted"

  mkdir -p "$RUN_DIR"          # the stop/compose steps below log into it
  stop_bridge || true
  stop_is || true
  if [ -d "$COMPOSE_DIR" ]; then
    say "removing containers and volumes"
    # Report failures rather than hiding them: a volume that survives here is
    # what produces the KER-KMA-004 keystore/database mismatch later.
    compose down -v >"$RUN_DIR/compose-down.out" 2>&1 \
      || { bad "docker compose down -v failed"; sanitize < "$RUN_DIR/compose-down.out" | tail -c 400; echo; }
  fi
  say "deleting files"
  rm -rf "$ESIGNET_DIR" "$IS_DIR" "$BRIDGE_DIR/node_modules" "$RUN_DIR"
  rm -f  "$REPO/$IS_ZIP" "$REPO/$IS_ZIP.part" "$ENV_FILE" \
         "$BRIDGE_DIR/private.jwk.json" "$BRIDGE_DIR/public.jwk.json" \
         "$BRIDGE_DIR/cookies.txt"
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
}

# Print the header comment block (everything from line 3 up to the first non-comment).
usage() { awk 'NR>2 { if (!/^#/) exit; sub(/^# ?/, ""); print }' "${BASH_SOURCE[0]}"; }

case "${1:-}" in
  setup)      cmd_setup ;;
  start)      cmd_start ;;
  stop)       cmd_stop ;;
  restart)    cmd_restart ;;
  status)     cmd_status ;;
  preflight)  cmd_preflight ;;
  logs)       shift; cmd_logs "${1:-bridge}" ;;
  clean)      shift; cmd_clean "${1:-}" ;;
  apikey)     cmd_apikey ;;
  reset-wso2) cmd_reset_wso2 ;;
  ""|-h|--help|help) usage ;;
  *)          printf 'unknown command: %s\n\n' "$1"; usage; exit 1 ;;
esac
