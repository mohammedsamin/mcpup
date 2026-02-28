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
mkdir -p "$HOME"

mkdir -p "$HOME/.cursor"
mkdir -p "$HOME/.codex"
mkdir -p "$HOME/.config/opencode"
mkdir -p "$HOME/Library/Application Support/Claude"

cp "$ROOT_DIR/testdata/fixtures/cursor/normal.json" "$HOME/.cursor/mcp.json"
cp "$ROOT_DIR/testdata/fixtures/claudecode/normal.json" "$HOME/.claude.json"
cp "$ROOT_DIR/testdata/fixtures/claudedesktop/normal.json" "$HOME/Library/Application Support/Claude/claude_desktop_config.json"
cp "$ROOT_DIR/testdata/fixtures/codex/normal.toml" "$HOME/.codex/config.toml"
cp "$ROOT_DIR/testdata/fixtures/opencode/normal.json" "$HOME/.config/opencode/opencode.json"

"$BIN" init
"$BIN" add github --command "echo github"

"$BIN" enable github --client claude-code
"$BIN" enable github --client cursor
"$BIN" enable github --client claude-desktop
"$BIN" enable github --client codex
"$BIN" enable github --client opencode

"$BIN" disable github --client cursor --tool delete_issue
"$BIN" --dry-run profile create coding --servers github
"$BIN" profile create coding --servers github
"$BIN" --dry-run profile apply coding
"$BIN" --json doctor

echo "smoke-cross-client: OK"
