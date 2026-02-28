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

# README quickstart validation
"$BIN" init
"$BIN" add github --command "echo github" --env GITHUB_TOKEN='${GITHUB_TOKEN}'
"$BIN" enable github --client cursor
"$BIN" disable github --client cursor --tool delete_issue
"$BIN" profile create coding --servers github
"$BIN" profile apply coding
"$BIN" doctor
"$BIN" rollback --client cursor

echo "quickstart-check: OK"
