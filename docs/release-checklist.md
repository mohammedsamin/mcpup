# Release Checklist

## Pre-Release

- [x] `make fmt`
- [x] `make test`
- [x] `make build`
- [x] smoke script run (`scripts/smoke-cross-client.sh`)
- [x] quickstart validation (`scripts/quickstart-check.sh`)
- [x] backup/rollback validation (`scripts/backup-check.sh`, `scripts/rollback-failure-check.sh`)
- [x] reproducible artifact + checksum generation

## Publish

- [ ] Create Git tag (for example `v0.1.0-alpha.1`)
- [ ] Push tag to trigger release workflow
- [ ] Attach release notes based on `docs/release-notes-template.md`
- [ ] Share release in community channels for feedback

## Post-Release Feedback

- [ ] Collect bug reports and UX issues
- [ ] Prioritize and triage for next patch
