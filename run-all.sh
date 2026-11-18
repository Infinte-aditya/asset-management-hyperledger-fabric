#!/usr/bin/env bash
#
# run-all.sh — one entrypoint for the Container Ledger demo.
#
# Brings up the Fabric test network, deploys/upgrades the chaincode, starts the
# GUI backend (and frontend), optionally runs the GPS-file demo, and cleans up
# leftover processes/ports on exit.
#
# Usage:
#   ./run-all.sh                 # network + chaincode + backend + frontend
#   ./run-all.sh --demo          # ...and ingest container_gps_simulation.json
#   ./run-all.sh --no-frontend   # headless: network + chaincode + backend only
#   ./run-all.sh --fresh         # tear the network down first (clean ledger)
#   ./run-all.sh --down          # stop everything (servers + network)
#   ./run-all.sh --help
#
# Environment overrides: CHANNEL_NAME, CHAINCODE_NAME, CC_VERSION, PORT, FRONTEND_PORT
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$SCRIPT_DIR"
TEST_NETWORK="$ROOT/fabric-samples/test-network"
FABRIC_BIN="$ROOT/fabric-samples/bin"
ORG1="$TEST_NETWORK/organizations/peerOrganizations/org1.example.com"
TLS_ROOT="$ORG1/peers/peer0.org1.example.com/tls/ca.crt"
ADMIN_MSP="$ORG1/users/Admin@org1.example.com/msp"

CHANNEL="${CHANNEL_NAME:-mychannel}"
CC_NAME="${CHAINCODE_NAME:-container-cc}"
CC_VERSION="${CC_VERSION:-2.0}"
CC_PATH="../asset-transfer-basic/chaincode-go"
BACKEND_PORT="${PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-5173}"
RUN_DIR="$ROOT/.run"
DEMO_CONTAINER="CONT-UHF-98231A"
GPS_FILE="$ROOT/container_gps_simulation.json"

MODE="up"
RUN_DEMO=0
START_FRONTEND=1
FRESH=0

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mWARN:\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31mERROR:\033[0m %s\n' "$*" >&2; exit 1; }

usage() {
  sed -n '2,18p' "$0" | sed 's/^# \{0,1\}//'
}

while [ $# -gt 0 ]; do
  case "$1" in
    --demo)                  RUN_DEMO=1 ;;
    --no-frontend|--headless) START_FRONTEND=0 ;;
    --fresh)                 FRESH=1 ;;
    --down)                  MODE="down" ;;
    -h|--help)               usage; exit 0 ;;
    *) die "unknown option: $1 (try --help)" ;;
  esac
  shift
done

# ---------------------------------------------------------------------------
# Process / port cleanup
# ---------------------------------------------------------------------------
port_pids() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    lsof -ti "tcp:$port" 2>/dev/null || true
  elif command -v fuser >/dev/null 2>&1; then
    fuser "$port/tcp" 2>/dev/null | tr -s ' ' '\n' || true
  else
    ss -ltnp 2>/dev/null | awk -v p=":$port" '$4 ~ p {print $NF}' \
      | sed -n 's/.*pid=\([0-9]*\).*/\1/p' || true
  fi
}

free_port() {
  local port="$1" pids
  pids="$(port_pids "$port")"
  if [ -n "$pids" ]; then
    log "freeing port $port (pid(s): $(echo "$pids" | tr '\n' ' '))"
    kill -TERM $pids 2>/dev/null || true
    sleep 1
    pids="$(port_pids "$port")"
    if [ -n "$pids" ]; then
      kill -KILL $pids 2>/dev/null || true
    fi
  fi
}

STARTED_PIDS=()
cleanup() {
  trap - EXIT INT TERM
  echo
  log "shutting down started processes"
  for pid in "${STARTED_PIDS[@]:-}"; do
    [ -n "$pid" ] || continue
    pkill -TERM -P "$pid" 2>/dev/null || true
    kill -TERM "$pid" 2>/dev/null || true
  done
  sleep 1
  for pid in "${STARTED_PIDS[@]:-}"; do
    [ -n "$pid" ] || continue
    kill -KILL "$pid" 2>/dev/null || true
  done
  free_port "$BACKEND_PORT"
  free_port "$FRONTEND_PORT"
  log "done. (Fabric network left as-is; use --down to stop it.)"
}
trap cleanup EXIT INT TERM

