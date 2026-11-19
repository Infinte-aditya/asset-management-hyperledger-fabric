#!/usr/bin/env bash
#
# load-example.sh — register an example route's container (if missing) and
# ingest its GPS fixes into the ledger.
#
# Usage:
#   ./examples/load-example.sh examples/shanghai-rotterdam.json
#   ./examples/load-example.sh examples/reefer-atlantic.json 40RH "Reefer Co" 3800 NLRTM
#
# Optional args: [Type] [Owner] [TareWeight] [Destination]
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

FILE="${1:?usage: load-example.sh <examples/foo.json> [Type] [Owner] [TareWeight] [Destination]}"
case "$FILE" in
  /*) : ;;
  *) FILE="$PWD/$FILE" ;;
esac
[ -f "$FILE" ] || { echo "error: file not found: $FILE" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "error: jq is required" >&2; exit 1; }

ID="$(jq -r '.[0].containerId' "$FILE")"
TYPE="${2:-40RH}"
OWNER="${3:-Demo Logistics}"
WT="${4:-3700}"
DEST="${5:-USLGB}"

echo "==> example: $FILE"
echo "==> container: $ID"

cd "$ROOT/application-gateway-go"
if go run . ReadContainer "$ID" >/dev/null 2>&1; then
  echo "==> container already on ledger"
else
  echo "==> registering $ID ($TYPE, owner=$OWNER)"
  go run . RegisterContainer "$ID" "$TYPE" "$OWNER" "$WT" "$DEST"
  go run . ReportEvent "$ID" Loaded CNSHA Depot-Ops "Loaded aboard vessel" >/dev/null 2>&1 || true
fi

echo "==> ingesting fixes"
cd "$ROOT/gps-ingestor"
go run . -file "$FILE"

cd "$ROOT/application-gateway-go"
echo "==> done. Track:"
go run . GetPositionHistory "$ID"
