# mcpup

Manage MCP servers across multiple AI clients from one CLI.

## What it does

- Keeps one canonical MCP registry in `~/.mcpup/config.json`
- Applies per-client server and per-tool enable/disable state
- Switches profile presets (`coding`, `writing`, `debug`, etc.)
- Writes directly to native client config files (no proxy)
- Creates backups before mutating client config files

## Supported v1 clients

- Claude Code
- Cursor
- Claude Desktop
- Codex
- OpenCode

## Install

### Build locally

```bash
make build
./bin/mcpup --help
```

### Go install

```bash
go install ./cmd/mcpup
```

## Quickstart

```bash
# Initialize canonical config
mcpup init

# Add a server definition
mcpup add github --command "npx -y @modelcontextprotocol/server-github" --env GITHUB_TOKEN=${GITHUB_TOKEN}

# Enable server for one client
mcpup enable github --client cursor

# Disable a specific tool for that server/client
mcpup disable github --client cursor --tool delete_issue

# Create and apply profile
mcpup profile create coding --servers github
mcpup profile apply coding

# Diagnostics and rollback
mcpup doctor
mcpup rollback --client cursor
```

## Core commands

- `mcpup init [--import]`
- `mcpup add <name> --command <cmd> [--arg ...] [--env KEY=VALUE ...]`
- `mcpup remove <name>`
- `mcpup enable <name> --client <client> [--tool <tool> ...]`
- `mcpup disable <name> --client <client> [--tool <tool> ...]`
- `mcpup list [--client <client>]`
- `mcpup status`
- `mcpup profile create|apply|list|delete`
- `mcpup clients list`
- `mcpup doctor`
- `mcpup rollback --client <client> [--to <timestamp>]`

## Global flags

- `--dry-run`
- `--json`
- `--verbose`
- `--yes`

## Development

```bash
make fmt
make test
make build
```

See `docs/` for architecture, command details, safety, and troubleshooting.
