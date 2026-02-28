# Client Behavior

## Claude Code (`claude-code`)

- Detect path order:
  - `<workspace>/.mcp.json`
  - `~/.claude/settings.json`
- Reads/writes top-level `mcpServers` object.

## Cursor (`cursor`)

- Detect path order:
  - `<workspace>/.cursor/mcp.json`
  - `~/.cursor/mcp.json`
- Reads/writes top-level `mcpServers` object.

## Claude Desktop (`claude-desktop`)

- Detect path by OS:
  - macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
  - Linux: `~/.config/Claude/claude_desktop_config.json`
  - Windows: `%APPDATA%/Claude/claude_desktop_config.json`
- Reads/writes top-level `mcpServers` object.

## Codex (`codex`)

- Detect path order:
  - `<workspace>/.codex/config.toml`
  - `~/.codex/config.toml`
- Preserves unknown TOML content.
- Uses managed block markers for safe MCP state writes:
  - `# mcpup:begin`
  - `# mcpup:end`

## OpenCode (`opencode`)

- Detect path order:
  - `<workspace>/opencode.json`
  - `~/.config/opencode/opencode.json`
- Reads/writes nested `mcp.servers` object.

## Preservation Policy

For JSON-based adapters, unknown keys are preserved at top-level and server-level where feasible.
For Codex, unknown file content is preserved and MCP state is isolated in a managed block.
