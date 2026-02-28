#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

go test ./internal/core -run 'TestReconcile(Write|Validation)FailureRestoresBackup' -count=1

echo "rollback-failure-check: OK"
