# Release Checklist

Use this as a per-release template. Check items only after running them for the current release candidate.

## Pre-Release

- [ ] `make fmt`
- [ ] `make test`
- [ ] `make build`
- [ ] smoke script run (`scripts/smoke-cross-client.sh`)
- [ ] quickstart validation (`scripts/quickstart-check.sh`)
- [ ] backup/rollback validation (`scripts/backup-check.sh`, `scripts/rollback-failure-check.sh`)
- [ ] reproducible artifact + checksum generation

## Publish

- [ ] Create Git tag (for example `v0.2.0`)
- [ ] Push tag to trigger release workflow
- [ ] Attach release notes based on `docs/release-notes-template.md`
- [ ] Share release in community channels for feedback

## Post-Release Feedback

- [ ] Collect bug reports and UX issues
- [ ] Prioritize and triage for next patch
