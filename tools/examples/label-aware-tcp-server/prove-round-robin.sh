#!/usr/bin/env bash
# Round-robin only: one door (:19123) with selector {kind: script} → east + west backends.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=mesh-lib.sh
source "$ROOT/mesh-lib.sh"

label_test_start_mesh "$0"
label_test_assert_round_robin 19123 12

echo "[round-robin] logs: $LOGDIR"
