# Changelog

All notable changes to this project are documented in this file.

The format is inspired by Keep a Changelog and semantic versioning.

## [Unreleased]

## [0.4.0] - 2026-03-06

### Added

- Guided `setup` onboarding for selecting clients, installing registry servers, and collecting required inputs.
- Remote `HTTP/SSE` MCP server support with canonical `url`, `headers`, and `transport` handling.
- Ownership-aware reconciliation that preserves unmanaged client entries instead of replacing full client maps.
- Registry prompt normalization and legacy definition migration for stale high-risk templates like Notion, Playwright, Filesystem, and ElevenLabs.

### Changed

- `setup --update` and `update` now reconcile affected clients before canonical config is saved.
- Wizard and CLI registry flows now collect user-facing inputs consistently instead of exposing low-level raw env shapes where avoidable.
- README and release-facing docs now reflect the current 13-client, 97-server product surface.

### Fixed

- Remove flows no longer leave canonical state ahead of failed client writes.
- Wizard add/enable flows no longer persist partial failed client mutations.
- Rollback now reports canonical-sync failures instead of claiming silent success.
- Doctor remains read-only and reports ownership, drift, legacy registry definitions, and required-env problems more accurately.
- Broken registry templates and stale legacy definitions for Playwright, Filesystem, Notion, and ElevenLabs are repaired or detected automatically.
