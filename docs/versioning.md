# Versioning Policy

`mcpup` uses semantic versioning with staged stability tags.

## Stages

- Alpha (`v0.x.y-alpha.n`): early features, breaking changes expected.
- Beta (`v0.x.y-beta.n`): feature complete for target milestone, stabilization phase.
- Stable (`v1.x.y` and above): backward compatibility commitments begin.

## Rules

- Patch (`x.y.Z`): bug fixes, no breaking CLI or config schema changes.
- Minor (`x.Y.z`): backward-compatible features and new adapters.
- Major (`X.y.z`): breaking CLI/config behavior changes.

## Schema Versioning

- Canonical config schema version is tracked in `~/.mcpup/config.json`.
- New schema versions require explicit migration strategy before rollout.

## Release Tags

- Stable: `v1.2.3`
- Prerelease: `v1.2.3-beta.1`, `v1.2.3-rc.1`
