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

### Fixed

- Reconcile failure recovery now restores from backup and returns stable exit semantics.
