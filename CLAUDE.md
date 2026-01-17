# Agent Instructions

## Philosophy

- Main user is AI coding agents — design for agent ergonomics first
- Unix philosophy — task management for AI agents, nothing else
- Keep it simple — resist feature bloat
- Labels are markers, not workflow — no enforcement, just conventions
- No agent identity — track tasks, not who's working on them

## Development

Use `make` for all build and test operations. Do not run `go build` directly.

```
make build    # builds to bin/tlog with version info
make test     # run tests
make check    # fmt, lint, vet, staticcheck, test
make clean    # remove artifacts
```

Run `make help` for all available targets.

## Task Tracking with tlog

This project uses `tlog` for task management. Use it to track work across sessions.

Run `tlog prime` to get started.
