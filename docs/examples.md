# Examples

## Add once, enable per client

```bash
mcpup add github --command "npx -y @modelcontextprotocol/server-github"
mcpup enable github --client cursor
mcpup enable github --client claude-code
```

## Tool-level control

```bash
mcpup disable github --client cursor --tool delete_issue
mcpup enable github --client cursor --tool search_issues
```

## Profile workflow

```bash
mcpup profile create coding --servers github,postgres
mcpup profile create writing --servers notion
mcpup profile apply coding
mcpup status
```

## Dry-run review before writing

```bash
mcpup --dry-run enable github --client cursor
mcpup --dry-run profile apply coding
```

## Diagnostics and recovery

```bash
mcpup doctor
mcpup rollback --client cursor
```

## Update from registry

```bash
mcpup update --yes
```

## Export and import

```bash
mcpup export --output team-pack.json
mcpup import team-pack.json
```

## Shell completion

```bash
mcpup completion zsh > ~/.zsh/completions/_mcpup
```
