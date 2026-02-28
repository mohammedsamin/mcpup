# Config Schema

Canonical path: `~/.mcpup/config.json`

## Versioning

- Current schema version: `1`
- Unknown newer versions are rejected.
- Older versions require explicit migrators before load/write.

## Top-Level Shape

```json
{
  "version": 1,
  "servers": {},
  "clients": {},
  "profiles": {},
  "activeProfile": ""
}
```

## `servers`

Map key: server name (`^[a-z0-9][a-z0-9-]{0,62}$`, reserved names blocked)

```json
{
  "github": {
    "command": "npx -y @modelcontextprotocol/server-github",
    "args": ["--foo"],
    "env": {
      "GITHUB_TOKEN": "${GITHUB_TOKEN}"
    },
    "description": "GitHub MCP server"
  }
}
```

Env values support:
- Literal value, for example `"abc123"`
- Reference value, for example `"${GITHUB_TOKEN}"`

## `clients`

Map key: client name (`claude-code`, `cursor`, `claude-desktop`, `codex`, `opencode`)

Each client stores per-server state:

```json
{
  "cursor": {
    "servers": {
      "github": {
        "enabled": true,
        "enabledTools": ["search_issues"],
        "disabledTools": ["delete_issue"]
      }
    }
  }
}
```

Rules:
- Referenced server must exist in `servers`.
- A tool cannot appear in both `enabledTools` and `disabledTools`.

## `profiles`

Profile structure:

```json
{
  "coding": {
    "servers": ["github", "postgres"],
    "tools": {
      "github": {
        "enabled": ["search_issues"],
        "disabled": ["delete_issue"]
      }
    }
  }
}
```

Rules:
- Profile name pattern: `^[a-z0-9][a-z0-9-_]{0,62}$`
- Referenced servers must exist in `servers`.

## `activeProfile`

- Optional string.
- If set, it must reference an existing profile key.

