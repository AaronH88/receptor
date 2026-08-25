# Label-aware `tcp-server` manual test

Proves **selector-based routing** on branch `feature/label-aware-tcp-server`.

## Topology

```
curl :19120 ──► hub tcp-server  selector {kind: script, region: us-east-1}
                    └── mesh ──► bridge-east / scr-east ──► backend :19001  "SCRIPT-EAST"

curl :19121 ──► hub tcp-server  selector {kind: script, region: us-west-2}
                    └── mesh ──► bridge-west / scr-west ──► backend :19002  "SCRIPT-WEST"

curl :19122 ──► hub tcp-server  selector {kind: agent}
                    └── mesh ──► bridge-east / agt ──► backend :19003  "AGENT-OK"

curl :19123 ──► hub tcp-server  selector {kind: script}   ← broad: matches BOTH script backends
                    ├── round-robin ──► scr-east  → SCRIPT-EAST
                    └── round-robin ──► scr-west  → SCRIPT-WEST
```

Each `tcp-client` advertises arbitrary **tags** on the mesh. The hub `tcp-server` picks a matching `(node, service)` per connection (round-robin if multiple match).

**Tags vs selectors:** bridges only **advertise** `tags`; hub doors only **select**. Same key names when a door targets a slice — not duplicated config on every file.

## Prerequisites

- Go 1.21+ (`/opt/homebrew/bin/go` on macOS is fine)
- Python 3
- `curl`
- Checkout branch `feature/label-aware-tcp-server`

## Run

```bash
cd /Users/ahetheri/openshell_explore/receptor
git checkout feature/label-aware-tcp-server

chmod +x tools/examples/label-aware-tcp-server/prove.sh
./tools/examples/label-aware-tcp-server/prove.sh
```

Runs **specific-selector** doors (`:19120`–`:19122`) and **round-robin** door (`:19123`).

### Round-robin only

```bash
chmod +x tools/examples/label-aware-tcp-server/prove-round-robin.sh
./tools/examples/label-aware-tcp-server/prove-round-robin.sh
```

Expected output (excerpt):

```
[round-robin] door :19123 selector {kind: script} — 12 sequential curls
  request 1 → SCRIPT-EAST
  request 2 → SCRIPT-WEST
  ...
[round-robin] PASS: both script backends reached via single door :19123
```

## Expected output (full prove.sh)

```
PASS selector kind=script region=us-east-1 (port 19120 → SCRIPT-EAST)
PASS selector kind=script region=us-west-2 (port 19121 → SCRIPT-WEST)
PASS selector kind=agent (port 19122 → AGENT-OK)
PASS: both script backends reached via single door :19123
ALL PASS — label-aware tcp-server routes by advertisement tags
```

Logs: `receptor/.tmp/receptor-label-test/logs/`

## Manual curls (if mesh already running)

```bash
curl -s http://127.0.0.1:19120/   # SCRIPT-EAST
curl -s http://127.0.0.1:19121/   # SCRIPT-WEST
curl -s http://127.0.0.1:19122/   # AGENT-OK
for i in 1 2 3 4 5 6; do curl -s http://127.0.0.1:19123/; echo; done   # EAST/WEST mix
```

## Inspect advertisements

After `prove.sh` starts the hub (or run receptors yourself):

```bash
receptorctl --socket /tmp/receptor-label-test/hub.sock status
```

Look for `scr-east`, `scr-west`, `agt` entries with `Tags` containing your key/value pairs.

## Cleanup

`prove.sh` stops processes on exit. To reset data dirs:

```bash
rm -rf /tmp/receptor-label-test receptor/.tmp/receptor-label-test
```
