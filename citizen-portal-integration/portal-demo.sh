#!/usr/bin/env bash
#
# portal-demo.sh — build, run and check citizen-portal-bff and gov-services-api,
#                  the M6 orchestration script for ../PORTAL-INTEGRATION-PLAN.md.
#
# It never starts WSO2 IS or eSignet — that is ../setup-without-bridge/demo.sh's
# job. This script only checks they are already up and points there if not.
#
#   ./portal-demo.sh setup       create both .env files from .env.example (idempotent)
#   ./portal-demo.sh build       go build both services; npm run build the SPA
#   ./portal-demo.sh start       start gov-services-api, then citizen-portal-bff
#   ./portal-demo.sh stop        stop both, keeping .env files and build output
#   ./portal-demo.sh restart     stop then start
#   ./portal-demo.sh status      what is up, what is down (IS, eSignet, both services)
#   ./portal-demo.sh preflight   the pre-demo check — curl-level, no browser
#   ./portal-demo.sh logs [svc]  tail bff | govapi
#   ./portal-demo.sh clean       delete build output and runtime state
#   ./portal-demo.sh clean --all also delete both .env files (asks first)
#
set -euo pipefail

REPO="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$REPO"

BFF_DIR="$REPO/citizen-portal-bff"
GOVAPI_DIR="$REPO/gov-services-api"
SPA_DIR="$REPO/../citizen-portal-demo-app"
RUN_DIR="$REPO/.run"
BIN_DIR="$REPO/bin"

BFF_ENV="$BFF_DIR/.env"
GOVAPI_ENV="$GOVAPI_DIR/.env"

# Endpoints this script only ever reads — it never starts any of these.
# setup-without-bridge/demo.sh owns WSO2 IS and eSignet; the two Go services
# below are the ones this script does own.
IS_URL_BASE="${IS_BASE_URL:-https://localhost:9443}"
ESIGNET_API="${ESIGNET_API:-http://localhost:8088/v1/esignet}"
ESIGNET_UI="${ESIGNET_UI:-http://localhost:3000}"
BFF_URL="${BFF_URL:-http://localhost:8090}"
GOVAPI_URL="${GOVAPI_URL:-http://localhost:8091}"
BFF_PORT="${BFF_PORT_HINT:-8090}"
GOVAPI_PORT="${GOVAPI_PORT_HINT:-8091}"

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

# Strip control characters from anything that came off the network or out of
# a log before printing it. Without this, escape sequences in a remote
# response could rewrite the terminal or forge output lines. Mirrors
# esignet-bridge/demo.sh and setup-without-bridge/demo.sh.
sanitize() { LC_ALL=C tr -d '\000-\010\013\014\016-\037\177' | LC_ALL=C tr -s '[:space:]' ' '; }

# ---------------------------------------------------------------- helpers ----
# TLS verification is disabled ONLY for WSO2 IS, which ships a self-signed
# certificate. Every other endpoint (eSignet, the two Go services) is
# verified normally — mirrors both existing demo.sh scripts' own invariant.
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
    printf '.'; sleep 2; waited=$((waited + 2))
  done
  printf ' timed out after %ss\n' "$timeout"
  return 1
}

need() { command -v "$1" >/dev/null 2>&1 || die "$1 is required but not installed"; }

# Vite 5 needs Node >= 18 (it calls a crypto API absent from older Node, which
# fails with an opaque "crypto$2.getRandomValues is not a function" rather
# than a version complaint); CLAUDE.md already flags Node 22 as this repo's
# baseline. Checked explicitly so the failure is legible instead of a stack
# trace three layers into vite's bundle.
need_node18() {
  need node
  local major
  major="$(node -p 'process.versions.node.split(".")[0]' 2>/dev/null || echo 0)"
  if [ "$major" -lt 18 ] 2>/dev/null; then
    die "node $(node -v) is too old for the SPA build (needs >= 18) — try 'nvm use 22'"
  fi
  ok "node $(node -v)"
}

