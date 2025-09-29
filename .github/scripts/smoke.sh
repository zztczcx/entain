#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   .github/scripts/smoke.sh [--start] [--stop] [--api-host HOST] [--api-port PORT] [--racing :grpc PORT] [--sports :grpc PORT]

API_HOST="localhost"
API_PORT="8000"
RACING_GRPC="localhost:9000"
SPORTS_GRPC="localhost:9001"
DO_START=false
DO_STOP=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --start) DO_START=true; shift ;;
    --stop) DO_STOP=true; shift ;;
    --api-host) API_HOST="$2"; shift 2 ;;
    --api-port) API_PORT="$2"; shift 2 ;;
    --racing) RACING_GRPC="$2"; shift 2 ;;
    --sports) SPORTS_GRPC="$2"; shift 2 ;;
    *) echo "Unknown arg: $1" >&2; exit 2 ;;
  esac
done

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"

start_services() {
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
    cd "$ROOT_DIR/racing"
    nohup "$DIST_DIR/racing" --grpc-endpoint "$RACING_GRPC" > "$ROOT_DIR/racing.out" 2>&1 & echo $! > "$ROOT_DIR/racing.pid"
  )
  (
    cd "$ROOT_DIR/sports"
    nohup "$DIST_DIR/sports" --grpc-endpoint "$SPORTS_GRPC" > "$ROOT_DIR/sports.out" 2>&1 & echo $! > "$ROOT_DIR/sports.pid"
  )
  (
    cd "$ROOT_DIR/api"
    nohup "$DIST_DIR/api" --api-endpoint "$API_HOST:$API_PORT" --racing-grpc-endpoint "$RACING_GRPC" --sports-grpc-endpoint "$SPORTS_GRPC" > "$ROOT_DIR/api.out" 2>&1 & echo $! > "$ROOT_DIR/api.pid"
  )
}

stop_services() {
  for svc in api sports racing; do
    if [[ -f "$ROOT_DIR/$svc.pid" ]]; then
      kill "$(cat "$ROOT_DIR/$svc.pid")" 2>/dev/null || true
      rm -f "$ROOT_DIR/$svc.pid"
    fi
  done
}

wait_for_api() {
  echo "Waiting for API at http://$API_HOST:$API_PORT ..."
  set +e
  for i in {1..30}; do
    code=$(curl -s -o /dev/null -w '%{http_code}' -H 'Content-Type: application/json' -d '{}' "http://$API_HOST:$API_PORT/v1/list-races") || code="000"
    if [[ "$code" == "200" ]]; then
      echo "API ready"
      set -e
      return 0
    fi
    sleep 1
  done
  set -e
  echo "API timeout" >&2
  tail -n +1 "$ROOT_DIR"/*.out || true
  return 1
}

run_checks() {
  set +e
  failures=0

  resp=$(curl -sS -H 'Content-Type: application/json' -d '{}' "http://$API_HOST:$API_PORT/v1/list-races")
  echo "$resp" | grep -q '"races"' || { echo "list-races missing races"; echo "$resp"; failures=$((failures+1)); }

  resp=$(curl -sS -H 'Content-Type: application/json' -d '{"filter":{"show_hidden": false}}' "http://$API_HOST:$API_PORT/v1/list-races")
  echo "$resp" | grep -q '"races"' || { echo "list-races filtered missing races"; echo "$resp"; failures=$((failures+1)); }

  code=$(curl -sS -o /dev/null -w '%{http_code}' "http://$API_HOST:$API_PORT/v1/races/1")
  [[ "$code" == "200" ]] || { echo "get-race 1 expected 200, got $code"; failures=$((failures+1)); }
  code=$(curl -sS -o /dev/null -w '%{http_code}' "http://$API_HOST:$API_PORT/v1/races/9999")
  [[ "$code" == "404" ]] || { echo "get-race 9999 expected 404, got $code"; failures=$((failures+1)); }

  resp=$(curl -sS -H 'Content-Type: application/json' -d '{}' "http://$API_HOST:$API_PORT/v1/list-events")
  echo "$resp" | grep -q '"events"' || { echo "list-events missing events"; echo "$resp"; failures=$((failures+1)); }
  resp=$(curl -sS -H 'Content-Type: application/json' -d '{"filter":{"show_hidden": false}}' "http://$API_HOST:$API_PORT/v1/list-events")
  echo "$resp" | grep -q '"events"' || { echo "list-events filtered missing events"; echo "$resp"; failures=$((failures+1)); }
  resp=$(curl -sS -H 'Content-Type: application/json' -d '{"filter":{"sport_ids": [1], "show_hidden": false}}' "http://$API_HOST:$API_PORT/v1/list-events")
  echo "$resp" | grep -q '"events"' || { echo "list-events by sport missing events"; echo "$resp"; failures=$((failures+1)); }

  if [[ $failures -ne 0 ]]; then
    echo "Smoke failed with $failures failure(s). Logs:" >&2
    tail -n +1 "$ROOT_DIR"/*.out || true
    return 1
  fi
  echo "Smoke passed"
  return 0
}

if $DO_START; then
  start_services
fi

trap '[[ $DO_STOP == true ]] && stop_services' EXIT

wait_for_api
run_checks


