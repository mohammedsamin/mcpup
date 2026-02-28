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
