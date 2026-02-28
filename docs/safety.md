# Safety Model

## Backup-First Writes

Before mutating a client config, `mcpup` snapshots file contents to:

`~/.mcpup/backups/<client>/<timestamp>.bak`

And writes metadata:

`~/.mcpup/backups/<client>/<timestamp>.meta.json`

Metadata includes:

- client
- source path
- command
- timestamp
- sha256 hash

## Atomic Canonical Writes

Canonical config writes use temp-file + rename semantics.

## Validation and Recovery

After client write, adapter validation runs.

If write or validation fails:

- restore from backup is attempted automatically
- partial recovery exit semantics are returned

## Rollback Command

```bash
mcpup rollback --client <client> [--to <timestamp>]
```

- no timestamp: restore latest backup
- with timestamp: restore specific backup

## Retention

Backup manager supports cleanup retention policy to keep newest N snapshots.
