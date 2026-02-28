#!/usr/bin/env bash
set -euo pipefail

TARGET_DIR="${1:-dist/repro}"
CHECKSUM_FILE="${2:-$TARGET_DIR/checksums.txt}"

if [[ ! -d "$TARGET_DIR" ]]; then
  echo "target directory not found: $TARGET_DIR" >&2
  exit 1
fi

if [[ ! -f "$CHECKSUM_FILE" ]]; then
  echo "checksum file not found: $CHECKSUM_FILE" >&2
  exit 1
fi

(
  cd "$TARGET_DIR"
  shasum -a 256 -c "$(basename "$CHECKSUM_FILE")"
)
