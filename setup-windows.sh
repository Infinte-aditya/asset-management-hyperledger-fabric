#!/usr/bin/env bash
#
# setup-windows.sh — complete one-shot setup for the Asset Management
# Hyperledger Fabric app on a Windows laptop.
#
# Run this from Git Bash, MSYS2, or WSL (e.g. Ubuntu on Windows).
#
#   git clone https://github.com/Infinte-aditya/asset-management-hyperledger-fabric.git
#   cd asset-management-hyperledger-fabric
#   chmod +x setup-windows.sh
#   ./setup-windows.sh           # full setup: fabric + network + chaincode + backend + frontend
#   ./setup-windows.sh --demo    # ...plus ingest the GPS demo file
#   ./setup-windows.sh --down    # stop everything (app processes + Fabric network)
#   ./setup-windows.sh --help
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$SCRIPT_DIR"
TEST_NETWORK="$ROOT/fabric-samples/test-network"
FABRIC_BIN="$ROOT/fabric-samples/bin"

BACKEND_PORT="${PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-5173}"

MODE="up"
RUN_DEMO=0
START_FRONTEND=1
FRESH=0

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mWARN:\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31mERROR:\033[0m %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<EOF
Usage: $0 [options]

Options:
  --demo          Also ingest container_gps_simulation.json (GPS demo)
  --no-frontend   Headless: network + chaincode + backend only
  --fresh         Tear the network down first (clean ledger)
  --down          Stop everything (app processes + Fabric network)
  -h, --help      Show this help

Environment overrides: PORT, FRONTEND_PORT, CHANNEL_NAME, CHAINCODE_NAME, CC_VERSION

Run from Git Bash, MSYS2, or WSL on Windows. Docker Desktop must be running.
EOF
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
# 0. Prerequisites
# ---------------------------------------------------------------------------
log "checking prerequisites"

command -v git   >/dev/null 2>&1 || die "git not found. Install Git for Windows."
command -v go    >/dev/null 2>&1 || die "go not found. Install Go from https://go.dev/dl/"
command -v docker >/dev/null 2>&1 || die "docker not found. Install Docker Desktop for Windows."
command -v jq    >/dev/null 2>&1 || die "jq not found. Install jq (e.g. via choco/scoop or Git Bash)."
command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1 || die "Docker is not running. Start Docker Desktop."
if [ "$START_FRONTEND" = 1 ]; then
  command -v npm >/dev/null 2>&1 || die "npm not found. Install Node.js from https://nodejs.org/"
fi
command -v curl >/dev/null 2>&1 || die "curl not found."

# Windows path hint (Git Bash / WSL already translate paths, but warn if on raw CMD)
if [ "${MSYSTEM:-}" = "" ] && [ "${WSL_DISTRO_NAME:-}" = "" ]; then
  warn "You appear to be running outside Git Bash/MSYS2/WSL. This script requires a bash shell."
  warn "Open 'Git Bash' or 'Ubuntu (WSL)' and re-run: ./setup-windows.sh"
fi

# ---------------------------------------------------------------------------
# 1. Install Hyperledger Fabric (samples, binaries, docker images)
# ---------------------------------------------------------------------------
if [ -d "$TEST_NETWORK" ]; then
  log "fabric-samples already present; skipping install-fabric.sh"
else
  log "installing Hyperledger Fabric (this may take several minutes)"
  chmod +x "$ROOT/install-fabric.sh"
  "$ROOT/install-fabric.sh" samples binary docker
fi

[ -d "$TEST_NETWORK" ] || die "test network not found at $TEST_NETWORK after fabric install"

# ---------------------------------------------------------------------------
# Down mode
# ---------------------------------------------------------------------------
if [ "$MODE" = "down" ]; then
  if [ -d "$TEST_NETWORK" ]; then
    log "stopping Fabric test network"
    ( cd "$TEST_NETWORK" && ./network.sh down ) || warn "network.sh down reported an error"
  fi
  pkill -f "asset-management-gui/server" 2>/dev/null || true
  pkill -f "vite" 2>/dev/null || true
  rm -rf "$ROOT/.run"
  log "everything stopped."
  exit 0
fi

# ---------------------------------------------------------------------------
# 2. Bring up everything (delegates to run-all.sh)
# ---------------------------------------------------------------------------
log "starting network, chaincode, backend, and frontend"

ARGS=()
[ "$FRESH" = 1 ] && ARGS+=(--fresh)
[ "$RUN_DEMO" = 1 ] && ARGS+=(--demo)
[ "$START_FRONTEND" = 0 ] && ARGS+=(--no-frontend)

chmod +x "$ROOT/run-all.sh"
"$ROOT/run-all.sh" "${ARGS[@]}"
