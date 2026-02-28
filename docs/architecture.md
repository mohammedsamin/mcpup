# Architecture

## Layers

- `cmd/mcpup`: process entrypoint and exit code handling
- `internal/cli`: command parsing + orchestration
- `internal/store`: canonical config schema, validation, persistence, CRUD
- `internal/planner`: desired/current state normalization, diffing, dry-run summaries
- `internal/adapters`: per-client detect/read/apply/write/validate
- `internal/core`: reconciler and exit semantics
- `internal/backup`: snapshot, rollback, retention
- `internal/profile`: profile create/list/apply/delete orchestration
- `internal/validate`: doctor diagnostics
- `internal/output`: human/json rendering contract

## Canonical Flow (enable/disable)

1. Load canonical config
2. Mutate desired canonical state
3. Build desired client state
4. Diff current vs desired via adapter
5. Dry-run: print summary and exit
6. Snapshot client config
7. Write desired client config
8. Validate client config
9. Save canonical config

## Failure Recovery

- Client write/validate failures trigger restore from backup
- Reconcile returns stable error code for partial recovery scenarios
- Profile apply rolls back previously changed clients on partial failure
