# E2E Testing

This directory contains end-to-end (e2e) tests that verify complete workflows by testing Claude CLI execution.

## Overview

E2E tests are separated from unit tests using Go build tags (`//go:build e2e`). We have two types of E2E tests:

1. **Mock-based tests** - Tests that use MockClaudeBuilder to simulate Claude CLI responses
   - Run in CI without requiring Claude API access
   - Fast and deterministic
   - Test argument construction, response parsing, and workflow logic

2. **Real CLI tests** - Tests that execute the actual Claude CLI
   - Skipped in CI (require Claude authentication)
   - Useful for local development and manual verification
   - Test real Claude API integration

## Test Files

### Mock-Based Tests (Run in CI)

- `workflow_e2e_test.go` - Tests workflow execution using MockClaudeBuilder
  - Argument construction and validation
  - Prompt generation for different workflow phases
  - Response parsing (streaming and non-streaming)
  - Error handling (e.g., "Prompt is too long")
  - Timeout and working directory handling

### Real CLI Tests (Skipped in CI)

- `claude_cli_e2e_test.go` - Tests using real Claude CLI
  - Simple execution and streaming modes
  - JSON schema validation
  - Working directory file access
  - Requires Claude authentication

### Helpers

- `helpers/` - Test helper utilities
  - `mock_claude.go` - MockClaudeBuilder for creating mock Claude CLI scripts
  - `claude.go` - Claude CLI detection and availability checks
  - `repo.go` - Temporary Git repository management
  - `git.go`, `gh.go` - Git and GitHub CLI test utilities
  - `cleanup.go` - Resource cleanup helpers

## Prerequisites

### Required Tools

- **git** - Required for all e2e tests
- **Go** - For running the tests

### Optional Tools

- **gh** - GitHub CLI (tests requiring gh authentication will be skipped if not available)
- **claude** - Claude Code CLI (real CLI tests will be skipped if not available)

### Tool Installation

```bash
# Install gh CLI (macOS)
brew install gh

# Install gh CLI (Linux)
# See https://cli.github.com/manual/installation

# Authenticate gh
gh auth login

# Install claude CLI
# See https://docs.anthropic.com/en/docs/claude-code
```

## Running E2E Tests

### Using the Script (Recommended)

```bash
# Run all e2e tests
./scripts/run-e2e-tests.sh

# Run with verbose output
E2E_VERBOSE=true ./scripts/run-e2e-tests.sh

# Run with custom timeout
E2E_TIMEOUT=2m ./scripts/run-e2e-tests.sh
```

### Using go test Directly

```bash
# Run all e2e tests
go test -tags=e2e ./test/e2e/...

# Run with verbose output
go test -tags=e2e -v ./test/e2e/...

# Run specific test file
go test -tags=e2e -v ./test/e2e/workflow_e2e_test.go

# Run specific test
go test -tags=e2e -v -run TestExecutor_ArgumentConstruction ./test/e2e/...
```

## Writing E2E Tests

### Using MockClaudeBuilder

MockClaudeBuilder creates a temporary mock Claude CLI script that simulates Claude's behavior. This is useful for testing in CI without requiring Claude API access.

#### Basic Usage

```go
//go:build e2e

package e2e

import (
    "context"
    "testing"

    "github.com/michael-freling/claude-code-tools/internal/workflow"
    "github.com/michael-freling/claude-code-tools/test/e2e/helpers"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMyFeature_WithMockClaude(t *testing.T) {
    // Create a mock Claude CLI that returns structured output
    mock := helpers.NewMockClaudeBuilder(t).
        WithStreamingResponse("Task completed", map[string]interface{}{
            "summary": "Feature implemented",
            "status":  "success",
        })
    claudePath := mock.Build()

    // Use the mock in your workflow
    logger := workflow.NewLogger(workflow.LogLevelNormal)
    executor := workflow.NewClaudeExecutorWithPath(claudePath, logger)

    ctx := context.Background()
    result, err := executor.ExecuteStreaming(ctx, workflow.ExecuteConfig{
        Prompt: "test prompt",
    }, nil)

    require.NoError(t, err)
    assert.NotEmpty(t, result.Output)

    // Verify what arguments were passed to Claude
    args := mock.GetCapturedArgs()
    assert.Contains(t, args, "--print")
    assert.Contains(t, args, "stream-json")
}
```

#### MockClaudeBuilder Methods

