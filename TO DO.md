# TO DO

Master execution checklist for `mcpup`.
No timeline. No scheduling. Just execution steps.

## Step 1 - Lock Product Contract

- [x] T001 Lock v1 product principles: CLI-only, no daemon, no proxy, direct config writes.
- [x] T002 Lock supported v1 clients: Claude Code, Cursor, Claude Desktop, Codex, OpenCode.
- [x] T003 Lock v1 core features: add, remove, enable, disable, per-tool enable/disable (v1), list, status, profiles, doctor, backup, rollback, dry-run/json output, import existing client configs.
- [x] T004 Lock non-goals: no GUI, no cloud sync, no hosted server registry/store, no team policy system, no daemon mode, no proxy/gateway mode, no marketplace/plugin ecosystem.
- [x] T005 Lock naming for all commands and flags so they do not change during implementation: `init`, `add`, `remove`, `enable`, `disable`, `list`, `status`, `profile create|apply|list|delete`, `clients list`, `doctor`, `rollback`; global flags `--dry-run`, `--json`, `--verbose`, `--yes`.
- [x] T006 Lock success criteria for v1 acceptance: all v1 commands work on 5 clients, per-tool controls work in v1, import existing client configs works, backups are created before every mutating command, rollback is proven in failure tests, core workflows require no manual JSON editing, and tests pass on macOS/Linux/Windows.

Locked decisions for Step 1:

- [x] D-STEP1-01 Product mode is CLI-only.
- [x] D-STEP1-02 v1 client list is locked to:
  - Claude Code - terminal devs, heaviest MCP users
  - Cursor - biggest AI editor, massive MCP adoption
  - Claude Desktop - most complained about config pain
  - Codex - OpenAI's CLI, growing fast
  - OpenCode - open source crowd, credibility
- [x] D-STEP1-03 Core features are locked, including per-tool controls in v1 and import of existing client configs.
- [x] D-STEP1-04 Non-goals are locked to prevent scope creep in v1.
- [x] D-STEP1-05 Command and flag naming is locked for implementation and docs stability.
- [x] D-STEP1-06 v1 success criteria are locked and measurable.

## Step 2 - Create Repository Skeleton

- [x] T007 Initialize Go module and base `cmd/mcpup` entrypoint.
- [x] T008 Create internal packages: `cli`, `core`, `store`, `profile`, `backup`, `planner`, `adapters`, `validate`, `output`.
- [x] T009 Create adapter subfolders: `claudecode`, `cursor`, `claudedesktop`, `codex`, `opencode`.
- [x] T010 Create `docs` folder with placeholders for command and client docs.
- [x] T011 Create `testdata/fixtures` folder for each client.
- [x] T012 Create `scripts` folder for local checks and release helpers.
- [x] T013 Create `.github/workflows` folder for CI.
- [x] T014 Add base `Makefile` with build, test, lint, and fmt commands.

## Step 3 - Define CLI Contract

- [x] T015 Implement root command with global flags: `--dry-run`, `--json`, `--verbose`, `--yes`.
- [x] T016 Implement `mcpup init`.
- [x] T017 Implement `mcpup add`.
- [x] T018 Implement `mcpup remove`.
- [x] T019 Implement `mcpup enable`.
- [x] T020 Implement `mcpup disable`.
- [x] T021 Implement `mcpup list` and `mcpup status`.
- [x] T022 Implement profile commands: `create`, `apply`, `list`, `delete`.
- [x] T023 Implement `mcpup clients list`, `mcpup doctor`, `mcpup rollback`.

## Step 4 - Define Canonical Config Model

- [x] T024 Define schema versioning for `~/.mcpup/config.json`.
- [x] T025 Define `servers` schema (`command`, `args`, `env`, `description`).
- [x] T026 Define `clients` schema (`enabledServers` map/list structure).
- [x] T027 Define `profiles` schema and `activeProfile`.
- [x] T028 Define server name validation rules and reserved names.
- [x] T029 Define env value handling rules (`literal` vs `${ENV_VAR}`).
- [x] T030 Define schema migration strategy for future versions.

