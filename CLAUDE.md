# Claude Code setup

This repository is configured for Claude Code Web. The SessionStart hook automatically prepares the Go workspace so you can start coding immediately.

## SessionStart hook
- Hook path: `.claude/hooks/SessionStart`
- Behavior: Runs `make deps` when a session begins to download and tidy Go modules. The hook detects the presence of `go.mod` before running.

## Development requirements
- Go 1.24.7
- make (for using the provided automation targets)

## Common commands
- `make deps` — Download and tidy Go module dependencies.
- `make test` — Run the full Go test suite.
- `make lint` — Format, vet, and test the code.
- `make build` — Compile the `site2skillgo` binary to `bin/`.

If an `AGENTS.md` file is added later, include it here using `@AGENTS.md` to keep contributor guidance in one place.