# .env files are DATA, never sourced: sourcing would execute whatever they
# contain as shell code (the same finding esignet-bridge/demo.sh's review
# fixed). Read key=value pairs and export them, nothing else.
load_env_file() {   # load_env_file <path>
  [ -f "$1" ] || return 0
  local key value
  while IFS='=' read -r key value; do
    [[ "$key" =~ ^[[:space:]]*# ]] && continue
    [[ -z "$key" ]] && continue
    key="$(echo -n "$key" | xargs)"
    [[ -z "$key" ]] && continue
    export "$key=$value"
  done < <(grep -E '^[A-Za-z_][A-Za-z0-9_]*=' "$1")
}

# Both .env files hold secrets (client secrets). Owner-read/write only, on
# every run, whether they were just created or already existed.
harden_env_files() {
  if [ -f "$BFF_ENV" ]; then
    chmod 600 "$BFF_ENV" 2>/dev/null || warn "could not chmod 600 citizen-portal-bff/.env"
  fi
  if [ -f "$GOVAPI_ENV" ]; then
    chmod 600 "$GOVAPI_ENV" 2>/dev/null || warn "could not chmod 600 gov-services-api/.env"
  fi
}

# This script never brings up WSO2 IS or eSignet itself — that is
# ../setup-without-bridge/demo.sh's job. It only checks they are already
# reachable and tells the caller where to go if not.
check_is_and_esignet_up() {
  local all_up=1
  is_up "$IS_URL_BASE/console" \
    && ok "WSO2 IS console   $IS_URL_BASE/console" \
    || { bad "WSO2 IS console   $IS_URL_BASE/console"; all_up=0; }
  is_up "$ESIGNET_API/oidc/.well-known/openid-configuration" \
    && ok "eSignet service   $ESIGNET_API" \
    || { bad "eSignet service   $ESIGNET_API"; all_up=0; }
  is_up "$ESIGNET_UI/" \
    && ok "eSignet UI        $ESIGNET_UI" \
    || { bad "eSignet UI        $ESIGNET_UI"; all_up=0; }
  [ "$all_up" -eq 1 ]
}

# Only ever signal a process this script can confirm is the one it started:
# a recycled PID from a stale pidfile would otherwise kill something
# unrelated. Mirrors esignet-bridge/demo.sh's is_our_bridge().
is_our_process() {   # is_our_process <pid> <command-substring>
  local pid="$1" match="$2" cmd
  case "$pid" in ''|*[!0-9]*) return 1 ;; esac      # numeric PIDs only
  kill -0 "$pid" 2>/dev/null || return 1
  cmd="$(ps -o command= -p "$pid" 2>/dev/null || true)"
  case "$cmd" in *"$match"*) return 0 ;; *) return 1 ;; esac
}

# =============================================================== setup ======
cmd_setup() {
  say "checking prerequisites"
  need go
  need npm
  ok "go $(go version | awk '{print $3}')"
  ok "npm $(npm -v)"

  say "creating .env files (existing files are never overwritten)"
  if [ -f "$BFF_ENV" ]; then
    ok "citizen-portal-bff/.env already exists"
  else
    [ -f "$BFF_DIR/.env.example" ] || die "citizen-portal-bff/.env.example is missing"
    ( umask 077; cp "$BFF_DIR/.env.example" "$BFF_ENV" )
    ok "created citizen-portal-bff/.env from .env.example"
  fi
  if [ -f "$GOVAPI_ENV" ]; then
    ok "gov-services-api/.env already exists"
  else
    [ -f "$GOVAPI_DIR/.env.example" ] || die "gov-services-api/.env.example is missing"
    ( umask 077; cp "$GOVAPI_DIR/.env.example" "$GOVAPI_ENV" )
    ok "created gov-services-api/.env from .env.example"
  fi
  harden_env_files

  say "checking the six client credentials"
  load_env_file "$BFF_ENV"
  local missing=0 var
  for var in PORTAL_CLIENT_ID PORTAL_CLIENT_SECRET \
             DL_CLIENT_ID DL_CLIENT_SECRET \
             VRL_CLIENT_ID VRL_CLIENT_SECRET; do
    if [ -z "${!var:-}" ]; then
      warn "$var is not set in citizen-portal-bff/.env"
      missing=1
    fi
  done
  if [ "$missing" -eq 1 ]; then
    printf '\nRegister all three applications in the WSO2 IS Console and fill in both\n'
    printf '.env files — see %sMANUAL-STEPS.md%s §1-6.\n\n' "$B" "$N"
  else
    ok "all six client credentials are set"
  fi

  say "IS certificate"
  if [ -f "$REPO/certs/wso2is-local.pem" ]; then
    ok "certs/wso2is-local.pem present"
  else
    warn "certs/wso2is-local.pem missing — see certs/README.md to export it"
  fi
}

# =============================================================== build ======
cmd_build() {
  mkdir -p "$BIN_DIR"

  say "building citizen-portal-bff"
  need go
  ( cd "$BFF_DIR" && go build -o "$BIN_DIR/bff" ./cmd/bff )
  ok "bin/bff"

  say "building gov-services-api"
  ( cd "$GOVAPI_DIR" && go build -o "$BIN_DIR/govapi" ./cmd/govapi )
  ok "bin/govapi"

  say "building the SPA (static mode)"
  need_node18
  need npm
  ( cd "$SPA_DIR" && npm install && npm run build )
  ok "citizen-portal-demo-app/dist"
}