## Step 5 - Implement Config Store

- [x] T031 Implement config path discovery and initialization.
- [x] T032 Implement read config with strict validation and typed errors.
- [x] T033 Implement write config with atomic file replacement.
- [x] T034 Implement CRUD operations for server definitions.
- [x] T035 Implement CRUD operations for profiles.
- [x] T036 Implement client-enable state updates in canonical config.

## Step 6 - Build Planner and Reconciler

- [x] T037 Implement desired-state model for each client.
- [x] T038 Implement diff planner from current client config to desired state.
- [x] T039 Implement no-op detection to skip unnecessary writes.
- [x] T040 Implement per-command dry-run planner output.
- [x] T041 Implement conflict handling when requested server is missing.
- [x] T042 Implement reconciler that applies changes through adapters safely.

## Step 7 - Build Safety Layer

- [x] T043 Implement backup snapshot before every mutating client write.
- [x] T044 Implement backup metadata (client, path, timestamp, command, hash).
- [x] T045 Implement rollback target selection by timestamp or latest.
- [x] T046 Implement automatic restore attempt when write validation fails.
- [x] T047 Implement failure-safe exit codes for partial-recovery cases.
- [x] T048 Implement retention rules for cleanup of old backups.

## Step 8 - Build Adapter Platform

- [x] T049 Define adapter interface: `Detect`, `Read`, `Apply`, `Write`, `Validate`.
- [x] T050 Define internal normalized client config model.
- [x] T051 Build shared parse/write helpers for JSON preservation.
- [x] T052 Build adapter registry and dispatch by client name.
- [x] T053 Build unknown-key preservation helpers to avoid data loss.
- [x] T054 Build adapter test harness used by all client adapters.

## Step 9 - Implement Claude Code Adapter

- [x] T055 Capture real Claude Code config fixtures for normal and edge cases.
- [x] T056 Implement Claude Code detect/read/apply/write/validate flow.
- [x] T057 Add tests for round-trip safety and unknown-key preservation.
- [x] T058 Verify enable/disable behavior with dry-run and real write modes.

## Step 10 - Implement Cursor Adapter

- [x] T059 Capture real Cursor config fixtures for normal and edge cases.
- [x] T060 Implement Cursor detect/read/apply/write/validate flow.
- [x] T061 Add tests for round-trip safety and unknown-key preservation.
- [x] T062 Verify enable/disable behavior with dry-run and real write modes.

## Step 11 - Implement Claude Desktop Adapter

- [x] T063 Capture real Claude Desktop config fixtures for normal and edge cases.
- [x] T064 Implement Claude Desktop detect/read/apply/write/validate flow.
- [x] T065 Add tests for round-trip safety and unknown-key preservation.
- [x] T066 Verify enable/disable behavior with dry-run and real write modes.

## Step 12 - Implement Codex Adapter

- [x] T067 Capture real Codex config fixtures for normal and edge cases.
- [x] T068 Implement Codex detect/read/apply/write/validate flow.
- [x] T069 Add tests for round-trip safety and unknown-key preservation.
- [x] T070 Verify enable/disable behavior with dry-run and real write modes.

## Step 13 - Implement OpenCode Adapter

- [x] T071 Capture real OpenCode config fixtures for normal and edge cases.
- [x] T072 Implement OpenCode detect/read/apply/write/validate flow.
- [x] T073 Add tests for round-trip safety and unknown-key preservation.
- [x] T074 Verify enable/disable behavior with dry-run and real write modes.

## Step 14 - Implement Profile System

