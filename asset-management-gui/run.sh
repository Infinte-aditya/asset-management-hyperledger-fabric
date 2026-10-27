#!/usr/bin/env bash
# Starts everything needed for the Container Ledger GUI:
#   1. Fabric test network + channel (if not already running)
#   2. "container-cc" chaincode (if not already deployed)
#   3. The GUI server (REST API + Svelte frontend)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TEST_NETWORK="$SCRIPT_DIR/../fabric-samples/test-network"
FABRIC_BIN="$SCRIPT_DIR/../fabric-samples/bin"
CHANNEL="${CHANNEL_NAME:-mychannel}"
CC_NAME="${CHAINCODE_NAME:-container-cc}"
CC_PATH="../asset-transfer-basic/chaincode-go"

log() { echo "==> $*"; }
die() { echo "ERROR: $*" >&2; exit 1; }

[ -d "$TEST_NETWORK" ] || die "test-network not found at $TEST_NETWORK"

# --- Docker ----------------------------------------------------------------
if ! docker info >/dev/null 2>&1; then
  die "Docker is not available. Start Docker or run: sudo usermod -aG docker \$USER, then re-login."
fi

# --- Helper: run peer CLI against org1 -------------------------------------
peer_cmd() {
  (
    cd "$TEST_NETWORK"
    export PATH="$FABRIC_BIN:$PATH"
    export FABRIC_CFG_PATH="$SCRIPT_DIR/../fabric-samples/config"
    export CORE_PEER_TLS_ENABLED=true
    export CORE_PEER_LOCALMSPID="Org1MSP"
    export CORE_PEER_TLS_ROOTCERT_FILE="$TEST_NETWORK/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="$TEST_NETWORK/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp"
    "$@"
  )
}

network_up() {
  docker ps --filter "name=^peer0.org1.example.com$" --filter "status=running" -q | grep -q .
}

channel_ready() {
  peer_cmd peer channel getinfo -c "$CHANNEL" >/dev/null 2>&1
}

chaincode_ready() {
  peer_cmd peer lifecycle chaincode querycommitted -C "$CHANNEL" --name "$CC_NAME" \
    --peerAddresses localhost:7051 \
    --tlsRootCertFiles "$TEST_NETWORK/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" \
    >/dev/null 2>&1
}

# --- 1. Test network ---------------------------------------------------------
if network_up; then
  log "Test network already running"
else
  log "Starting Fabric test network (channel: $CHANNEL)..."
  if ! (cd "$TEST_NETWORK" && ./network.sh up createChannel); then
    # Channel may already exist in persisted volumes; only fail if unusable
    network_up || die "failed to start the test network"
    channel_ready || die "test network started but channel '$CHANNEL' is not usable"
    log "Network up (channel already existed)"
  fi
fi

# --- 2. Chaincode ------------------------------------------------------------
if chaincode_ready; then
  log "Chaincode '$CC_NAME' already deployed on '$CHANNEL'"
else
  log "Deploying chaincode '$CC_NAME'..."
  (cd "$TEST_NETWORK" && ./network.sh deployCC -ccn "$CC_NAME" -ccp "$CC_PATH" -ccl go)
fi

# --- 3. GUI ------------------------------------------------------------------
log "Container Ledger → http://localhost:5173  (frontend)  /  http://localhost:8080 (API)  (Ctrl+C to stop)"
echo
cd "$SCRIPT_DIR/server"
exec go run .
