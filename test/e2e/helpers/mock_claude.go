//go:build e2e

package helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MockClaudeBuilder creates mock Claude CLI scripts for testing.
// This allows testing CLI argument construction and response parsing in CI
// without requiring real Claude API access.
type MockClaudeBuilder struct {
	t                *testing.T
	response         string
	streamingResult  string
	structuredOutput interface{}
	exitCode         int
	stderr           string
	scriptPath       string
	argsPath         string
	isStreaming      bool
}

// NewMockClaudeBuilder creates a new mock builder
func NewMockClaudeBuilder(t *testing.T) *MockClaudeBuilder {
	t.Helper()

	return &MockClaudeBuilder{
		t:        t,
		exitCode: 0,
	}
}

// WithResponse sets the response to return (for non-streaming mode)
func (m *MockClaudeBuilder) WithResponse(response string) *MockClaudeBuilder {
	m.response = response
	return m
}

// WithStreamingResponse sets streaming JSON response
func (m *MockClaudeBuilder) WithStreamingResponse(result string, structuredOutput interface{}) *MockClaudeBuilder {
	m.streamingResult = result
	m.structuredOutput = structuredOutput
	m.isStreaming = true
	return m
}

// WithExitCode sets the exit code to return
func (m *MockClaudeBuilder) WithExitCode(code int) *MockClaudeBuilder {
	m.exitCode = code
	return m
}

// WithStderr sets stderr output (for error cases)
func (m *MockClaudeBuilder) WithStderr(stderr string) *MockClaudeBuilder {
	m.stderr = stderr
	return m
}

// Build creates the mock script and returns the path
func (m *MockClaudeBuilder) Build() string {
	m.t.Helper()

	tmpDir, err := os.MkdirTemp("", "mock-claude-*")
	if err != nil {
		m.t.Fatalf("failed to create temp dir: %v", err)
	}

	m.scriptPath = filepath.Join(tmpDir, "claude")
	m.argsPath = m.scriptPath + ".args"

	script := m.generateScript()

	if err := os.WriteFile(m.scriptPath, []byte(script), 0755); err != nil {
		_ = os.RemoveAll(tmpDir)
		m.t.Fatalf("failed to write mock script: %v", err)
	}

	m.t.Cleanup(func() {
		_ = os.RemoveAll(tmpDir)
	})

	return m.scriptPath
}

// GetCapturedArgs returns the arguments that were passed to the mock script
func (m *MockClaudeBuilder) GetCapturedArgs() []string {
	m.t.Helper()

	if m.argsPath == "" {
		m.t.Fatal("mock script not built yet - call Build() first")
	}

	data, err := os.ReadFile(m.argsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}
		}
		m.t.Fatalf("failed to read captured args: %v", err)
	}

	if len(data) == 0 {
		return []string{}
	}

	args := strings.Split(strings.TrimSpace(string(data)), "\n")
	return args
}

// GetCapturedPrompt returns the prompt that was passed (last argument)
func (m *MockClaudeBuilder) GetCapturedPrompt() string {
	m.t.Helper()

	args := m.GetCapturedArgs()
	if len(args) == 0 {
		return ""
	}

	return args[len(args)-1]
}

func (m *MockClaudeBuilder) generateScript() string {
	var scriptBuilder strings.Builder

	scriptBuilder.WriteString("#!/bin/bash\n")
	scriptBuilder.WriteString("# Mock Claude CLI script for testing\n\n")

	scriptBuilder.WriteString(fmt.Sprintf("ARGS_FILE=\"%s\"\n\n", m.argsPath))

	scriptBuilder.WriteString("# Capture all arguments\n")
	scriptBuilder.WriteString("for arg in \"$@\"; do\n")
	scriptBuilder.WriteString("  echo \"$arg\" >> \"$ARGS_FILE\"\n")
	scriptBuilder.WriteString("done\n\n")

	if m.stderr != "" {
		scriptBuilder.WriteString("# Write stderr output\n")
		scriptBuilder.WriteString(fmt.Sprintf("echo %s >&2\n\n", shellQuote(m.stderr)))
	}

	scriptBuilder.WriteString("# Check if streaming mode is requested\n")
	scriptBuilder.WriteString("IS_STREAMING=false\n")
	scriptBuilder.WriteString("for arg in \"$@\"; do\n")
	scriptBuilder.WriteString("  if [ \"$arg\" = \"stream-json\" ]; then\n")
	scriptBuilder.WriteString("    IS_STREAMING=true\n")
	scriptBuilder.WriteString("    break\n")
	scriptBuilder.WriteString("  fi\n")
	scriptBuilder.WriteString("done\n\n")

	scriptBuilder.WriteString("# Output response based on mode\n")
	scriptBuilder.WriteString("if [ \"$IS_STREAMING\" = \"true\" ]; then\n")

	if m.isStreaming {
		assistantMsg := map[string]interface{}{
			"type": "assistant",
			"message": map[string]interface{}{
				"content": []map[string]string{
					{"type": "text", "text": "Working..."},
				},
			},
		}
		assistantJSON, _ := json.Marshal(assistantMsg)

		resultMsg := map[string]interface{}{
			"type":    "result",
			"subtype": "success",
			"result":  m.streamingResult,
		}
		if m.structuredOutput != nil {
			resultMsg["structured_output"] = m.structuredOutput
		}
		resultJSON, _ := json.Marshal(resultMsg)

		scriptBuilder.WriteString(fmt.Sprintf("  echo %s\n", shellQuote(string(assistantJSON))))
		scriptBuilder.WriteString(fmt.Sprintf("  echo %s\n", shellQuote(string(resultJSON))))
	} else {
		scriptBuilder.WriteString("  echo 'Streaming mode not configured'\n")
	}

	scriptBuilder.WriteString("else\n")

	if m.response != "" {
		scriptBuilder.WriteString(fmt.Sprintf("  echo %s\n", shellQuote(m.response)))
	} else {
		scriptBuilder.WriteString("  echo 'Non-streaming mode not configured'\n")
	}

	scriptBuilder.WriteString("fi\n\n")

	scriptBuilder.WriteString(fmt.Sprintf("exit %d\n", m.exitCode))

	return scriptBuilder.String()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
