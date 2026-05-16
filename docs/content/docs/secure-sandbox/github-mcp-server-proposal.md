---
title: "Proposal: GitHub MCP Server for claude-forge"
weight: 4
---

# Proposal: GitHub MCP Server for claude-forge

**Status**: Draft
**Depends on**: [Secure Sandbox Architecture]({{< relref "/docs/secure-sandbox/secure-sandbox-architecture" >}})

## 1. Motivation

Today, the agent container interacts with GitHub through two mechanisms:

1. **Git transport** (push/pull/fetch): The agent's gitconfig rewrites `https://github.com/` to `http://gateway:8080/github.com/`, and the gateway's HTTP proxy (`internal/gateway/proxy.go`) injects the credential and forwards to GitHub. This is transparent to git and works well.

2. **GitHub API** (PRs, issues, checks): A custom `forge-gh` binary in the agent container emulates a subset of `gh` CLI syntax. It parses CLI args, fetches a schema from the gateway's REST API server (`:8083`), maps commands to operations, and the gateway proxies to `api.github.com` with authentication.

The `forge-gh` approach has several problems:

- **Limited coverage**: Only 14 operations are wired up. Every new GitHub API operation requires changes to three places: the operation list in `api.go`, the command-to-operation mapping in `forgegh/client.go`, and the request body builder.
- **Fragile CLI emulation**: `forge-gh` parses `gh`-style arguments through manual string matching. It doesn't support `gh api` (raw API calls), JSON output formatting (`--json`), or many flags that the real `gh` supports.
- **Discovery friction**: Claude Code must use `gh`-style commands that may not match what it was trained on. The schema endpoint provides some discoverability, but Claude can't introspect available operations the way it can with MCP tools.
- **No structured responses**: Results come back as printed JSON to stdout. Claude parses this as text rather than receiving structured tool results.

Claude Code has native MCP (Model Context Protocol) support. Exposing GitHub operations as MCP tools would let Claude discover, invoke, and receive structured results from GitHub operations without any wrapper binary.

## 2. Goals and Non-Goals

### Goals

- Replace the `forge-gh` REST API with an MCP server running in the gateway container.
- Claude Code discovers available GitHub tools via MCP protocol at startup.
- All current `forge-gh` operations continue to work via MCP tools.
- Additional operations (labels, reviews, review comments, workflow dispatch, file contents, branch management) become trivial to add — just define the tool schema.
- Policy enforcement (owner/repo allowlist, read vs. write) is preserved.
- The git HTTP proxy remains unchanged — MCP replaces only the API layer.
- The agent container no longer needs the `forge-gh` binary.

### Non-Goals

