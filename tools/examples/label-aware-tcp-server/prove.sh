#!/usr/bin/env bash
# Manual proof for label-aware tcp-server (feature/label-aware-tcp-server).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=mesh-lib.sh
source "$ROOT/mesh-lib.sh"

label_test_start_mesh "$0"

assert_body() {
  local port="$1"
  local want="$2"
  local label="$3"
  local body
  body="$(curl -sf "http://127.0.0.1:${port}/")"
  if [[ "$body" != "$want" ]]; then
    echo "[prove] FAIL $label: port $port got '$body', want '$want'" >&2
    exit 1
  fi
  echo "[prove] PASS $label (port $port → $body)"
}

echo "[prove] === specific selectors (one backend per door) ==="
assert_body 19120 "SCRIPT-EAST" "selector kind=script region=us-east-1"
assert_body 19121 "SCRIPT-WEST" "selector kind=script region=us-west-2"
assert_body 19122 "AGENT-OK"     "selector kind=agent"

echo "[prove] === broad selector (round-robin across script backends) ==="
label_test_assert_round_robin 19123 12

if [[ -S "$RUNDIR/hub.sock" ]] && command -v receptorctl >/dev/null 2>&1; then
  echo "[prove] hub advertisements (receptorctl):"
  receptorctl --socket "$RUNDIR/hub.sock" status 2>/dev/null | head -40 || true
fi

echo "[prove] ALL PASS — label-aware tcp-server routes by advertisement tags"
echo "[prove] logs: $LOGDIR"
