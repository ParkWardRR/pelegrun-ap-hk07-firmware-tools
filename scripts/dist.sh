#!/usr/bin/env bash
# Build cross-compiled release artifacts into dist/ with a SHA256SUMS manifest.
#
#   swallow (Go)  — the CLI/TUI, cross-compiled to 5 os/arch targets
#   lure    (Zig) — the TFTP recovery responder, POSIX targets only (libc sockets)
#   quarry  (Rust)— built for the host only here; CI builds it per-runner
#
# Usage:  scripts/dist.sh [version]   (version defaults to `git describe`)
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
OUT="dist"
rm -rf "$OUT"
mkdir -p "$OUT"
echo "building swallow-ap-hk07-firmware-tools $VERSION → $OUT/"

# ---- swallow (Go) : os/arch matrix ----
GO_TARGETS=(
  "darwin/arm64" "darwin/amd64"
  "linux/amd64"  "linux/arm64"
  "windows/amd64"
)
for t in "${GO_TARGETS[@]}"; do
  os="${t%/*}"; arch="${t#*/}"
  ext=""; [ "$os" = "windows" ] && ext=".exe"
  bin="$OUT/swallow-${os}-${arch}${ext}"
  ( cd go && GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
      go build -trimpath -ldflags "-s -w -X main.Version=${VERSION#v}" \
      -o "../$bin" ./cmd/swallow )
  echo "  swallow  $os/$arch"
done

# ---- lure (Zig) : POSIX targets only (uses libc sockets; not ported to winsock) ----
if command -v zig >/dev/null; then
  ZIG_TARGETS=(
    "aarch64-macos"    "x86_64-macos"
    "x86_64-linux-musl" "aarch64-linux-musl"
  )
  for zt in "${ZIG_TARGETS[@]}"; do
    ( cd zig && zig build -Dtarget="$zt" -Doptimize=ReleaseSafe >/dev/null )
    cp "zig/zig-out/bin/lure" "$OUT/lure-${zt}"
    echo "  lure     $zt"
  done
else
  echo "  (skipping lure: zig not found)"
fi

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
