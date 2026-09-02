#!/usr/bin/env bash
# Integration test for lure — the TFTP (RFC 1350) recovery responder.
# Builds lure, serves a temp dir, and verifies a real multi-block read
# transfer round-trips byte-for-byte, plus a clean file-not-found path.
set -euo pipefail
cd "$(dirname "$0")"

command -v curl >/dev/null || { echo "curl required"; exit 1; }

zig build
BIN=./zig-out/bin/lure

work="$(mktemp -d)"
trap 'kill "${LPID:-0}" 2>/dev/null || true; rm -rf "$work"' EXIT

# payload spans 3 blocks (512+512+276) to exercise the short-final-block path
python3 - "$work/testfw.bin" <<'PY'
import sys
open(sys.argv[1], "wb").write(bytes((i * 7 + 3) & 0xFF for i in range(1300)))
PY

( cd "$work" && exec "$OLDPWD/$BIN" ) >"$work/lure.log" 2>&1 &
LPID=$!
disown "$LPID" 2>/dev/null || true  # suppress job-control "Terminated" notice on cleanup
sleep 1

echo "== transfer test =="
curl -s --max-time 8 "tftp://127.0.0.1:6969/testfw.bin" -o "$work/got.bin"
cmp -s "$work/testfw.bin" "$work/got.bin" \
  && echo "PASS: $(wc -c < "$work/got.bin" | tr -d ' ') bytes round-tripped" \
  || { echo "FAIL: byte mismatch"; exit 1; }

echo "== not-found test =="
if curl -s --max-time 5 "tftp://127.0.0.1:6969/nope.bin" -o /dev/null; then
  echo "FAIL: expected non-zero exit for missing file"; exit 1
else
  echo "PASS: missing file rejected"
fi
kill -0 "$LPID" 2>/dev/null && echo "PASS: server survived the error" || { echo "FAIL: server died"; exit 1; }

echo "ALL LURE TESTS PASSED"
