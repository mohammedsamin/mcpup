# Contributing

## Prerequisites

- Go 1.26+
- make

## Setup

```bash
git clone <repo>
cd MCPUP
make test
```

## Development Loop

```bash
make fmt
make test
make build
./bin/mcpup --help
```

## Testing Expectations

- Add/extend unit tests for changed packages
- Keep adapter fixture tests passing
- Keep CLI golden tests updated if output contract changes

## Pull Requests

- Explain behavior changes and motivation
- Update docs when command behavior changes
- Keep changelog entries under `## [Unreleased]`
