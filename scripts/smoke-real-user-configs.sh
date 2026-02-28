#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BIN="${ROOT_DIR}/bin/mcpup"

if [[ ! -x "$BIN" ]]; then
  echo "binary not found at $BIN; run make build first" >&2
  exit 1
fi

# Non-destructive smoke run against the current user environment.
# Uses doctor only so no client config writes are performed.
"$BIN" doctor || true

echo "smoke-real-user-configs: completed"
