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
    "custom-unmanaged": {
      "command": "echo",
      "args": ["custom"],
      "enabled": true
    }
  }
}
JSON

"$BIN" init
"$BIN" add github --command echo --arg gh
"$BIN" enable github --client cursor

rg -q '"custom-unmanaged"' "$HOME/.cursor/mcp.json"
rg -q '"github"' "$HOME/.cursor/mcp.json"
rg -q '"managedServers"' "$HOME/.cursor/mcp.json"

"$BIN" remove github --yes

rg -q '"custom-unmanaged"' "$HOME/.cursor/mcp.json"
if rg -q '"github"' "$HOME/.cursor/mcp.json"; then
  echo "managed github entry should have been removed" >&2
  exit 1
fi

echo "smoke-preserve-unmanaged: OK"
