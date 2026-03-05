# Changelog

All notable changes to this project are documented in this file.

The format is inspired by Keep a Changelog and semantic versioning.

## [Unreleased]

### Added

- Full CLI contract for `init`, `add`, `remove`, `enable`, `disable`, `list`, `status`, `profile`, `clients list`, `doctor`, `rollback`.
- Canonical config schema, validation, migrations, and store operations.
- Planner, diffing, dry-run summaries, reconciler with backup safety.
- Client adapters for Claude Code, Cursor, Claude Desktop, Codex, and OpenCode.
- Adapter fixtures and integration tests.
- Doctor diagnostics and actionable output.
- CI and release automation configuration.

### Changed

- Mutating CLI operations are idempotent and support dry-run flow.
- Managed writes now preserve unmanaged client MCP entries instead of replacing whole maps.
- `setup --update` and `update` now reconcile affected clients before canonical config is saved.

### Fixed

- Reconcile failure recovery now restores from backup and returns stable exit semantics.
- Rollback sync now skips unmanaged external entries instead of forcing them into canonical config.
- Registry-backed add/setup/update now fail early on missing launchers or required env vars.
- Doctor remains read-only and now reports ownership and managed drift.
