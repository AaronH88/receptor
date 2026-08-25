#!/usr/bin/env bash
# Shared mesh startup for label-aware tcp-server manual tests.
set -euo pipefail

label_test_root() {
  cd "$(dirname "${BASH_SOURCE[1]}")"
  pwd
}

label_test_start_mesh() {
  ROOT="$(label_test_root)"
  RECEPTOR_ROOT="$(cd "$ROOT/../../.." && pwd)"
  BIN="${RECEPTOR_ROOT}/.tmp/receptor-label-test/receptor"
  LOGDIR="${RECEPTOR_ROOT}/.tmp/receptor-label-test/logs"
  RUNDIR="/tmp/receptor-label-test"
  export ROOT RECEPTOR_ROOT BIN LOGDIR RUNDIR

  mkdir -p "$LOGDIR" "$RUNDIR"
  LABEL_TEST_PIDS=()

  label_test_cleanup() {
    for pid in "${LABEL_TEST_PIDS[@]}"; do
      kill "$pid" 2>/dev/null || true
    done
    wait 2>/dev/null || true
  }
  trap label_test_cleanup EXIT

  echo "[mesh] building receptor from $RECEPTOR_ROOT"
  export PATH="/opt/homebrew/bin:${PATH:-}"
  (cd "$RECEPTOR_ROOT" && go build -o "$BIN" ./cmd/receptor-cl)

  echo "[mesh] starting HTTP backends"
  python3 "$ROOT/backend_server.py" --port 19001 --body "SCRIPT-EAST" >"$LOGDIR/backend-19001.log" 2>&1 &
  LABEL_TEST_PIDS+=($!)
  python3 "$ROOT/backend_server.py" --port 19002 --body "SCRIPT-WEST" >"$LOGDIR/backend-19002.log" 2>&1 &
  LABEL_TEST_PIDS+=($!)
  python3 "$ROOT/backend_server.py" --port 19003 --body "AGENT-OK" >"$LOGDIR/backend-19003.log" 2>&1 &
  LABEL_TEST_PIDS+=($!)
  sleep 1

  echo "[mesh] starting receptor hub + bridge-east + bridge-west"
  "$BIN" --config "$ROOT/configs/hub.yaml" >"$LOGDIR/hub.log" 2>&1 &
  LABEL_TEST_PIDS+=($!)
  sleep 1
  "$BIN" --config "$ROOT/configs/bridge-east.yaml" >"$LOGDIR/bridge-east.log" 2>&1 &
  LABEL_TEST_PIDS+=($!)
  "$BIN" --config "$ROOT/configs/bridge-west.yaml" >"$LOGDIR/bridge-west.log" 2>&1 &
  LABEL_TEST_PIDS+=($!)

  echo "[mesh] waiting for hub door :19120"
  local deadline=$((SECONDS + 30))
  until curl -sf --max-time 1 "http://127.0.0.1:19120/" >/dev/null 2>&1; do
    if (( SECONDS > deadline )); then
      echo "[mesh] timed out waiting for mesh" >&2
      tail -20 "$LOGDIR/hub.log" >&2 || true
      exit 1
    fi
    sleep 1
  done
}

label_test_assert_round_robin() {
  local port="${1:-19123}"
  local attempts="${2:-12}"
  local saw_east=0
  local saw_west=0

  echo "[round-robin] door :${port} selector {kind: script} — ${attempts} sequential curls"
  echo "[round-robin] (one connection per curl → hub picks next matching backend)"
  for i in $(seq 1 "$attempts"); do
    local body
    body="$(curl -sf "http://127.0.0.1:${port}/")"
    echo "  request $i → $body"
    case "$body" in
      SCRIPT-EAST) saw_east=1 ;;
      SCRIPT-WEST) saw_west=1 ;;
      *)
        echo "[round-robin] FAIL: unexpected body '$body'" >&2
        exit 1
        ;;
    esac
  done

  if (( saw_east == 0 || saw_west == 0 )); then
    echo "[round-robin] FAIL: expected both SCRIPT-EAST and SCRIPT-WEST over ${attempts} requests" >&2
    echo "[round-robin]       saw_east=$saw_east saw_west=$saw_west" >&2
    exit 1
  fi

  echo "[round-robin] PASS: both script backends reached via single door :${port}"
}
