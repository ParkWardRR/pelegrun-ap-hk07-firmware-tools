#!/usr/bin/env bash
# Build cross-compiled release artifacts into dist/ with a SHA256SUMS manifest.
#
#   pelegrun (Go)  — three CLI utilities (discover, envcheck, redact)
#   quarry  (Rust) — the core image re-head + serial tool
#
# Usage:  scripts/dist.sh [version]   (version defaults to `git describe`)
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
OUT="dist"
rm -rf "$OUT"
mkdir -p "$OUT"
echo "building pelegrun-ap-hk07-firmware-tools $VERSION → $OUT/"

# ---- pelegrun (Go) : os/arch matrix ----
GO_TARGETS=(
  "darwin/arm64" "darwin/amd64"
  "linux/amd64"  "linux/arm64"
  "windows/amd64"
)
for t in "${GO_TARGETS[@]}"; do
  os="${t%/*}"; arch="${t#*/}"
  ext=""; [ "$os" = "windows" ] && ext=".exe"
  bin="$OUT/pelegrun-${os}-${arch}${ext}"
  ( cd go && GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
      go build -trimpath -ldflags "-s -w -X main.Version=${VERSION#v}" \
      -o "../$bin" ./cmd/pelegrun )
  echo "  pelegrun  $os/$arch"
done

# ---- quarry (Rust) : host build only; CI cross-builds per runner ----
if command -v cargo >/dev/null; then
  cargo build -p quarry --release --quiet
  host="$(rustc -vV | sed -n 's/host: //p')"
  cp "target/release/quarry" "$OUT/quarry-${host}"
  echo "  quarry   $host (host only)"
fi

# ---- checksums ----
( cd "$OUT" && shasum -a 256 * > SHA256SUMS ) 2>/dev/null || \
  ( cd "$OUT" && sha256sum * > SHA256SUMS )
echo "wrote $OUT/SHA256SUMS"
ls -1 "$OUT"
