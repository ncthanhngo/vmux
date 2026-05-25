#!/usr/bin/env bash
# Cross-compile the Go sidecar for arm64 + amd64 and lipo them into a single
# universal binary at sidecar/bin/vmux-sidecar.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SIDECAR_DIR="$REPO_ROOT/sidecar"
OUT_DIR="$SIDECAR_DIR/bin"
VERSION="${VMUX_VERSION:-0.0.0}"
LDFLAGS="-s -w -X main.version=${VERSION}"

mkdir -p "$OUT_DIR"

echo "Building vmux-sidecar v${VERSION} (universal)…"
( cd "$SIDECAR_DIR" && GOOS=darwin GOARCH=arm64 \
    go build -trimpath -ldflags "$LDFLAGS" -o "$OUT_DIR/vmux-sidecar-arm64" ./cmd/vmux-sidecar )
( cd "$SIDECAR_DIR" && GOOS=darwin GOARCH=amd64 \
    go build -trimpath -ldflags "$LDFLAGS" -o "$OUT_DIR/vmux-sidecar-amd64" ./cmd/vmux-sidecar )

lipo -create \
  "$OUT_DIR/vmux-sidecar-arm64" \
  "$OUT_DIR/vmux-sidecar-amd64" \
  -output "$OUT_DIR/vmux-sidecar"
chmod +x "$OUT_DIR/vmux-sidecar"
rm -f "$OUT_DIR/vmux-sidecar-arm64" "$OUT_DIR/vmux-sidecar-amd64"

echo "Built: $OUT_DIR/vmux-sidecar"
lipo -info "$OUT_DIR/vmux-sidecar"
