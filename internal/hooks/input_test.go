package hooks

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseToolInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *ToolInput
		wantErr bool
	}{
		{
			name:  "valid input with tool_input",
			input: `{"tool_name": "Bash", "tool_input": {"command": "ls -la"}}`,
			want: &ToolInput{
				ToolName: "Bash",
			},
			wantErr: false,
		},
		{
			name:  "valid input without tool_input",
			input: `{"tool_name": "Test"}`,
			want: &ToolInput{
				ToolName: "Test",
			},
			wantErr: false,
		},
		{
			name:  "valid input with empty tool_input",
			input: `{"tool_name": "Test", "tool_input": {}}`,
			want: &ToolInput{
				ToolName: "Test",
			},
			wantErr: false,
		},
		{
			name:    "missing tool_name",
			input:   `{"tool_input": {"command": "ls"}}`,
			wantErr: true,
		},
		{
			name:    "empty tool_name",
			input:   `{"tool_name": "", "tool_input": {"command": "ls"}}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "invalid tool_input JSON",
			input:   `{"tool_name": "Test", "tool_input": "not an object"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			got, err := ParseToolInput(reader)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.ToolName, got.ToolName)
		})
	}
}

func TestToolInput_GetStringArg(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		argName   string
		wantValue string
		wantOk    bool
	}{
		{
			name:      "existing string argument",
			input:     `{"tool_name": "Bash", "tool_input": {"command": "ls -la"}}`,
			argName:   "command",
			wantValue: "ls -la",
			wantOk:    true,
		},
		{
			name:      "non-existent argument",
			input:     `{"tool_name": "Bash", "tool_input": {"command": "ls -la"}}`,
			argName:   "nonexistent",
			wantValue: "",
			wantOk:    false,
		},
		{
			name:      "non-string argument",
			input:     `{"tool_name": "Test", "tool_input": {"count": 123}}`,
			argName:   "count",
			wantValue: "",
			wantOk:    false,
		},
		{
			name:      "empty tool_input",
			input:     `{"tool_name": "Test", "tool_input": {}}`,
			argName:   "command",
			wantValue: "",
			wantOk:    false,
		},
		{
			name:      "nil tool_input",
			input:     `{"tool_name": "Test"}`,
			argName:   "command",
			wantValue: "",
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			gotValue, gotOk := toolInput.GetStringArg(tt.argName)
			assert.Equal(t, tt.wantValue, gotValue)
			assert.Equal(t, tt.wantOk, gotOk)
		})
	}
}

func TestToolInput_GetBoolArg(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		argName   string
		wantValue bool
		wantOk    bool
	}{
		{
			name:      "existing bool argument true",
			input:     `{"tool_name": "Test", "tool_input": {"enabled": true}}`,
			argName:   "enabled",
			wantValue: true,
			wantOk:    true,
		},
		{
			name:      "existing bool argument false",
			input:     `{"tool_name": "Test", "tool_input": {"enabled": false}}`,
			argName:   "enabled",
			wantValue: false,
			wantOk:    true,
		},
		{
			name:      "non-existent argument",
			input:     `{"tool_name": "Test", "tool_input": {"enabled": true}}`,
			argName:   "nonexistent",
			wantValue: false,
			wantOk:    false,
		},
		{
			name:      "non-bool argument",
			input:     `{"tool_name": "Test", "tool_input": {"name": "test"}}`,
			argName:   "name",
			wantValue: false,
			wantOk:    false,
		},
		{
			name:      "empty tool_input",
			input:     `{"tool_name": "Test", "tool_input": {}}`,
			argName:   "enabled",
			wantValue: false,
			wantOk:    false,
		},
		{
			name:      "nil tool_input",
			input:     `{"tool_name": "Test"}`,
			argName:   "enabled",
			wantValue: false,
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			gotValue, gotOk := toolInput.GetBoolArg(tt.argName)
			assert.Equal(t, tt.wantValue, gotValue)
			assert.Equal(t, tt.wantOk, gotOk)
		})
	}
}