# ---------------------------------------------------------------------------
# Fabric helpers
# ---------------------------------------------------------------------------
peer_cmd() {
  (
    cd "$TEST_NETWORK"
    export PATH="$FABRIC_BIN:$PATH"
    export FABRIC_CFG_PATH="$ROOT/fabric-samples/config"
    export CORE_PEER_TLS_ENABLED=true
    export CORE_PEER_LOCALMSPID="Org1MSP"
    export CORE_PEER_ADDRESS="localhost:7051"
    export CORE_PEER_TLS_ROOTCERT_FILE="$TLS_ROOT"
    export CORE_PEER_MSPCONFIGPATH="$ADMIN_MSP"
    "$@"
  )
}

network_up() {
  docker ps --filter "name=^peer0.org1.example.com$" --filter "status=running" -q | grep -q .
}

cc_committed_output() {
  peer_cmd peer lifecycle chaincode querycommitted -C "$CHANNEL" --name "$CC_NAME" \
    --peerAddresses localhost:7051 --tlsRootCertFiles "$TLS_ROOT" 2>/dev/null
}
cc_committed_version()  { cc_committed_output | sed -n "/Version:/{s/.*Version: //; s/, Sequence:.*$//; p;}"  | head -1 | tr -d ' '; }
cc_committed_sequence() { cc_committed_output | sed -n "/Version:/{s/.*Sequence: //; s/, Endorsement Plugin:.*$//; p;}" | head -1 | tr -d ' '; }

deploy_chaincode() {
  local seq="$1"
  # Sync source-of-truth chaincode into the fabric-samples copy (no --delete:
  # that would wipe the sample's vendor/ and mocks/).
  rsync -a "$ROOT/chaincode/asset-transfer-basic/chaincode-go/" \
           "$ROOT/fabric-samples/asset-transfer-basic/chaincode-go/"
  ( cd "$TEST_NETWORK" && ./network.sh deployCC -ccn "$CC_NAME" -ccp "$CC_PATH" -ccl go -ccv "$CC_VERSION" -ccs "$seq" )
}

# ---------------------------------------------------------------------------
# Down mode
# ---------------------------------------------------------------------------
if [ "$MODE" = "down" ]; then
  free_port "$BACKEND_PORT"
  free_port "$FRONTEND_PORT"
  if [ -d "$TEST_NETWORK" ]; then
    log "stopping Fabric test network"
    ( cd "$TEST_NETWORK" && ./network.sh down ) || warn "network.sh down reported an error"
  fi
  rm -rf "$RUN_DIR"
  log "everything stopped."
  trap - EXIT INT TERM
  exit 0
fi

# ---------------------------------------------------------------------------
# Prerequisites
# ---------------------------------------------------------------------------
command -v docker >/dev/null 2>&1 || die "docker not found"
docker info  >/dev/null 2>&1 || die "Docker is not running (start it, or: sudo usermod -aG docker \$USER then re-login)"
command -v go  >/dev/null 2>&1 || die "go not found"
command -v jq  >/dev/null 2>&1 || die "jq not found"
[ -d "$TEST_NETWORK" ] || die "test network not found at $TEST_NETWORK (run ./install-fabric.sh first)"
if [ "$START_FRONTEND" = 1 ]; then
  command -v npm >/dev/null 2>&1 || die "npm not found"
fi

mkdir -p "$RUN_DIR"

# Clear leftovers from a previous run before starting.
free_port "$BACKEND_PORT"
free_port "$FRONTEND_PORT"

# ---------------------------------------------------------------------------
# 1. Network
# ---------------------------------------------------------------------------
if [ "$FRESH" = 1 ] && [ -d "$TEST_NETWORK" ]; then
  log "fresh start: bringing the network down first"
  ( cd "$TEST_NETWORK" && ./network.sh down ) || warn "network.sh down reported an error"
fi

if network_up; then
  log "test network already running"
else
  log "starting Fabric test network (channel: $CHANNEL)"
  ( cd "$TEST_NETWORK" && ./network.sh up createChannel )
fi

# ---------------------------------------------------------------------------
# 2. Chaincode (version-aware)
# ---------------------------------------------------------------------------
if ! cc_committed_output >/dev/null 2>&1; then
  log "deploying chaincode '$CC_NAME' v$CC_VERSION (sequence 1)"
  deploy_chaincode 1
else
  committed_version="$(cc_committed_version)"
  if [ "$committed_version" = "$CC_VERSION" ]; then
    log "chaincode '$CC_NAME' v$CC_VERSION already deployed on '$CHANNEL'"
  else
    next_seq=$(( $(cc_committed_sequence) + 1 ))
    log "upgrading chaincode '$CC_NAME' ${committed_version:-unknown} -> v$CC_VERSION (sequence $next_seq)"
    deploy_chaincode "$next_seq"
  fi