# ================================================================ start ======
start_govapi() {
  say "starting gov-services-api"
  [ -f "$GOVAPI_ENV" ] || die "gov-services-api/.env missing — run './portal-demo.sh setup'"
  [ -x "$BIN_DIR/govapi" ] || die "bin/govapi missing — run './portal-demo.sh build'"
  if is_up "$GOVAPI_URL/public/catalogue"; then ok "gov-services-api already running"; return 0; fi
  mkdir -p "$RUN_DIR"
  (
    cd "$GOVAPI_DIR"
    load_env_file "$GOVAPI_ENV"
    nohup "$BIN_DIR/govapi" >"$RUN_DIR/govapi.log" 2>&1 &
    echo $! > "$RUN_DIR/govapi.pid"
  )
  wait_for "gov-services-api" "$GOVAPI_URL/public/catalogue" 200 20 \
    || die "gov-services-api did not start — check './portal-demo.sh logs govapi'"
}

start_bff() {
  say "starting citizen-portal-bff"
  [ -f "$BFF_ENV" ] || die "citizen-portal-bff/.env missing — run './portal-demo.sh setup'"
  [ -x "$BIN_DIR/bff" ] || die "bin/bff missing — run './portal-demo.sh build'"
  load_env_file "$BFF_ENV"
  if [ -z "${DEV_PROXY_TARGET:-}" ] && [ ! -f "$SPA_DIR/dist/index.html" ]; then
    die "SPA not built and DEV_PROXY_TARGET not set — run './portal-demo.sh build', or set DEV_PROXY_TARGET in citizen-portal-bff/.env to proxy a running 'npm run dev'"
  fi
  if is_up "$BFF_URL/"; then ok "citizen-portal-bff already running"; return 0; fi
  mkdir -p "$RUN_DIR"
  (
    cd "$BFF_DIR"
    load_env_file "$BFF_ENV"
    nohup "$BIN_DIR/bff" >"$RUN_DIR/bff.log" 2>&1 &
    echo $! > "$RUN_DIR/bff.pid"
  )
  wait_for "citizen-portal-bff" "$BFF_URL/" 200 20 \
    || die "citizen-portal-bff did not start — check './portal-demo.sh logs bff'"
}

cmd_start() {
  say "checking WSO2 IS and eSignet"
  if ! check_is_and_esignet_up; then
    printf '\nBring both up first — this script never starts them:\n'
    printf '  cd ../setup-without-bridge && ./demo.sh start   (or ./demo.sh status to check)\n\n'
    die "WSO2 IS and/or eSignet are not reachable"
  fi
  start_govapi
  start_bff
  echo
  cmd_status
  printf '\nOpen %s%s%s — never the Vite dev server directly.\n' "$B" "$BFF_URL" "$N"
}

# ================================================================= stop ======
stop_process() {   # stop_process <label> <pidfile> <command-substring> <health-url> <port>
  local label="$1" pidfile="$2" match="$3" url="$4" port="$5" pid="" candidate
  say "stopping $label"
  [ -f "$pidfile" ] && pid="$(head -c 32 "$pidfile" | tr -dc '0-9')"
  if is_our_process "$pid" "$match"; then
    kill "$pid" 2>/dev/null || true
  elif command -v lsof >/dev/null 2>&1; then
    # Fall back to whatever holds the port (e.g. started by hand in a
    # terminal), still confirming each candidate before signalling it.
    for candidate in $(lsof -ti "tcp:$port" 2>/dev/null || true); do
      if is_our_process "$candidate" "$match"; then
        kill "$candidate" 2>/dev/null || true
      else
        warn "PID $candidate holds port $port but is not $label — leaving it alone"
      fi
    done
  fi
  rm -f "$pidfile"
  sleep 1
  is_up "$url" && warn "something is still answering on $url" || ok "$label stopped"
}

cmd_stop() {
  stop_process "citizen-portal-bff" "$RUN_DIR/bff.pid" "bin/bff" "$BFF_URL/" "$BFF_PORT"
  stop_process "gov-services-api" "$RUN_DIR/govapi.pid" "bin/govapi" "$GOVAPI_URL/public/catalogue" "$GOVAPI_PORT"
  say "citizen-portal-bff and gov-services-api stopped — WSO2 IS and eSignet untouched"
}

cmd_restart() { cmd_stop; echo; cmd_start; }

# =============================================================== status ======
cmd_status() {
  say "status"
  check_is_and_esignet_up || true
  is_up "$GOVAPI_URL/public/catalogue" \
    && ok "gov-services-api  $GOVAPI_URL" || bad "gov-services-api  $GOVAPI_URL"
  is_up "$BFF_URL/" \
    && ok "citizen-portal-bff $BFF_URL" || bad "citizen-portal-bff $BFF_URL"
}

