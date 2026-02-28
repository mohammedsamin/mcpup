# mcpup

> Manage all your MCP servers across all your AI clients from one command line.

## Mission

You configure your MCP servers once. mcpup puts them where they need to be, lets you toggle them per-client, and lets you switch entire setups with one command. No apps, no RAM, no JSON editing. Ever again.

## What mcpup Is

mcpup is the config layer between your MCP servers and your AI clients. A single CLI tool that gives you full control over which tools are available to which clients, instantly, from your terminal.

## What mcpup Is NOT

- Not an app — CLI only, zero background processes
- Not a gateway/proxy — writes directly to each client's native config files
- Not a server registry — that's Smithery, mcp.run, etc.

## Three Pillars

1. **One place** — All your MCP servers registered in one config (`~/.mcpup/`)
2. **Every client** — Claude Code, Claude Desktop, Cursor, VS Code, Windsurf, Gemini CLI, Zed, OpenCode
3. **Instant control** — `mcpup enable`, `mcpup disable`, `mcpup profile coding` — changes apply immediately

## Core Features

- **Add once, use everywhere** — register an MCP server once, deploy to any client
- **Toggle anything** — enable/disable any server for any client instantly
- **Profiles** — save tool combinations ("coding", "writing", "debug") and switch with one command
- **Per-client control** — give Cursor access to GitHub but not Slack, give Claude Code everything
- **Zero overhead** — no daemon, no Electron, no cloud, no RAM usage
- **Real config writes** — mcpup writes directly to each client's config files, no proxy layer
- **Free and open source** — forever

## Supported Clients (Target)

- Claude Code
- Claude Desktop
- Cursor
- VS Code (Copilot)
- Windsurf
- Gemini CLI
- Zed
- OpenCode

## Example Usage

```sh
# Add a server once
mcpup add github --command "npx -y @modelcontextprotocol/server-github" --env GITHUB_TOKEN=xxx

# Enable it for specific clients
mcpup enable github --client cursor
mcpup enable github --client claude-code

# Disable it for one client
mcpup disable github --client cursor

# Create and switch profiles
mcpup profile create coding --servers github,context7,postgres
mcpup profile create writing --servers notion,brave-search
mcpup profile coding    # instantly switches all clients

# See everything
mcpup list              # all servers and their status per client
mcpup status            # which profile is active, what's enabled where
```

## The Gap We Fill

| Tool | What it does | What it misses |
|------|-------------|----------------|
| add-mcp | Installs MCP servers across clients | Can't remove, disable, toggle, or manage |
| MCP Router | Desktop app with toggles and workspaces | Electron app, uses compute, not CLI |
| MCP Click | Menu bar app with profiles and store | Paid ($3.99/mo), closed source, not CLI |
| mcp-toggle | Bash scripts for toggling | Fragile, no profiles, no real tool |
| **mcpup** | CLI — add, toggle, profile, manage | — |

## Tech Stack

- **Language:** Go — single binary, no runtime dependencies, instant startup, cross-platform
- **Config format:** JSON — industry standard for MCP, consistent with every client's native format
- **Install:** GitHub Releases (download one binary, run it — nothing else needed on the user's machine) + `go install` for Go devs. Curl script and Homebrew added later.

## First Target Clients (v1)

- Claude Code — terminal devs, heaviest MCP users
- Cursor — biggest AI editor, massive MCP adoption
- Claude Desktop — most complained about config pain
- Codex — OpenAI's CLI, growing fast
- OpenCode — open source crowd, credibility

## Who It's For

Developers who live in the terminal, use multiple AI clients daily, run 20+ MCP servers, and are tired of editing JSON files by hand. Install mcpup once, never touch a config file again.