- [x] T075 Implement `profile create` with server existence validation.
- [x] T076 Implement `profile apply` to update all selected clients.
- [x] T077 Implement `profile list` with active profile marker.
- [x] T078 Implement `profile delete` with active-profile safety behavior.
- [x] T079 Implement profile application dry-run preview.
- [x] T080 Implement profile apply rollback behavior on partial failure.

## Step 15 - Implement Doctor and Diagnostics

- [x] T081 Implement `doctor` check for missing client config files.
- [x] T082 Implement `doctor` check for invalid JSON or invalid schema.
- [x] T083 Implement `doctor` check for permission/write access issues.
- [x] T084 Implement `doctor` check for missing server commands in PATH.
- [x] T085 Implement actionable fix suggestions for each failed check.

## Step 16 - Implement Output and UX Rules

- [x] T086 Define one consistent human-readable output style.
- [x] T087 Define one consistent machine-readable JSON output schema.
- [x] T088 Implement clear error messages with cause and fix action.
- [x] T089 Implement command summaries after successful writes.
- [x] T090 Implement diff summary output for `--dry-run`.
- [x] T091 Implement quiet/noise control for `--verbose` and normal mode.

## Step 17 - Implement Test Coverage

- [x] T092 Add unit tests for config schema validation.
- [x] T093 Add unit tests for server/profile CRUD.
- [x] T094 Add unit tests for planner/diff/reconciler behavior.
- [x] T095 Add unit tests for backup and rollback logic.
- [x] T096 Add integration tests for each adapter using fixtures.
- [x] T097 Add mutation/failure tests for write interruption scenarios.
- [x] T098 Add tests for idempotency of all mutating commands.
- [x] T099 Add tests for dry-run no-write guarantees.
- [x] T100 Add golden tests for CLI text output.
- [x] T101 Add golden tests for CLI JSON output.

## Step 18 - CI, Build, and Release Setup

- [x] T102 Add CI workflow for lint, unit tests, integration tests.
- [x] T103 Add CI matrix for macOS, Linux, and Windows.
- [x] T104 Add race detector and strict failure gates in CI.
- [x] T105 Add reproducible build configuration.
- [x] T106 Add goreleaser config for multi-platform binaries.
- [x] T107 Add checksum generation and verification.
- [x] T108 Add release notes template and changelog workflow.
- [x] T109 Add versioning policy for alpha, beta, and stable tags.

## Step 19 - Documentation

- [x] T110 Write `README.md` with mission, install, and quickstart.
- [x] T111 Write `docs/commands.md` with complete command and flag reference.
- [x] T112 Write `docs/clients.md` with per-client path and behavior details.
- [x] T113 Write `docs/config-schema.md` for canonical config format.
- [x] T114 Write `docs/safety.md` for backup, rollback, and failure behavior.
- [x] T115 Write `docs/troubleshooting.md` for common errors and fixes.
- [x] T116 Write `docs/examples.md` with real workflow scenarios.
- [x] T117 Write `docs/contributing.md` with local dev setup and test flow.
- [x] T118 Write `docs/architecture.md` with core and adapter design.

## Step 20 - Launch Readiness

- [x] T119 Run full local test suite and fix all failures.
- [x] T120 Run cross-client smoke tests with real user configs.
- [x] T121 Validate all mutating commands create backups every time.
- [x] T122 Validate rollback works after forced failed writes.
- [x] T123 Validate docs by following quickstart from zero.
- [x] T124 Cut first public release and collect external feedback.

## Definition of Done for v1

- [x] D001 All v1 commands work as specified.
- [x] D002 All five v1 client adapters are stable.
- [x] D003 Profiles are reliable and reversible.
- [x] D004 Backups and rollbacks are proven under failure tests.
- [x] D005 CI is green on macOS, Linux, and Windows.
- [x] D006 Quickstart works without manual JSON editing.

## Start Here Right Now

- [x] S001 Execute Step 1 completely.
- [x] S002 After Step 1 is done, execute Step 2 completely.
- [x] S003 Continue top-to-bottom until Step 20 is done.
