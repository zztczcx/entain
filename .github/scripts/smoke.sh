#!/usr/bin/env bash
set -euo pipefail

# Simple smoke test with no options. Requires jq.
API_HOST="localhost"
API_PORT="8000"
RACING_GRPC="localhost:9000"
SPORTS_GRPC="localhost:9001"

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required. Please install jq and re-run." >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"

mkdir -p "$DIST_DIR"
if [[ ! -x "$DIST_DIR/racing" || ! -x "$DIST_DIR/sports" || ! -x "$DIST_DIR/api" ]]; then
  echo "Building services into dist/ ..."
  (cd "$ROOT_DIR/racing" && go build -buildvcs=false -o "$DIST_DIR/racing" .)
  (cd "$ROOT_DIR/sports" && go build -buildvcs=false -o "$DIST_DIR/sports" .)
  (cd "$ROOT_DIR/api" && go build -buildvcs=false -o "$DIST_DIR/api" .)
fi

echo "Starting services..."
chmod +x "$DIST_DIR"/*
(
  cd "$ROOT_DIR/racing"; nohup "$DIST_DIR/racing" --grpc-endpoint "$RACING_GRPC" > "$ROOT_DIR/racing.out" 2>&1 & echo $! > "$ROOT_DIR/racing.pid"
)
(
  cd "$ROOT_DIR/sports"; nohup "$DIST_DIR/sports" --grpc-endpoint "$SPORTS_GRPC" > "$ROOT_DIR/sports.out" 2>&1 & echo $! > "$ROOT_DIR/sports.pid"
)
(
  cd "$ROOT_DIR/api"; nohup "$DIST_DIR/api" --api-endpoint "$API_HOST:$API_PORT" --racing-grpc-endpoint "$RACING_GRPC" --sports-grpc-endpoint "$SPORTS_GRPC" > "$ROOT_DIR/api.out" 2>&1 & echo $! > "$ROOT_DIR/api.pid"
)

cleanup() { for svc in api sports racing; do [[ -f "$ROOT_DIR/$svc.pid" ]] && kill "$(cat "$ROOT_DIR/$svc.pid")" 2>/dev/null || true; rm -f "$ROOT_DIR/$svc.pid"; done; }
trap cleanup EXIT

echo "Waiting for API at http://$API_HOST:$API_PORT ..."
set +e
for i in {1..30}; do
  code=$(curl -s -o /dev/null -w '%{http_code}' -H 'Content-Type: application/json' -d '{}' "http://$API_HOST:$API_PORT/v1/list-races") || code="000"
  [[ "$code" == "200" ]] && echo "API ready" && break
  sleep 1
done
set -e

echo "Running checks..."
resp=$(curl -sS -H 'Content-Type: application/json' -d '{}' "http://$API_HOST:$API_PORT/v1/list-races")
echo "$resp" | jq -e 'has("races") and (.races|type=="array")' >/dev/null

resp=$(curl -sS -H 'Content-Type: application/json' -d '{"filter":{"show_hidden": false}}' "http://$API_HOST:$API_PORT/v1/list-races")
echo "$resp" | jq -e 'has("races") and (.races|type=="array")' >/dev/null

code=$(curl -sS -o /dev/null -w '%{http_code}' "http://$API_HOST:$API_PORT/v1/races/1")
test "$code" = "200"
code=$(curl -sS -o /dev/null -w '%{http_code}' "http://$API_HOST:$API_PORT/v1/races/9999")
test "$code" = "404"

resp=$(curl -sS -H 'Content-Type: application/json' -d '{}' "http://$API_HOST:$API_PORT/v1/list-events")
echo "$resp" | jq -e 'has("events") and (.events|type=="array")' >/dev/null

resp=$(curl -sS -H 'Content-Type: application/json' -d '{"filter":{"show_hidden": false}}' "http://$API_HOST:$API_PORT/v1/list-events")
echo "$resp" | jq -e 'has("events") and (.events|type=="array")' >/dev/null

resp=$(curl -sS -H 'Content-Type: application/json' -d '{"filter":{"sport_ids": [1], "show_hidden": false}}' "http://$API_HOST:$API_PORT/v1/list-events")
echo "$resp" | jq -e 'has("events") and (.events|type=="array")' >/dev/null

echo "Smoke passed"
