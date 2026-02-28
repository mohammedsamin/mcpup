# Commands

## Global Flags

- `--dry-run`: plan actions but do not write files.
- `--json`: output machine-readable JSON.
- `--verbose`: include detailed command data.
- `--yes`: skip interactive confirmations for destructive or multi-client actions.

## interactive mode

```bash
mcpup
```

When run with no arguments in an interactive terminal, `mcpup` opens the menu wizard.

## init

```bash
mcpup init [--import]
```

Creates canonical config if missing. With `--import`, imports discovered client states into canonical registry.

## add

```bash
mcpup add <name> --command <cmd> [--arg <arg> ...] [--env KEY=VALUE ...] [--description <text>]
mcpup add <name> [--env KEY=VALUE ...] [--description <text>]
```

Adds a server definition in canonical config. Use `--update` to overwrite an existing server.
If `--command` is omitted and `<name>` exists in the built-in registry, command/args are auto-filled.

## update

```bash
mcpup update [server ...] [--yes]
```

Refreshes existing server definitions from the built-in registry (command, args, description).
Only affects servers that already exist in your canonical config and are known in the registry.

## remove

```bash
mcpup remove <name>
```

Removes server definition and related client/profile references.

## enable / disable

```bash
mcpup enable <name> --client <client> [--tool <tool> ...]
mcpup disable <name> --client <client> [--tool <tool> ...]
```

- Without `--tool`: toggles server-level enabled state for the target client.
- With `--tool`: toggles specific tools in per-tool lists.

## list

```bash
mcpup list [--client <client>]
```

Lists canonical servers. With `--client`, includes client-specific enabled/tool state.

## status

```bash
mcpup status
```

Shows active profile and high-level server status per client.

## export

```bash
mcpup export [--servers a,b,c] [--output <path>]
```

Exports server definitions from canonical config to JSON.  
Without `--output`, writes JSON to stdout.

## import

```bash
mcpup import <file> [--overwrite] [--yes]
```

Imports server definitions from an export JSON file.
- default: existing servers are skipped
- with `--overwrite`: existing servers are replaced

## registry

```bash
mcpup registry [query]
```

Lists built-in server templates. With an optional query, filters by name, description, or category.

## profile create

```bash
mcpup profile create <name> --servers a,b,c
```

Creates or updates a profile after server validation.

## profile apply

```bash
mcpup profile apply <name> [--client <client> ...] [--yes]
```

Applies profile across all supported clients through the reconciler. Supports dry-run preview and rollback behavior on partial failure.

## profile list

```bash
mcpup profile list
```

Lists profiles with active marker.

## profile delete

```bash
mcpup profile delete <name>
```

Deletes profile (idempotent). If active, active profile is cleared.

## clients list

```bash
mcpup clients list
```

Prints v1 supported clients.

## completion

```bash
mcpup completion <bash|zsh|fish>
```

Prints a shell completion script.

## doctor

```bash
mcpup doctor
```

Runs diagnostics:

- canonical config existence and schema
- per-client config detection and parseability
- write permission checks
- server command executable lookup in PATH

## rollback

```bash
mcpup rollback --client <client> [--to <timestamp>]
```

Restores the latest or selected backup snapshot for a client.
