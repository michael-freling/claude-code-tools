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
review workflow can authenticate. By default it mints a **long-lived** token
(valid ~1 year) with `claude setup-token` and stores it via the `gh` CLI, so the
CI secret does not go stale.

### Installation

```bash
go install github.com/michael-freling/claude-code-tools/cmd/update-ci-secrets@latest
```

### Usage

```bash
# Default: run `claude setup-token` and upload the resulting ~1-year token
# to the current repository. This runs an interactive OAuth authorization
# flow, so complete the browser login when prompted.
update-ci-secrets

# Upload a token you already have to a specific repository.
update-ci-secrets --repo owner/name --oauth-token "$TOKEN"
```

### Why not the local subscription token?

`~/.claude/.credentials.json` holds the **interactive Claude subscription access
token**, which Claude Code rotates roughly hourly. Uploading that would make the
`CLAUDE_CODE_OAUTH_TOKEN` secret expire within the hour and the PR review
workflow would start failing until you re-ran the command. `claude setup-token`
instead mints a separate token valid for ~1 year, which is what CI needs.

`claude setup-token` prints the token to the terminal and never writes it to
`.credentials.json`. The default path runs it for you and captures the token; if
you already have one, upload it directly with `--oauth-token`.

> **Note:** because `claude setup-token` requires an interactive browser login,
> `update-ci-secrets` cannot mint a token unattended — run it from a terminal.
> Requires the `gh` CLI (authenticated with repo admin access) and, for the
> default path, the `claude` CLI.

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
