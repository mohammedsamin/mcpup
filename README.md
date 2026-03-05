<div align="center">

# mcpup

**One CLI to manage MCP servers across all your AI clients.**

Add a server once, enable it everywhere. No more editing 5 config files.

[![CI](https://github.com/mohammedsamin/mcpup/actions/workflows/ci.yml/badge.svg)](https://github.com/mohammedsamin/mcpup/actions/workflows/ci.yml)
[![Release](https://github.com/mohammedsamin/mcpup/releases/latest/badge.svg)](https://github.com/mohammedsamin/mcpup/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/mohammedsamin/mcpup)](https://goreportcard.com/report/github.com/mohammedsamin/mcpup)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

</div>

---

## The Problem

You use Claude Code, Cursor, Claude Desktop, Codex, and OpenCode. You want the GitHub MCP server on all of them. That means editing **5 different config files** in 5 different formats in 5 different locations.

Add Notion? Edit 5 files again. Disable Playwright? 5 more edits. Switch to a "work" profile? Manually update every single one.

## The Solution

```bash
mcpup add github
mcpup enable github --client cursor
mcpup enable github --client claude-code
mcpup enable github --client claude-desktop
```

Or just run `mcpup` and use the interactive wizard:

```
  mcpup  MCP configuration manager

? What would you like to do?
  → Add a server
    Remove a server
    Enable / Disable a server
    List servers
    Browse server registry
    Status overview
    Profiles
    Run doctor
    Rollback a client
    Exit
```

## Install

### Homebrew (macOS/Linux)

```bash
brew tap mohammedsamin/tap
brew install mcpup
```

### Go

```bash
go install github.com/mohammedsamin/mcpup/cmd/mcpup@latest
```

### Binary

Download from [Releases](https://github.com/mohammedsamin/mcpup/releases/latest) — available for macOS, Linux, and Windows (amd64 + arm64).

## Quickstart

### Interactive mode (recommended)

Just run `mcpup` with no arguments:

```bash
mcpup
```

The wizard walks you through everything with arrow-key navigation. First time? It'll offer to import your existing servers automatically.

### CLI mode (for power users)

```bash
# One-command onboarding (interactive)
mcpup setup

# Add from the built-in registry (knows the command, args, everything)
mcpup add github --env GITHUB_TOKEN=ghp_xxx

# Add a custom server
mcpup add my-server --command npx --arg -y --arg my-mcp-package

# Add a remote HTTP/SSE server
mcpup add my-remote --url https://api.example.com/mcp --header "Authorization:Bearer sk-xxx"

# Enable on clients
mcpup enable github --client cursor
mcpup enable github --client claude-code

# Disable a specific tool
mcpup disable github --client cursor --tool delete_issue

# Check what's configured
mcpup list
mcpup status

# Switch profiles
mcpup profile create work --servers github,notion,slack
mcpup profile apply work

# Diagnostics
mcpup doctor

# Undo mistakes
mcpup rollback --client cursor
```

## Built-in Server Registry

97 curated MCP servers ready to install — no need to look up package names or commands:

```bash
mcpup registry
```

```
NAME                 CATEGORY      DESCRIPTION
brave-search         search        Brave Search - web and local search
context7             search        Context7 - up-to-date docs for any library
elevenlabs           media         ElevenLabs - text-to-speech, voice generation
everart              media         EverArt - AI image generation
fetch                search        Fetch - retrieve and convert web pages to markdown
filesystem           utility       Filesystem - read, write, search files
github               developer     GitHub - search repos, issues, PRs, create branches
gitlab               developer     GitLab - manage repos, issues, merge requests
google-maps          productivity  Google Maps - geocode, directions, places
linear               developer     Linear - manage issues and projects
memory               utility       Memory - persistent knowledge graph for context
notion               productivity  Notion - search and manage pages, databases
playwright           automation    Playwright - browser automation, screenshots, testing
postgres             database      PostgreSQL - query and explore databases
puppeteer            automation    Puppeteer - headless Chrome automation
sentry               developer     Sentry - search errors, get issue details
sequential-thinking  utility       Sequential Thinking - step-by-step reasoning helper
slack                productivity  Slack - read/send messages, manage channels
supabase             database      Supabase - manage projects, tables, edge functions
```

Add any of them with just the name:

```bash
mcpup add github --env GITHUB_TOKEN=ghp_xxx
mcpup add notion --env NOTION_TOKEN=ntn_xxx
mcpup add playwright
mcpup add memory
```

## Supported Clients

| Client | Config Format | Config Location |
|--------|--------------|-----------------|
| **Claude Code** | JSON | `~/.claude/settings.json` |
| **Cursor** | JSON | `~/.cursor/mcp.json` |
| **Claude Desktop** | JSON | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| **Codex** | TOML | `~/.codex/config.toml` |
| **OpenCode** | JSON | `~/.config/opencode/opencode.json` |
| **Windsurf** | JSON | `~/.codeium/windsurf/mcp_config.json` |
| **Zed** | JSON/JSONC | `~/.zed/settings.json` (macOS), `~/.config/zed/settings.json` (Linux) |
| **Continue (VS Code)** | JSON | `~/.continue/mcpServers/mcpup.json` |

mcpup writes directly to each client's native config file. No proxy, no middleware, no daemon.

## How It Works

```
~/.mcpup/config.json          ← single source of truth
        │
        ├─→ ~/.claude/settings.json         (Claude Code)
        ├─→ ~/.cursor/mcp.json              (Cursor)
        ├─→ ~/Library/.../claude_desktop_config.json  (Claude Desktop)
        ├─→ ~/.codex/config.toml            (Codex)
        ├─→ ~/.config/opencode/opencode.json (OpenCode)
        ├─→ ~/.codeium/windsurf/mcp_config.json (Windsurf)
        ├─→ ~/.zed/settings.json            (Zed)
        └─→ ~/.continue/mcpServers/mcpup.json (Continue)
```

1. You define servers once in `~/.mcpup/config.json`
2. You enable/disable per client
3. mcpup writes the native config file for each client
4. Every write creates a backup first
5. If the write fails, it auto-restores from backup
6. Manual client entries that mcpup does not own stay untouched

## Features

- **One config, all clients** — define a server once, enable it on any client
- **Preserve unmanaged entries** — leaves manual client-only MCP servers untouched
- **HTTP/SSE transport** — manage remote MCP servers via URL, not just local commands
- **Interactive wizard** — arrow-key menu for everything, no commands to memorize
- **Setup command** — guided onboarding to select clients, servers, and required keys
- **Built-in registry** — 97 curated servers with pre-filled commands and args
- **Registry preflight** — catches missing launchers and required env before writes
- **Update command** — refresh registry-backed server definitions with `mcpup update`
- **Export / Import** — share server packs with `mcpup export` and `mcpup import`
- **Profiles** — switch between "work", "personal", "debug" setups in one command
- **Per-tool control** — enable a server but disable specific tools on specific clients
- **Shell completion** — generate completions for bash/zsh/fish
- **Auto-backup** — every config write creates a timestamped backup
- **Rollback** — restore any client config from any backup with one command
- **Doctor** — diagnose config issues, ownership, drift, missing executables, URL validation
- **Dry-run** — preview changes without writing anything (`--dry-run`)
- **JSON output** — pipe to `jq` for scripting (`--json`)
- **Zero dependencies** — single binary, no runtime requirements
- **Import existing** — `mcpup init --import` discovers servers already configured in your clients

## Commands

| Command | Description |
|---------|-------------|
| `mcpup` | Launch interactive wizard |
| `mcpup init [--import]` | Initialize config (optionally import existing) |
| `mcpup setup` | Guided onboarding across clients and servers |
| `mcpup add <name>` | Add server (from registry, custom `--command`, or remote `--url`) |
| `mcpup update [name...]` | Refresh registry-backed server definitions |
| `mcpup remove <name>` | Remove a server |
| `mcpup enable <name> --client <c>` | Enable server on a client |
| `mcpup disable <name> --client <c>` | Disable server on a client |
| `mcpup list [--client <c>]` | List configured servers |
| `mcpup status` | Show overview of all clients |
| `mcpup export [--servers a,b]` | Export server definitions as JSON |
| `mcpup import <file>` | Import server definitions from JSON |
| `mcpup completion <shell>` | Generate shell completion script |
| `mcpup registry [query]` | Browse built-in server catalog |
| `mcpup profile create <n> --servers a,b` | Create a profile |
| `mcpup profile apply <name>` | Apply profile to all clients |
| `mcpup profile list` | List profiles |
| `mcpup profile delete <name>` | Delete a profile |
| `mcpup doctor` | Run diagnostics |
| `mcpup rollback --client <c>` | Restore from backup |
| `mcpup clients list` | Show supported clients |

### Global Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Preview changes without writing |
| `--json` | Output as JSON |
| `--verbose` | Show detailed output |
| `--yes` | Skip confirmation prompts |

## Examples

### Add GitHub and enable on all clients

```bash
mcpup setup --server github --env GITHUB_TOKEN=ghp_xxx

# or manual mode:
mcpup add github --env GITHUB_TOKEN=ghp_xxx
mcpup enable github --client claude-code
mcpup enable github --client cursor
mcpup enable github --client claude-desktop
mcpup enable github --client codex
mcpup enable github --client opencode
```

### Add a remote HTTP/SSE server

```bash
# Add a remote MCP server with authentication
mcpup add company-server --url https://mcp.company.com/sse --header "Authorization:Bearer token123"

# Specify transport explicitly
mcpup add my-api --url https://api.example.com/mcp --transport streamable-http

# Enable on clients — writes url/headers instead of command/args
mcpup enable company-server --client cursor
mcpup enable company-server --client claude-code
```

### Create a work profile

```bash
mcpup add github --env GITHUB_TOKEN=ghp_xxx
mcpup add slack --env SLACK_BOT_TOKEN=xoxb-xxx
mcpup add notion --env NOTION_TOKEN=ntn_xxx
mcpup add sentry --env SENTRY_AUTH_TOKEN=sntrys_xxx

mcpup profile create work --servers github,slack,notion,sentry
mcpup profile apply work --yes
```

### Dry-run to preview

```bash
mcpup enable github --client cursor --dry-run
```

### JSON output for scripting

```bash
mcpup list --json | jq '.data.servers[].name'
mcpup status --json | jq '.data.clients'
```

### Rollback a mistake

```bash
mcpup rollback --client cursor
# picks latest backup, or specify: --to 20260228T103000.000000000Z+0000
```

## Architecture

```
cmd/mcpup/          → entry point
internal/
  cli/              → command routing + interactive wizard
  registry/         → built-in server catalog (97 servers)
  store/            → config read/write with schema validation
  planner/          → diff engine (current state → desired state)
  core/             → reconciler (backup → write → validate → rollback)
  adapters/         → per-client config writers
    claudecode/     → Claude Code adapter (JSON)
    cursor/         → Cursor adapter (JSON)
    claudedesktop/  → Claude Desktop adapter (JSON)
    codex/          → Codex adapter (TOML)
    opencode/       → OpenCode adapter (JSON)
    windsurf/       → Windsurf adapter (JSON)
    zed/            → Zed adapter (JSON/JSONC)
    continuedev/    → Continue adapter (JSON)
  profile/          → profile create/apply/delete
  backup/           → snapshot + restore + retention
  validate/         → doctor diagnostics
  output/           → colored output, tables, interactive prompts
```

**Zero external dependencies.** Everything is Go standard library — ANSI colors, terminal raw mode, table rendering, interactive prompts — all built from scratch.

## Contributing

```bash
git clone https://github.com/mohammedsamin/mcpup.git
cd mcpup
go test ./...
go build -o bin/mcpup ./cmd/mcpup
./bin/mcpup
```

See [`docs/contributing.md`](docs/contributing.md) for details.

## License

[MIT](LICENSE)
