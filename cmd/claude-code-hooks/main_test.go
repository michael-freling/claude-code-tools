package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootCmd(t *testing.T) {
	cmd := newRootCmd()

	assert.Equal(t, "claude-hooks", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	commandNames := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		commandNames = append(commandNames, c.Name())
	}
	assert.ElementsMatch(t, []string{"pre-tool-use"}, commandNames)
}

func TestNewPreToolUseCmd(t *testing.T) {
	cmd := newPreToolUseCmd()

	assert.Equal(t, "pre-tool-use", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	err := cmd.Args(cmd, []string{})
	assert.NoError(t, err)

	err = cmd.Args(cmd, []string{"extra"})
	assert.Error(t, err)
}

func TestPreToolUseCmd_Execute(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		wantExit bool
	}{
		{
			name:     "valid input with no rules allows",
			input:    `{"tool_name": "Bash", "tool_input": {"command": "ls"}}`,
			wantErr:  false,
			wantExit: false,
		},
		{
			name:    "invalid JSON returns error",
			input:   `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "missing tool_name returns error",
			input:   `{"tool_input": {"command": "ls"}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newPreToolUseCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetIn(strings.NewReader(tt.input))

			err := cmd.Execute()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestPreToolUseCmd_ExitCodes(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "valid input exits successfully",
			input: `{"tool_name": "Bash", "tool_input": {"command": "ls"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newPreToolUseCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetIn(strings.NewReader(tt.input))

			err := cmd.Execute()
			require.NoError(t, err)
		})
	}
}

func TestPreToolUseCmd_IntegrationAllowedCommands(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "allows safe commands",
			input: `{"tool_name": "Bash", "tool_input": {"command": "ls -la"}}`,
		},
		{
			name:  "allows git status",
			input: `{"tool_name": "Bash", "tool_input": {"command": "git status"}}`,
		},
		{
			name:  "allows push to feature branch",
			input: `{"tool_name": "Bash", "tool_input": {"command": "git push origin feature/test"}}`,
		},
		{
			name:  "allows gh api GET branch protection",
			input: `{"tool_name": "Bash", "tool_input": {"command": "gh api /repos/owner/repo/branches/main/protection"}}`,
		},
		{
			name:  "allows gh pr list",
			input: `{"tool_name": "Bash", "tool_input": {"command": "gh pr list"}}`,
		},
		{
			name:  "allows git commit without --no-verify",
			input: `{"tool_name": "Bash", "tool_input": {"command": "git commit -m 'test'"}}`,
		},
		{
			name:  "allows non-Bash tools",
			input: `{"tool_name": "Read", "tool_input": {"file_path": "/tmp/test.txt"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newPreToolUseCmd()
			outBuf := new(bytes.Buffer)
			errBuf := new(bytes.Buffer)
			cmd.SetOut(outBuf)
			cmd.SetErr(errBuf)
			cmd.SetIn(strings.NewReader(tt.input))

			err := cmd.Execute()

			require.NoError(t, err)
			assert.Empty(t, errBuf.String())
		})
	}
}
