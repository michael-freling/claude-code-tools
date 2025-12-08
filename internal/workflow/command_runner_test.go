package workflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRunner_Run(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		args       []string
		wantStdout string
		wantErr    bool
	}{
		{
			name:       "successful command",
			command:    "echo",
			args:       []string{"hello"},
			wantStdout: "hello",
			wantErr:    false,
		},
		{
			name:       "command with multiple args",
			command:    "echo",
			args:       []string{"hello", "world"},
			wantStdout: "hello world",
			wantErr:    false,
		},
		{
			name:    "command that fails",
			command: "false",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewCommandRunner()
			ctx := context.Background()

			stdout, stderr, err := runner.Run(ctx, tt.command, tt.args...)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantStdout, stdout)
			assert.Empty(t, stderr)
		})
	}
}

func TestCommandRunner_RunInDir(t *testing.T) {
	tests := []struct {
		name       string
		dir        string
		command    string
		args       []string
		wantStdout string
		wantErr    bool
	}{
		{
			name:       "run in current directory",
			dir:        "",
			command:    "echo",
			args:       []string{"test"},
			wantStdout: "test",
			wantErr:    false,
		},
		{
			name:       "run in /tmp directory",
			dir:        "/tmp",
			command:    "pwd",
			args:       []string{},
			wantStdout: "/tmp",
			wantErr:    false,
		},
		{
			name:    "command fails in directory",
			dir:     "/tmp",
			command: "false",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewCommandRunner()
			ctx := context.Background()

			stdout, stderr, err := runner.RunInDir(ctx, tt.dir, tt.command, tt.args...)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantStdout, stdout)
			assert.Empty(t, stderr)
		})
	}
}

func TestCommandRunner_Run_WithContext(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
	}{
		{
			name:    "context cancellation",
			command: "sleep",
			args:    []string{"10"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewCommandRunner()
			ctx, cancel := context.WithCancel(context.Background())

			cancel()

			_, _, err := runner.Run(ctx, tt.command, tt.args...)

			if tt.wantErr {
				require.Error(t, err)
			}
		})
	}
}

func TestCommandRunner_RunInDir_WithStderr(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		args       []string
		wantStderr bool
	}{
		{
			name:       "command with stderr output",
			command:    "sh",
			args:       []string{"-c", "echo error >&2"},
			wantStderr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewCommandRunner()
			ctx := context.Background()

			_, stderr, err := runner.RunInDir(ctx, "", tt.command, tt.args...)

			require.NoError(t, err)
			if tt.wantStderr {
				assert.NotEmpty(t, stderr)
			}
		})
	}
}
