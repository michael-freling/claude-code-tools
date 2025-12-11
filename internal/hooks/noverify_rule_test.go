package hooks

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNoVerifyRule(t *testing.T) {
	rule := NewNoVerifyRule()
	assert.NotNil(t, rule)
	assert.Equal(t, "no-verify", rule.Name())
	assert.Equal(t, "Blocks Bash commands containing the --no-verify flag", rule.Description())
}

func TestNoVerifyRule_Evaluate(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		command     string
		wantAllowed bool
		wantMessage string
	}{
		{
			name:        "allow non-Bash tool",
			toolName:    "Write",
			command:     "git commit --no-verify",
			wantAllowed: true,
		},
		{
			name:        "allow Bash without --no-verify",
			toolName:    "Bash",
			command:     "git commit -m 'test message'",
			wantAllowed: true,
		},
		{
			name:        "block git commit --no-verify",
			toolName:    "Bash",
			command:     "git commit --no-verify",
			wantAllowed: false,
			wantMessage: "Command contains --no-verify flag which bypasses git hooks",
		},
		{
			name:        "block git commit with message and --no-verify",
			toolName:    "Bash",
			command:     "git commit -m 'message' --no-verify",
			wantAllowed: false,
			wantMessage: "Command contains --no-verify flag which bypasses git hooks",
		},
		{
			name:        "block --no-verify in middle of command",
			toolName:    "Bash",
			command:     "git commit --no-verify -m 'message'",
			wantAllowed: false,
			wantMessage: "Command contains --no-verify flag which bypasses git hooks",
		},
		{
			name:        "allow --no-verify in single quotes",
			toolName:    "Bash",
			command:     "echo '--no-verify'",
			wantAllowed: true,
		},
		{
			name:        "allow --no-verify in double quotes",
			toolName:    "Bash",
			command:     `echo "--no-verify"`,
			wantAllowed: true,
		},
		{
			name:        "allow --no-verify as part of string in single quotes",
			toolName:    "Bash",
			command:     "git commit -m 'do not use --no-verify flag'",
			wantAllowed: true,
		},
		{
			name:        "allow --no-verify as part of string in double quotes",
			toolName:    "Bash",
			command:     `git commit -m "do not use --no-verify flag"`,
			wantAllowed: true,
		},
		{
			name:        "block git push --no-verify",
			toolName:    "Bash",
			command:     "git push --no-verify",
			wantAllowed: false,
			wantMessage: "Command contains --no-verify flag which bypasses git hooks",
		},
		{
			name:        "block command with multiple flags including --no-verify",
			toolName:    "Bash",
			command:     "git commit -a -m 'message' --no-verify --verbose",
			wantAllowed: false,
			wantMessage: "Command contains --no-verify flag which bypasses git hooks",
		},
		{
			name:        "allow Bash with no command argument",
			toolName:    "Bash",
			command:     "",
			wantAllowed: true,
		},
		{
			name:        "allow command with --no-verify-like string",
			toolName:    "Bash",
			command:     "echo no-verify-test",
			wantAllowed: true,
		},
		{
			name:        "block --no-verify with tabs",
			toolName:    "Bash",
			command:     "git\tcommit\t--no-verify",
			wantAllowed: false,
			wantMessage: "Command contains --no-verify flag which bypasses git hooks",
		},
		{
			name:        "block --no-verify with newlines",
			toolName:    "Bash",
			command:     "git commit\n--no-verify",
			wantAllowed: false,
			wantMessage: "Command contains --no-verify flag which bypasses git hooks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewNoVerifyRule()

			var toolInput *ToolInput
			if tt.command != "" {
				jsonInput := `{"tool_name": "` + tt.toolName + `", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
				reader := strings.NewReader(jsonInput)
				var err error
				toolInput, err = ParseToolInput(reader)
				require.NoError(t, err)
			} else {
				jsonInput := `{"tool_name": "` + tt.toolName + `", "tool_input": {}}`
				reader := strings.NewReader(jsonInput)
				var err error
				toolInput, err = ParseToolInput(reader)
				require.NoError(t, err)
			}

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAllowed, got.Allowed)

			if !tt.wantAllowed {
				assert.Equal(t, "no-verify", got.RuleName)
				assert.Equal(t, tt.wantMessage, got.Message)
			}
		})
	}
}

// escapeJSON escapes a string for use in JSON.
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}
