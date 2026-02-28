# Troubleshooting

## `usage error`

Cause: invalid command shape or missing required flags.

Fix:

- run `mcpup --help`
- run command with `--verbose` to inspect parsed fields

## `server not found`

Cause: enabling/disabling unknown canonical server.

Fix:

1. `mcpup list`
2. `mcpup add <name> --command <cmd>`

## `unsupported client`

Cause: unsupported `--client` value.

Fix:

- run `mcpup clients list`
- use one of: `claude-code`, `cursor`, `claude-desktop`, `codex`, `opencode`, `windsurf`, `zed`, `continue`

## `doctor detected failure checks`

Cause: one or more diagnostics failed.

Fix:

1. run `mcpup doctor --json`
2. inspect failed checks and suggestions
3. repair config or permissions

## rollback failed

Cause: backup timestamp not found or source path permission issue.

Fix:

1. inspect backup directory `~/.mcpup/backups/<client>/`
2. run `mcpup rollback --client <client>` without `--to`
