//go:build e2e

package helpers

import (
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockClaudeBuilder_WithResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{
			name:     "returns simple response",
			response: "Hello, world!",
		},
		{
			name:     "returns multiline response",
			response: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "returns response with special characters",
			response: "Response with 'quotes' and \"double quotes\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewMockClaudeBuilder(t)
			scriptPath := builder.WithResponse(tt.response).Build()

			cmd := exec.Command(scriptPath, "--print", "test prompt")
			output, err := cmd.CombinedOutput()

			require.NoError(t, err)
			assert.Equal(t, tt.response+"\n", string(output))
		})
	}
}

func TestMockClaudeBuilder_WithStreamingResponse(t *testing.T) {
	tests := []struct {
		name             string
		result           string
		structuredOutput interface{}
	}{
		{
			name:   "returns streaming response without structured output",
			result: "Task completed",
		},
		{
			name:   "returns streaming response with structured output",
			result: "Task completed",
			structuredOutput: map[string]interface{}{
				"key":   "value",
				"count": 42,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewMockClaudeBuilder(t)
			scriptPath := builder.WithStreamingResponse(tt.result, tt.structuredOutput).Build()

			cmd := exec.Command(scriptPath, "--output-format", "stream-json", "test prompt")
			output, err := cmd.CombinedOutput()

			require.NoError(t, err)

			lines := splitLines(string(output))
			require.Len(t, lines, 2, "expected 2 JSON lines")

			var assistantMsg map[string]interface{}
			err = json.Unmarshal([]byte(lines[0]), &assistantMsg)
			require.NoError(t, err)
			assert.Equal(t, "assistant", assistantMsg["type"])

			var resultMsg map[string]interface{}
			err = json.Unmarshal([]byte(lines[1]), &resultMsg)
			require.NoError(t, err)
			assert.Equal(t, "result", resultMsg["type"])
			assert.Equal(t, "success", resultMsg["subtype"])
			assert.Equal(t, tt.result, resultMsg["result"])

			if tt.structuredOutput != nil {
				assert.NotNil(t, resultMsg["structured_output"])
			}
		})
	}
}

func TestMockClaudeBuilder_WithExitCode(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		wantErr  bool
	}{
		{
			name:     "exits with code 0",
			exitCode: 0,
			wantErr:  false,
		},
		{
			name:     "exits with code 1",
			exitCode: 1,
			wantErr:  true,
		},
		{
			name:     "exits with code 2",
			exitCode: 2,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewMockClaudeBuilder(t)
			scriptPath := builder.WithExitCode(tt.exitCode).Build()

			cmd := exec.Command(scriptPath, "test prompt")
			err := cmd.Run()

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMockClaudeBuilder_WithStderr(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
	}{
		{
			name:   "outputs stderr message",
			stderr: "Error: something went wrong",
		},
		{
			name:   "outputs multiline stderr",
			stderr: "Error line 1\nError line 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewMockClaudeBuilder(t)
			scriptPath := builder.WithStderr(tt.stderr).Build()

			cmd := exec.Command(scriptPath, "test prompt")
			output, _ := cmd.CombinedOutput()

			assert.Contains(t, string(output), tt.stderr)
		})
	}
}

func TestMockClaudeBuilder_GetCapturedArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "captures single argument",
			args: []string{"test prompt"},
			want: []string{"test prompt"},
		},
		{
			name: "captures multiple arguments",
			args: []string{"--print", "--model", "opus", "test prompt"},
			want: []string{"--print", "--model", "opus", "test prompt"},
		},
		{
			name: "captures arguments with spaces",
			args: []string{"--print", "prompt with spaces"},
			want: []string{"--print", "prompt with spaces"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewMockClaudeBuilder(t)
			scriptPath := builder.Build()

			cmd := exec.Command(scriptPath, tt.args...)
			_, err := cmd.CombinedOutput()
			require.NoError(t, err)

			got := builder.GetCapturedArgs()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMockClaudeBuilder_GetCapturedPrompt(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "returns last argument as prompt",
			args: []string{"--print", "test prompt"},
			want: "test prompt",
		},
		{
			name: "returns empty string when no args",
			args: []string{},
			want: "",
		},
		{
			name: "returns prompt with special characters",
			args: []string{"--print", "prompt with 'quotes'"},
			want: "prompt with 'quotes'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewMockClaudeBuilder(t)
			scriptPath := builder.Build()

			if len(tt.args) > 0 {
				cmd := exec.Command(scriptPath, tt.args...)
				_, err := cmd.CombinedOutput()
				require.NoError(t, err)
			}

			got := builder.GetCapturedPrompt()
			assert.Equal(t, tt.want, got)
		})
	}
}

func splitLines(s string) []string {
	var lines []string
	for _, line := range splitByNewline(s) {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func splitByNewline(s string) []string {
	var lines []string
	current := ""
	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