```go
// Create a new mock builder
mock := helpers.NewMockClaudeBuilder(t)

// Set response for non-streaming mode
mock.WithResponse("Simple response text")

// Set response for streaming mode with structured output
mock.WithStreamingResponse("Result message", map[string]interface{}{
    "key": "value",
})

// Simulate error conditions
mock.WithExitCode(1).
    WithStderr("Prompt is too long")

// Build the mock script (returns path to executable)
claudePath := mock.Build()

// After execution, inspect captured arguments
args := mock.GetCapturedArgs()        // All arguments
prompt := mock.GetCapturedPrompt()    // Last argument (the prompt)
```

### Using Real Claude CLI

For tests that require real Claude execution (skipped in CI):

```go
func TestRealClaude_MyFeature(t *testing.T) {
    // Skip if Claude CLI not available
    helpers.RequireClaude(t)

    logger := workflow.NewLogger(workflow.LogLevelNormal)
    executor := workflow.NewClaudeExecutor(logger)

    ctx := context.Background()
    result, err := executor.Execute(ctx, workflow.ExecuteConfig{
        Prompt:  "Reply with: PONG",
        Timeout: 30 * time.Second,
    })

    require.NoError(t, err)
    assert.Contains(t, result.Output, "PONG")
}
```

### Test Template

```go
//go:build e2e

package e2e

import (
    "testing"

    "github.com/michael-freling/claude-code-tools/test/e2e/helpers"
    "github.com/stretchr/testify/require"
)

func TestMyFeature_E2E(t *testing.T) {
    tests := []struct {
        name       string
        mockSetup  func(*helpers.MockClaudeBuilder) *helpers.MockClaudeBuilder
        wantErr    bool
        wantResult string
    }{
        {
            name: "success case",
            mockSetup: func(m *helpers.MockClaudeBuilder) *helpers.MockClaudeBuilder {
                return m.WithStreamingResponse("done", map[string]string{
                    "status": "ok",
                })
            },
            wantResult: "ok",
        },
        {
            name: "error case",
            mockSetup: func(m *helpers.MockClaudeBuilder) *helpers.MockClaudeBuilder {
                return m.WithExitCode(1).WithStderr("error occurred")
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mock := tt.mockSetup(helpers.NewMockClaudeBuilder(t))
            claudePath := mock.Build()

            // Your test logic here
        })
    }
}
```

### Skip Functions

```go
// Skip if git not available
helpers.RequireGit(t)

// Skip if gh not available
helpers.RequireGH(t)

// Skip if gh not authenticated
helpers.RequireGHAuth(t)

// Skip if claude not available
helpers.RequireClaude(t)

// Check claude availability without skipping
if helpers.IsCLIAvailable() {
    // claude is available
}
```

### Best Practices

1. **Always use build tags**: Start every e2e test file with `//go:build e2e`
2. **Use MockClaudeBuilder for CI**: Write tests that can run without Claude API access
3. **Check prerequisites**: Use `helpers.Require*` functions for real CLI tests
4. **Table-driven tests**: Use table-driven approach for better organization
5. **Descriptive names**: Use descriptive test and subtest names
6. **Independence**: Each test should be independent and not rely on other tests
7. **Verify arguments**: Use `GetCapturedArgs()` to verify correct CLI arguments
8. **Test both modes**: Test both streaming and non-streaming execution when relevant

## Troubleshooting

### Tests are skipped

If tests are being skipped, check that required tools are installed and in PATH:

```bash
# Check git
git --version

# Check gh
gh --version
gh auth status

# Check claude
claude --version
```

### Permission errors

If you see permission errors when creating temporary directories:

```bash
# Check temp directory permissions
ls -la /tmp

# Set custom temp directory
export TMPDIR=/path/to/writable/dir
```

### Timeout errors

If tests timeout, increase the timeout:

```bash
# Using script
E2E_TIMEOUT=5m ./scripts/run-e2e-tests.sh

# Using go test
go test -tags=e2e -timeout=5m ./test/e2e/...
```

### Git configuration issues

If git operations fail with user configuration errors:

```bash
# Set global git config
git config --global user.email "test@test.com"
git config --global user.name "Test User"
```

Note: The TempRepo helper automatically configures git user for each test repository.

## CI Integration

E2E tests run automatically in CI via GitHub Actions. The workflow:

1. Installs Go
2. Runs `./scripts/run-e2e-tests.sh`
3. Uses 1-minute timeout
4. Skips real Claude CLI tests (runs only mock-based tests)

Mock-based tests provide fast, reliable verification without requiring Claude API credentials.

See `.github/workflows/go.yml` for the CI configuration.

## Related Documentation

- [Go Testing](https://golang.org/pkg/testing/)
- [Build Tags](https://pkg.go.dev/cmd/go#hdr-Build_constraints)
- [testify](https://github.com/stretchr/testify)
