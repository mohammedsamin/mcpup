#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-dev}"
OUT_DIR="dist/repro"
rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

build_one() {
  local os="$1"
  local arch="$2"
  local ext=""
  if [[ "$os" == "windows" ]]; then
    ext=".exe"
  fi

  local output="$OUT_DIR/mcpup_${VERSION}_${os}_${arch}${ext}"
  echo "building $output"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -trimpath -ldflags="-s -w -buildid=" -o "$output" ./cmd/mcpup
}

if [[ "${MCPUP_BUILD_ALL:-0}" == "1" ]]; then
  build_one darwin amd64
  build_one darwin arm64
  build_one linux amd64
  build_one linux arm64
  build_one windows amd64
  build_one windows arm64
else
  HOST_OS="$(go env GOOS)"
  HOST_ARCH="$(go env GOARCH)"
  build_one "$HOST_OS" "$HOST_ARCH"
fi

(
  cd "$OUT_DIR"
  shasum -a 256 * > checksums.txt
)

echo "reproducible build artifacts generated in $OUT_DIR"
