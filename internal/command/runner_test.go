package command

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunner(t *testing.T) {
	got := NewRunner()
	require.NotNil(t, got)
}

func TestRunner_Run(t *testing.T) {
	tests := []struct {
		name       string
		cmd        string
		args       []string
		wantStdout string
		wantErr    bool
	}{
		{
			name:       "runs echo command successfully",
			cmd:        "echo",
			args:       []string{"hello"},
			wantStdout: "hello",
			wantErr:    false,
		},
		{
			name:    "fails with non-existent command",
			cmd:     "non-existent-command-12345",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "runs echo command successfully" {
				if _, err := exec.LookPath(tt.cmd); err != nil {
					t.Skipf("command %s not found in PATH", tt.cmd)
				}
			}

			runner := NewRunner()
			ctx := context.Background()

			stdout, _, err := runner.Run(ctx, tt.cmd, tt.args...)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantStdout, stdout)
		})
	}
}

func TestRunner_RunInDir(t *testing.T) {
	tests := []struct {
		name       string
		dir        string
		cmd        string
		args       []string
		wantStdout string
		wantErr    bool
	}{
		{
			name:       "runs command in current directory when dir is empty",
			dir:        "",
			cmd:        "echo",
			args:       []string{"test"},
			wantStdout: "test",
			wantErr:    false,
		},
		{
			name:    "fails when directory does not exist",
			dir:     "/non/existent/directory",
			cmd:     "echo",
			args:    []string{"test"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd == "echo" {
				if _, err := exec.LookPath(tt.cmd); err != nil {
					t.Skipf("command %s not found in PATH", tt.cmd)
				}
			}

			runner := NewRunner()
			ctx := context.Background()

			stdout, _, err := runner.RunInDir(ctx, tt.dir, tt.cmd, tt.args...)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantStdout, stdout)
		})
	}
}
