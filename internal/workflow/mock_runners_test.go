package workflow

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockCommandRunner_Run(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func(*MockCommandRunner)
		cmdName    string
		args       []string
		wantStdout string
		wantStderr string
		wantErr    bool
	}{
		{
			name: "successful command execution",
			setupMock: func(m *MockCommandRunner) {
				m.On("Run", context.Background(), "echo", "hello", "world").Return("hello world\n", "", nil)
			},
			cmdName:    "echo",
			args:       []string{"hello", "world"},
			wantStdout: "hello world\n",
			wantStderr: "",
			wantErr:    false,
		},
		{
			name: "command execution with error",
			setupMock: func(m *MockCommandRunner) {
				m.On("Run", context.Background(), "false").Return("", "error occurred", fmt.Errorf("exit status 1"))
			},
			cmdName:    "false",
			args:       []string{},
			wantStdout: "",
			wantStderr: "error occurred",
			wantErr:    true,
		},
		{
			name: "command with multiple arguments",
			setupMock: func(m *MockCommandRunner) {
				m.On("Run", context.Background(), "git", "status", "--short").Return(" M file.go\n", "", nil)
			},
			cmdName:    "git",
			args:       []string{"status", "--short"},
			wantStdout: " M file.go\n",
			wantStderr: "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			stdout, stderr, err := mockRunner.Run(context.Background(), tt.cmdName, tt.args...)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantStdout, stdout)
			assert.Equal(t, tt.wantStderr, stderr)
			mockRunner.AssertExpectations(t)
		})
	}
}
