# Claude Code Guidelines for claude-code-tools

## Scope

This repository holds standalone Claude Code tools. `claude-forge` (the
sandboxed Docker runner and its MCP servers) lives in a separate repository:
`github.com/michael-freling/claude-forge`.

Tools here:

- `cmd/claude-code-hooks` — the `claude-hooks` PreToolUse hook engine
  (`internal/hooks`, `internal/command`)
- `cmd/update-ci-secrets` — provisions the `CLAUDE_CODE_OAUTH_TOKEN` GitHub
  Actions secret (`internal/cisecrets`, `internal/auth`)

## Testing

```bash
go build ./...
go test ./...
```

E2E tests use the `e2e` build tag and require external CLIs in PATH:

```bash
./scripts/run-e2e-tests.sh
```

## Coverage threshold

CI enforces 90% coverage (excluding generated mocks). When adding new exported
functions, add corresponding tests before pushing.