- Replacing the git transport proxy. Git push/pull/fetch works well over the HTTP proxy and doesn't benefit from MCP.
- Building a general-purpose GitHub MCP server. This is scoped to the gateway's allowlisted repo(s).
- Supporting GitHub Enterprise Server in v1 (though the architecture doesn't preclude it).

## 3. Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Host                                                                       │
│                                                                             │
│  claude-forge start                                                         │
│    ├─ writes MCP config to agent's .claude/settings.json                    │
│    │    { "mcpServers": { "github": { "url": "http://gateway:8083/mcp" }}}  │
│    └─ starts containers                                                     │
│                                                                             │
│  ┌──────────────────────────┐    ┌───────────────────────────────────────┐  │
│  │ Agent container          │    │ Gateway container                     │  │
│  │                          │    │                                       │  │
│  │ Claude Code              │    │ :8080  git HTTP proxy (unchanged)     │  │
│  │   ├─ git push/pull ──────────►│                                       │  │
│  │   │                      │    │ :8083  MCP server (replaces REST API) │  │
│  │   └─ MCP tool calls ─���───────►│   ├─ tools/list → available ops      │  │
│  │       (github.pr_list,   │    │   ├─ tools/call → policy check →     │  │
│  │        github.pr_create, │    │   │    forward to api.github.com      │  │
│  │        github.issue_view)│    │   └─ auth injection (Bearer token)    │  │
│  │                          │    │                                       │  │
│  │ No forge-gh binary       │    │                                       │  │
│  │ No schema fetching       │    │                                       │  │
│  └──────────────────────────┘    └───────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.1 MCP Transport

The MCP server uses **Streamable HTTP transport** (the standard for remote MCP servers). The gateway exposes a single endpoint at `http://gateway:8083/mcp` that handles the MCP protocol. Claude Code connects to it as a remote MCP server configured in the agent's settings.

### 3.2 Tool Definitions

Each GitHub operation becomes an MCP tool with a typed JSON Schema for inputs and structured JSON outputs. Example tools:

| Tool Name | Description | Inputs |
|---|---|---|
| `github_pr_list` | List pull requests | `state?`, `per_page?`, `sort?`, `direction?` |
| `github_pr_get` | Get a pull request | `number` |
| `github_pr_create` | Create a pull request | `title`, `body?`, `head`, `base` |
| `github_pr_update` | Update a pull request | `number`, `title?`, `body?`, `state?`, `base?` |
| `github_pr_merge` | Merge a pull request | `number`, `merge_method?`, `commit_title?` |
| `github_pr_comment` | Comment on a PR | `number`, `body` |
| `github_pr_reviews` | List PR reviews | `number` |
| `github_issue_list` | List issues | `state?`, `per_page?`, `labels?` |
| `github_issue_get` | Get an issue | `number` |
| `github_issue_create` | Create an issue | `title`, `body?`, `labels?` |
| `github_issue_comment` | Comment on an issue | `number`, `body` |
| `github_repo_get` | Get repository info | (none) |
| `github_release_list` | List releases | `per_page?` |
| `github_checks_list` | List check runs | `ref` |
| `github_api` | Raw GitHub API call | `method`, `path`, `body?` |

The `github_api` tool is a catch-all that allows Claude to call any GitHub API endpoint within the allowed repo, preserving the policy enforcement. This eliminates the need to pre-wire every possible operation.

### 3.3 Policy Enforcement

The MCP server enforces the same policy the REST API enforces today:

- **Owner/repo scope**: Write operations are restricted to the configured `AllowedOwner/AllowedRepo`. Read operations to other public repos are allowed.
- **Auth injection**: The MCP server adds `Authorization: Bearer <token>` to all upstream requests. The agent never sees the token.
- **No credential exposure**: MCP tool responses never include auth headers or tokens.

### 3.4 Agent Configuration

At container startup, `claude-forge` writes the MCP server configuration into the agent's Claude settings:

```json
{
  "mcpServers": {
    "github": {
      "type": "url",
      "url": "http://gateway:8083/mcp"
    }
  }
}
```

Claude Code discovers the available tools via the MCP `tools/list` method when it starts.

## 4. Advantages over forge-gh

| Dimension | forge-gh (current) | MCP server (proposed) |
|---|---|---|
| **Discoverability** | Agent must know `gh` CLI syntax | Claude sees typed tool schemas via MCP |
| **Coverage** | 14 hardcoded operations | Unlimited via `github_api` catch-all + typed shortcuts |
| **Adding operations** | 3 code changes (api.go, client.go, body builder) | 1 tool definition in the MCP handler |
| **Response format** | Unstructured JSON printed to stdout | Structured MCP tool results |
| **Agent binary** | Requires `forge-gh` binary in container | No extra binary needed |
| **Error handling** | Exit codes + stderr text | Structured MCP error responses with `isError` flag |
| **Streaming** | Not supported | MCP supports streaming for large responses |
| **Multi-repo reads** | Limited by hardcoded patterns | `github_api` allows reads to any public repo |

## 5. Implementation Plan

### 5.1 New Package: `internal/gateway/mcpserver/`

A new package implementing the MCP server:

| File | Purpose |
|---|---|
| `server.go` | MCP protocol handler (initialize, tools/list, tools/call) |
| `tools.go` | Tool definitions and JSON Schema generation |
| `github.go` | GitHub API execution (reuses auth injection pattern from current `api.go`) |
| `policy.go` | Owner/repo policy enforcement |

### 5.2 Changes to Existing Code

| File | Change |
|---|---|
| `internal/gateway/server.go` | Replace `apiServer` with MCP server on `:8083` |
| `internal/forge/orchestrator.go` | Write MCP config to agent's settings.json |
| `internal/forge/claudecode/settings.go` | Add MCP server config to settings generation |
| `docker/agent/Dockerfile` | Remove `forge-gh` binary |
| `internal/forgegh/` | Delete package (after migration) |

### 5.3 Migration Path

1. **Phase 1**: Add MCP server alongside existing REST API on a different path (`/mcp`). Both `forge-gh` and MCP work simultaneously.
2. **Phase 2**: Configure agent to use MCP. Verify all operations work. Keep `forge-gh` as fallback.
3. **Phase 3**: Remove `forge-gh` binary and REST API. MCP is the sole GitHub API interface.

## 6. MCP Protocol Details

### 6.1 Tool Call Flow

```
Claude Code                    Gateway MCP Server              GitHub API
    │                               │                              │
    ├─ POST /mcp ──────────────────►│                              │
    │  {"method": "tools/call",     │                              │
    │   "params": {                 │                              │
    │     "name": "github_pr_list", │                              │
    │     "arguments": {            │                              │
    │       "state": "open"         │                              │
    │     }                         │                              │
    │   }}                          │                              │
    │                               ├─ GET /repos/o/r/pulls ──────►│
    │                               │   Authorization: Bearer xxx  │
    │                               │   state=open                 │
    │                               │                              │
    │                               │◄── 200 [{...}, {...}] ───────┤
    │                               │                              │
    │◄─ {"result": {"content": [    │                              │
    │     {"type": "text",          │                              │
    │      "text": "[{...}]"}       │                              │
    │   ]}} ────────────────────────┤                              │
```

### 6.2 Tool Schema Example

```json
{
  "name": "github_pr_create",
  "description": "Create a pull request in the project repository",
  "inputSchema": {
    "type": "object",
    "properties": {
      "title": { "type": "string", "description": "PR title" },
      "body": { "type": "string", "description": "PR description (markdown)" },
      "head": { "type": "string", "description": "Branch containing changes" },
      "base": { "type": "string", "description": "Branch to merge into (default: main)" }
    },
    "required": ["title", "head"]
  }
}
```

### 6.3 Error Handling

MCP tool errors are returned as structured responses:

```json
{
  "result": {
    "content": [{ "type": "text", "text": "GitHub API error: 422 - head branch does not exist" }],
    "isError": true
  }
}
```

## 7. Threat Model

The security model is identical to the current gateway architecture:

| # | Concern | Defense |
|---|---|---|
| 1 | Agent tries to call operations on repos outside allowlist | MCP policy layer enforces owner/repo on write operations |
| 2 | Agent tries to extract GitHub token via MCP | Token is never included in MCP responses; auth is injected server-side |
| 3 | Agent tries to call unauthorized endpoints | `github_api` catch-all still enforces owner/repo policy for writes |
| 4 | Prompt injection via MCP tool results | Same risk as current JSON responses — Claude Code's safety layers apply |

## 8. Open Questions

1. **MCP library choice**: Use an existing Go MCP library (e.g., `mark3labs/mcp-go`) or implement the subset needed (initialize, tools/list, tools/call) directly? The protocol subset needed is small enough that a direct implementation avoids dependency risk.

2. **Pagination**: GitHub API responses can be paginated. Should the MCP server auto-paginate and return all results, or expose pagination parameters and return one page at a time? Recommendation: expose `per_page` and `page` params, let Claude decide when to paginate.

3. **Webhook events / notifications**: MCP supports server-initiated notifications. Could the gateway push PR review events to Claude? Out of scope for v1 but the transport supports it.

4. **Multiple repos**: The current model allowlists one repo for writes. Should MCP support configuring multiple repos? The `github_api` catch-all already allows reads to any repo; writes to additional repos would need config changes.

## 9. Rejected Alternatives

### 9.1 Install Real `gh` CLI in Container

Mount the host's gh config and install the real `gh` CLI. Rejected because:
- Violates Invariant 1 (agent has the credential directly).
- `gh` supports arbitrary API calls (`gh api`) with no policy enforcement.
- Token in container can be exfiltrated via prompt injection.

### 9.2 Keep forge-gh, Add More Operations

Extend the current forge-gh approach. Rejected because:
- Each new operation requires coordinated changes across 3 files.
- Claude doesn't get typed tool schemas — it must know gh CLI syntax.
- The REST API + CLI emulation pattern doesn't scale and adds maintenance burden.
- No path toward streaming, notifications, or resource exposure that MCP provides.

### 9.3 Stdio MCP Server in Agent Container

Run the MCP server as a local stdio process in the agent container. Rejected because:
- The MCP server needs the GitHub token to make API calls.
- Putting the token in the agent container violates Invariant 1.
- The gateway architecture keeps credentials in a separate container by design.

## 10. Rollout

1. Land this proposal doc.
2. Implement MCP server in `internal/gateway/mcpserver/` with tool definitions for current 14 operations + `github_api` catch-all.
3. Wire MCP server into gateway alongside existing REST API (Phase 1).
4. Update orchestrator to write MCP config to agent settings. Test that Claude uses MCP tools.
5. Remove `forge-gh` binary and REST API after confirming MCP covers all use cases (Phase 3).
6. Add e2e test: start session, verify Claude can list PRs, create PR, and comment via MCP tools.
