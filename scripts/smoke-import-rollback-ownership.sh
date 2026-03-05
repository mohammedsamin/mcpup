#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BIN="${ROOT_DIR}/bin/mcpup"

if [[ ! -x "$BIN" ]]; then
  echo "binary not found at $BIN; run make build first" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

export HOME="$TMP_DIR/home"
export MCPUP_CONFIG="$TMP_DIR/config.json"
mkdir -p "$HOME/.cursor"

cat >"$HOME/.cursor/mcp.json" <<'JSON'
{
  "mcpServers": {
    "github": {
      "command": "echo",
      "args": ["gh"],
      "enabled": true
    },
    "external-only": {
      "command": "echo",
      "args": ["external"],
      "enabled": true
    }
  }
}
JSON

"$BIN" init --import

rg -q '"github"' "$MCPUP_CONFIG"
if rg -q '"external-only"' "$MCPUP_CONFIG"; then
  echo "external-only should not be imported into canonical config" >&2
  exit 1
fi

"$BIN" disable github --client cursor
"$BIN" rollback --client cursor

rg -q '"github"' "$MCPUP_CONFIG"
if rg -q '"external-only"' "$MCPUP_CONFIG"; then
  echo "external-only should remain unmanaged after rollback" >&2
  exit 1
fi
rg -q '"external-only"' "$HOME/.cursor/mcp.json"

echo "smoke-import-rollback-ownership: OK"
