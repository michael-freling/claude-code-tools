# Claude Code Tools

A collection of CLI tools for working with Claude Code.

> `claude-forge` (the sandboxed Docker runner for Claude Code) lives in its own
> repository at
> [`michael-freling/claude-forge`](https://github.com/michael-freling/claude-forge).

## claude-hooks

A hook system for Claude Code that controls which tools can run and under what
conditions. It reads a `PreToolUse` payload from stdin and evaluates a set of
rules, returning exit code `0` to allow a tool call or `2` to block it.

Built-in rules cover:

- Branch protection — block direct pushes/merges to protected branches
- `--no-verify` detection — block commits/pushes that skip hooks
- PR merge restrictions
- GitHub ruleset enforcement

### Installation

```bash
go install github.com/michael-freling/claude-code-tools/cmd/claude-code-hooks@latest
```

### Usage

Wire it into Claude Code's `settings.json` as a `PreToolUse` hook:

```json
{
  "hooks": {
    "PreToolUse": [
      { "command": "claude-hooks pre-tool-use" }
    ]
  }
}
```

## update-ci-secrets

Sets the `CLAUDE_CODE_OAUTH_TOKEN` GitHub Actions secret so the Claude Code PR
review workflow can authenticate. It resolves the OAuth token from your local
Claude Code credentials (or `--oauth-token`) and stores it via the `gh` CLI.

### Installation

```bash
go install github.com/michael-freling/claude-code-tools/cmd/update-ci-secrets@latest
```

### Usage

```bash
# Resolve the token from ~/.claude/.credentials.json (or the
# ANTHROPIC_API_KEY / CLAUDE_CODE_OAUTH_TOKEN env vars) and set the secret
# on the current repository.
update-ci-secrets --from-credentials

# Set a token explicitly on a specific repository.
update-ci-secrets --repo owner/name --oauth-token "$TOKEN"
```

Token resolution order (via `internal/auth`):

1. `ANTHROPIC_API_KEY` environment variable
2. `CLAUDE_CODE_OAUTH_TOKEN` environment variable
3. `~/.claude/.credentials.json`

## Development

```bash
# Build and test everything
go build ./...
go test ./...

# Coverage (CI enforces 90%, excluding generated mocks)
./scripts/check-coverage.sh
```

## License

MIT