# ============================================================ preflight ======
# The curl-level subset of PORTAL-INTEGRATION-PLAN.md's "Manual, end to end"
# checklist — everything that does not need a browser.
cmd_preflight() {
  local pass=0 fail=0
  chk() { # chk <label> <url> <code>
    if is_up "$2" "$3"; then ok "$1"; pass=$((pass + 1))
    else bad "$1 (HTTP $(http_code "$2"), want $3)"; fail=$((fail + 1)); fi
  }

  say "WSO2 IS and eSignet"
  chk "IS console reachable" "$IS_URL_BASE/console" 200
  chk "eSignet discovery"    "$ESIGNET_API/oidc/.well-known/openid-configuration" 200
  chk "eSignet UI serving"   "$ESIGNET_UI/" 200

  say "gov-services-api"
  chk "public catalogue (no token needed)" "$GOVAPI_URL/public/catalogue" 200
  local govcode
  govcode="$(http_code "$GOVAPI_URL/portal/catalogue")"
  if [ "$govcode" = "400" ] || [ "$govcode" = "401" ]; then
    ok "portal router rejects a request with no bearer token ($govcode)"; pass=$((pass + 1))
  else
    bad "portal router answered $govcode with no token, want 400 or 401"; fail=$((fail + 1))
  fi

  say "citizen-portal-bff"
  chk "serves the SPA"                      "$BFF_URL/" 200
  chk "deep link survives a hard refresh"   "$BFF_URL/apps/driving-licence/step/2" 200
  chk "unknown /bff path is JSON, not HTML" "$BFF_URL/bff/portal/nonexistent" 404
  chk "public catalogue via the BFF"        "$BFF_URL/bff/portal/public/catalogue" 200
  chk "authenticated route with no cookie"  "$BFF_URL/bff/portal/api/catalogue" 401

  say "privilege-escalation check"
  if diff <(curl -s --max-time 8 "$BFF_URL/bff/portal/public/catalogue") \
          <(curl -s --max-time 8 "$BFF_URL/bff/portal/public/catalogue?assuranceLevel=substantial") \
       >/dev/null 2>&1
  then ok "the public catalogue ignores a caller-supplied assuranceLevel"; pass=$((pass + 1))
  else bad "assuranceLevel changed the public catalogue response — privilege escalation"; fail=$((fail + 1))
  fi

  printf '\npassed=%s failed=%s\n' "$pass" "$fail"
  if [ "$fail" -eq 0 ]; then
    ok "ready to present"
  else
    die "fix the failures above before presenting (see PORTAL-INTEGRATION-PLAN.md's 'Manual, end to end' section, and MANUAL-STEPS.md)"
  fi
}

# ================================================================= misc ======
cmd_logs() {
  case "${1:-bff}" in
    bff)    tail -f "$RUN_DIR/bff.log" ;;
    govapi) tail -f "$RUN_DIR/govapi.log" ;;
    *)      die "unknown log target '$1' (bff | govapi)" ;;
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
    "")    all=0 ;;
    --all) all=1 ;;
    *)     die "unknown option '$1' (use --all to also delete both .env files)" ;;
  esac

  if [ "$all" -eq 0 ]; then
    say "removing build output and runtime state"
    show_target "$RUN_DIR"
    show_target "$BIN_DIR"
    show_target "$SPA_DIR/dist"
    if is_up "$BFF_URL/" || is_up "$GOVAPI_URL/public/catalogue"; then
      warn "a service is still running — stop it first: ./portal-demo.sh stop"
    else
      rm -rf "$RUN_DIR" "$BIN_DIR" "$SPA_DIR/dist"
    fi
    ok "build output and runtime state removed — .env files and node_modules untouched"
    printf '\nTo also delete both .env files: %s./portal-demo.sh clean --all%s\n\n' "$B" "$N"
    return 0
  fi

  say "this also deletes both .env files — you will redo MANUAL-STEPS.md §6"
  printf '  to be deleted:\n'
  show_target "$RUN_DIR"
  show_target "$BIN_DIR"
  show_target "$SPA_DIR/dist"
  show_target "$BFF_ENV"
  show_target "$GOVAPI_ENV"
  printf '\nType YES to continue: '
  read -r answer
  [ "$answer" = "YES" ] || die "aborted — nothing deleted"

  cmd_stop || true
  rm -rf "$RUN_DIR" "$BIN_DIR" "$SPA_DIR/dist"
  rm -f "$BFF_ENV" "$GOVAPI_ENV"
  ok "clean — only committed files remain"
  printf '\nStart over with: %s./portal-demo.sh setup%s\n\n' "$B" "$N"
}

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
  logs)       shift; cmd_logs "${1:-bff}" ;;
  clean)      shift; cmd_clean "${1:-}" ;;
  ""|-h|--help|help) usage ;;
  *)          printf 'unknown command: %s\n\n' "$1"; usage; exit 1 ;;
esac
