# Scripts

## reproducible-build.sh

Builds cross-platform binaries with reproducible flags and generates `checksums.txt`.

```bash
scripts/reproducible-build.sh v0.1.0
```

## verify-checksums.sh

Verifies checksums for generated artifacts.

```bash
scripts/verify-checksums.sh dist/repro dist/repro/checksums.txt
```

## smoke-preserve-unmanaged.sh

Verifies that `mcpup` preserves manual client entries it does not own while still removing managed entries.

```bash
make build
scripts/smoke-preserve-unmanaged.sh
```

## smoke-import-rollback-ownership.sh

Checks that `init --import` skips unknown external servers and that rollback sync keeps those entries unmanaged.

```bash
make build
scripts/smoke-import-rollback-ownership.sh
```