fi

# ---------------------------------------------------------------------------
# 3. Backend
# ---------------------------------------------------------------------------
log "building backend"
( cd "$ROOT/asset-management-gui/server" && go build -o "$RUN_DIR/backend-bin" . )

log "starting backend on :$BACKEND_PORT (log: $RUN_DIR/backend.log)"
( cd "$ROOT/asset-management-gui/server" && exec env PORT="$BACKEND_PORT" "$RUN_DIR/backend-bin" ) >"$RUN_DIR/backend.log" 2>&1 &
BACKEND_PID=$!
echo "$BACKEND_PID" >"$RUN_DIR/backend.pid"
STARTED_PIDS+=("$BACKEND_PID")

backend_ok=0
for _ in $(seq 1 40); do
  if curl -sf "http://localhost:$BACKEND_PORT/api/health" >/dev/null 2>&1; then backend_ok=1; break; fi
  sleep 1
done
if [ "$backend_ok" != 1 ]; then
  warn "backend did not become healthy; last log lines:"
  tail -n 20 "$RUN_DIR/backend.log" >&2 || true
  die "backend failed to start"
fi
log "backend healthy: $(curl -s "http://localhost:$BACKEND_PORT/api/health")"

# ---------------------------------------------------------------------------
# 4. Frontend
# ---------------------------------------------------------------------------
if [ "$START_FRONTEND" = 1 ]; then
  if [ ! -x "$ROOT/asset-management-gui/web/node_modules/.bin/vite" ]; then
    log "installing frontend dependencies"
    ( cd "$ROOT/asset-management-gui/web" && npm install )
  fi
  log "starting frontend on :$FRONTEND_PORT (log: $RUN_DIR/frontend.log)"
  ( cd "$ROOT/asset-management-gui/web" && exec ./node_modules/.bin/vite --host --port "$FRONTEND_PORT" --strictPort ) >"$RUN_DIR/frontend.log" 2>&1 &
  FRONTEND_PID=$!
  echo "$FRONTEND_PID" >"$RUN_DIR/frontend.pid"
  STARTED_PIDS+=("$FRONTEND_PID")

  frontend_ok=0
  for _ in $(seq 1 40); do
    if curl -sf "http://localhost:$FRONTEND_PORT/" >/dev/null 2>&1; then frontend_ok=1; break; fi
    sleep 1
  done
  if [ "$frontend_ok" != 1 ]; then
    warn "frontend did not become ready; see $RUN_DIR/frontend.log"
  fi
fi

# ---------------------------------------------------------------------------
# 5. Optional GPS demo
# ---------------------------------------------------------------------------
if [ "$RUN_DEMO" = 1 ]; then
  if [ ! -f "$GPS_FILE" ]; then
    warn "GPS file not found: $GPS_FILE (skipping demo)"
  else
    log "GPS demo: ensuring container '$DEMO_CONTAINER' exists and is Loaded"
    (
      cd "$ROOT/application-gateway-go"
      go run . ReadContainer "$DEMO_CONTAINER" >/dev/null 2>&1 \
        || go run . RegisterContainer "$DEMO_CONTAINER" 40RH "Test Logistics" 3700 USLGB
      go run . ReportEvent "$DEMO_CONTAINER" Loaded CNSHA Depot-Ops "Loaded aboard vessel" >/dev/null
    )
    log "GPS demo: dry-run"
    ( cd "$ROOT/gps-ingestor" && go run . -file "$GPS_FILE" -dry-run ) || true
    log "GPS demo: submitting"
    ( cd "$ROOT/gps-ingestor" && go run . -file "$GPS_FILE" )
    count="$( cd "$ROOT/application-gateway-go" && go run . GetPositionHistory "$DEMO_CONTAINER" | grep -c '"Timestamp"' || true )"
    log "GPS demo: '$DEMO_CONTAINER' now has $count fixes on the ledger"
  fi
fi

# ---------------------------------------------------------------------------
# 6. Ready
# ---------------------------------------------------------------------------
echo
log "READY"
echo "  API:        http://localhost:$BACKEND_PORT"
echo "  API health: http://localhost:$BACKEND_PORT/api/health"
if [ "$START_FRONTEND" = 1 ]; then
  echo "  UI (map):   http://localhost:$FRONTEND_PORT"
fi
echo "  logs:       $RUN_DIR/"
echo
log "Press Ctrl+C to stop the app processes (the Fabric network keeps running; use './run-all.sh --down' to stop it)."
wait
