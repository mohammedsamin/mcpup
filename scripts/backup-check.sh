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

cat > "$HOME/.cursor/mcp.json" <<JSON
{"mcpServers":{"github":{"enabled":false}}}
JSON

"$BIN" init
"$BIN" add github --command "echo github"
"$BIN" enable github --client cursor
"$BIN" disable github --client cursor --tool delete_issue
"$BIN" profile create coding --servers github
"$BIN" profile apply coding --yes

BACKUP_DIR="$HOME/.mcpup/backups/cursor"
if [[ ! -d "$BACKUP_DIR" ]]; then
  echo "backup dir missing: $BACKUP_DIR" >&2
  exit 1
fi

META_COUNT=$(find "$BACKUP_DIR" -name '*.meta.json' | wc -l | tr -d ' ')
if [[ "$META_COUNT" -lt 1 ]]; then
  echo "expected at least one backup metadata file" >&2
  exit 1
fi

echo "backup-check: OK ($META_COUNT snapshots)"
