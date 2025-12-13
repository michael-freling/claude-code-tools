package workflow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/michael-freling/claude-code-tools/internal/command"
)

// mockExecutor is a mock implementation of ClaudeExecutor for testing
type mockExecutor struct {
	executeFunc func(ctx context.Context, config ExecuteConfig) (*ExecuteResult, error)
}

func (m *mockExecutor) Execute(ctx context.Context, config ExecuteConfig) (*ExecuteResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, config)
	}
	return &ExecuteResult{
		Output:   "mock output",
		ExitCode: 0,
		Duration: 100 * time.Millisecond,
	}, nil
}

func TestNewClaudeExecutor(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates executor successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(LogLevelNormal)
			got := NewClaudeExecutor(logger)
			assert.NotNil(t, got)
		})
	}
}

func TestNewClaudeExecutorWithPath(t *testing.T) {
	tests := []struct {
		name       string
		claudePath string
	}{
		{
			name:       "creates executor with custom path",
			claudePath: "/usr/local/bin/claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(LogLevelNormal)
			got := NewClaudeExecutorWithPath(tt.claudePath, logger)
			assert.NotNil(t, got)

			executor, ok := got.(*claudeExecutor)
			require.True(t, ok)
			assert.Equal(t, tt.claudePath, executor.claudePath)
		})
	}
}

func TestMockExecutor_Execute_Success(t *testing.T) {
	tests := []struct {
		name       string
		config     ExecuteConfig
		mockFunc   func(ctx context.Context, config ExecuteConfig) (*ExecuteResult, error)
		wantOutput string
		wantErr    bool
	}{
		{
			name: "executes successfully with mock",
			config: ExecuteConfig{
				Prompt:           "test prompt",
				WorkingDirectory: "/tmp",
				Timeout:          5 * time.Second,
			},
			mockFunc: func(ctx context.Context, config ExecuteConfig) (*ExecuteResult, error) {
				return &ExecuteResult{
					Output:   "test output",
					ExitCode: 0,
					Duration: 50 * time.Millisecond,
				}, nil
			},
			wantOutput: "test output",
			wantErr:    false,
		},
		{
			name: "executes successfully with JSON schema",
			config: ExecuteConfig{
				Prompt:     "test prompt",
				JSONSchema: `{"type": "object"}`,
				Timeout:    5 * time.Second,
			},
			mockFunc: func(ctx context.Context, config ExecuteConfig) (*ExecuteResult, error) {
				if config.JSONSchema == "" {
					return nil, errors.New("expected JSONSchema to be set")
				}
				return &ExecuteResult{
					Output:   `{"result": "success"}`,
					ExitCode: 0,
					Duration: 50 * time.Millisecond,
				}, nil
			},
			wantOutput: `{"result": "success"}`,
			wantErr:    false,
		},
		{
			name: "handles timeout error",
			config: ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 1 * time.Millisecond,
			},
			mockFunc: func(ctx context.Context, config ExecuteConfig) (*ExecuteResult, error) {
				return &ExecuteResult{
					Error: ErrClaudeTimeout,
				}, ErrClaudeTimeout
			},
			wantErr: true,
		},
		{
			name: "handles execution error",
			config: ExecuteConfig{
				Prompt: "test prompt",
			},
			mockFunc: func(ctx context.Context, config ExecuteConfig) (*ExecuteResult, error) {
				return &ExecuteResult{
					ExitCode: 1,
					Error:    errors.New("execution failed"),
				}, ErrClaude
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &mockExecutor{
				executeFunc: tt.mockFunc,
			}

			ctx := context.Background()
			got, err := executor.Execute(ctx, tt.config)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantOutput, got.Output)
			assert.Equal(t, 0, got.ExitCode)
		})
	}
}

func TestMockExecutor_Execute_Timeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
	}{
		{
			name:    "respects timeout",
			timeout: 1 * time.Millisecond,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &mockExecutor{
				executeFunc: func(ctx context.Context, config ExecuteConfig) (*ExecuteResult, error) {
					if config.Timeout > 0 {
						var cancel context.CancelFunc
						ctx, cancel = context.WithTimeout(ctx, config.Timeout)
						defer cancel()
					}

					select {
					case <-time.After(100 * time.Millisecond):
						return &ExecuteResult{Output: "completed"}, nil
					case <-ctx.Done():
						return &ExecuteResult{Error: ErrClaudeTimeout}, ErrClaudeTimeout
					}
				},
			}

			ctx := context.Background()
			config := ExecuteConfig{
				Prompt:  "test",
				Timeout: tt.timeout,
			}

			_, err := executor.Execute(ctx, config)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrClaudeTimeout)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestMockExecutor_Execute_WithEnv(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{
			name: "executes with environment variables",
			env: map[string]string{
				"TEST_VAR": "test_value",
			},
			wantErr: false,
		},
		{
			name:    "executes without environment variables",
			env:     nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &mockExecutor{
				executeFunc: func(ctx context.Context, config ExecuteConfig) (*ExecuteResult, error) {
					return &ExecuteResult{
						Output:   "success",
						ExitCode: 0,
					}, nil
				},
			}

			ctx := context.Background()
			config := ExecuteConfig{
				Prompt: "test",
				Env:    tt.env,
			}

			got, err := executor.Execute(ctx, config)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "success", got.Output)
		})
	}
}

func TestClaudeExecutor_buildEnv(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantLen int
	}{
		{
			name: "builds environment variables",
			env: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			wantLen: 2,
		},
		{
			name:    "handles empty environment",
			env:     map[string]string{},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &claudeExecutor{}
			got := executor.buildEnv(tt.env)
			assert.Len(t, got, tt.wantLen)

			for key, value := range tt.env {
				expected := key + "=" + value
				assert.Contains(t, got, expected)
			}
		})
	}
}

func TestClaudeExecutor_findClaudePath(t *testing.T) {
	tests := []struct {
		name       string
		claudePath string
		wantPath   string
		wantErr    bool
	}{
		{
			name:       "returns custom path when set",
			claudePath: "/usr/local/bin/claude",
			wantPath:   "/usr/local/bin/claude",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &claudeExecutor{
				claudePath: tt.claudePath,
			}

			got, err := executor.findClaudePath()

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrClaudeNotFound)
				assert.Empty(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, got)
		})
	}
}

func TestClaudeExecutor_findClaudePath_FromPATH(t *testing.T) {
	// This test checks the behavior when claudePath is "claude" or empty
	// and relies on exec.LookPath to find claude in PATH
	executor := &claudeExecutor{
		claudePath: "claude",
	}

	got, err := executor.findClaudePath()

	// claude may or may not be in PATH depending on the environment
	if err != nil {
		// If claude is not in PATH, we should get ErrClaudeNotFound
		assert.ErrorIs(t, err, ErrClaudeNotFound)
		assert.Empty(t, got)
	} else {
		// If claude is in PATH, we should get a valid path
		assert.NotEmpty(t, got)
	}
}

func TestClaudeExecutor_findClaudePath_CustomPath(t *testing.T) {
	tests := []struct {
		name       string
		claudePath string
		wantPath   string
	}{
		{
			name:       "returns custom path without validation",
			claudePath: "/custom/path/to/claude",
			wantPath:   "/custom/path/to/claude",
		},
		{
			name:       "returns nonexistent custom path without validation",
			claudePath: "/nonexistent/path/that/does/not/exist",
			wantPath:   "/nonexistent/path/that/does/not/exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &claudeExecutor{
				claudePath: tt.claudePath,
			}

			got, err := executor.findClaudePath()

			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, got)
		})
	}
}

func TestMockExecutor_Execute_ExitCode(t *testing.T) {
	tests := []struct {
		name         string
		mockExitCode int
		wantErr      bool
	}{
		{
			name:         "handles non-zero exit code",
			mockExitCode: 1,
			wantErr:      true,
		},
		{
			name:         "handles zero exit code",
			mockExitCode: 0,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &mockExecutor{
				executeFunc: func(ctx context.Context, config ExecuteConfig) (*ExecuteResult, error) {
					result := &ExecuteResult{
						ExitCode: tt.mockExitCode,
						Output:   "output",
					}

					if tt.mockExitCode != 0 {
						result.Error = errors.New("command failed")
						return result, ErrClaude
					}

					return result, nil
				},
			}

			ctx := context.Background()
			config := ExecuteConfig{Prompt: "test"}

			got, err := executor.Execute(ctx, config)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.mockExitCode, got.ExitCode)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, 0, got.ExitCode)
		})
	}
}

func TestClaudeExecutor_Execute_WithMockScript(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		script      string
		config      ExecuteConfig
		wantOutput  string
		wantErr     bool
		errContains string
	}{
		{
			name: "successful execution",
			script: `#!/bin/bash
echo "Hello from mock claude"
exit 0`,
			config: ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 5 * time.Second,
			},
			wantOutput: "Hello from mock claude\n",
			wantErr:    false,
		},
		{
			name: "execution with JSON schema",
			script: `#!/bin/bash
echo '{"result": "success"}'
exit 0`,
			config: ExecuteConfig{
				Prompt:     "test prompt",
				JSONSchema: `{"type": "object"}`,
				Timeout:    5 * time.Second,
			},
			wantOutput: "{\"result\": \"success\"}\n",
			wantErr:    false,
		},
		{
			name: "execution with working directory",
			script: `#!/bin/bash
pwd
exit 0`,
			config: ExecuteConfig{
				Prompt:           "test prompt",
				WorkingDirectory: tmpDir,
				Timeout:          5 * time.Second,
			},
			wantOutput: tmpDir + "\n",
			wantErr:    false,
		},
		{
			name: "execution failure with non-zero exit code",
			script: `#!/bin/bash
echo "error message" >&2
exit 1`,
			config: ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 5 * time.Second,
			},
			wantErr:     true,
			errContains: "exit code 1",
		},
		{
			name: "execution with dangerously skip permissions",
			script: `#!/bin/bash
if [[ "$*" == *"--dangerously-skip-permissions"* ]]; then
  echo "permissions skipped"
else
  echo "permissions not skipped"
fi
exit 0`,
			config: ExecuteConfig{
				Prompt:                     "test prompt",
				DangerouslySkipPermissions: true,
				Timeout:                    5 * time.Second,
			},
			wantOutput: "permissions skipped\n",
			wantErr:    false,
		},
		{
			name: "execution with environment variables",
			script: `#!/bin/bash
echo "output"
exit 0`,
			config: ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 5 * time.Second,
				Env: map[string]string{
					"TEST_VAR": "test_value",
				},
			},
			wantOutput: "output\n",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath := filepath.Join(tmpDir, "claude-"+tt.name)
			err := os.WriteFile(scriptPath, []byte(tt.script), 0755)
			require.NoError(t, err)

			executor := NewClaudeExecutorWithPath(scriptPath, NewLogger(LogLevelNormal))
			ctx := context.Background()

			got, err := executor.Execute(ctx, tt.config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantOutput, got.Output)
			assert.Equal(t, 0, got.ExitCode)
		})
	}
}

func TestClaudeExecutor_Execute_Timeout_Real(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test in short mode")
	}

	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "claude-slow")
	script := `#!/bin/bash
sleep 0.05
echo "done"
exit 0`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	executor := NewClaudeExecutorWithPath(scriptPath, NewLogger(LogLevelNormal))
	ctx := context.Background()

	config := ExecuteConfig{
		Prompt:  "test prompt",
		Timeout: 20 * time.Millisecond,
	}

	got, err := executor.Execute(ctx, config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.NotNil(t, got)
}

func TestClaudeExecutor_Execute_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test in short mode")
	}

	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "claude-sleep")
	script := `#!/bin/bash
sleep 0.5
echo "done"
exit 0`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	executor := NewClaudeExecutorWithPath(scriptPath, NewLogger(LogLevelNormal))
	ctx, cancel := context.WithCancel(context.Background())

	// Start execution in goroutine
	var execErr error
	done := make(chan struct{})
	go func() {
		_, execErr = executor.Execute(ctx, ExecuteConfig{Prompt: "test prompt"})
		close(done)
	}()

	// Give the script time to start
	time.Sleep(1 * time.Millisecond)
	cancel() // cancel while script is sleeping

	<-done

	require.Error(t, execErr)
}

func TestClaudeExecutor_ExecuteStreaming_WithMockScript(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		script         string
		config         ExecuteConfig
		wantOutput     string
		wantErr        bool
		errContains    string
		wantToolEvents int
	}{
		{
			name: "successful streaming execution",
			script: `#!/bin/bash
echo '{"type":"system","subtype":"init"}'
echo '{"type":"assistant","message":{"content":[{"type":"text","text":"Hello"}]}}'
echo '{"type":"result","result":"Final output","is_error":false}'
exit 0`,
			config: ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 5 * time.Second,
			},
			wantOutput: "Final output",
			wantErr:    false,
		},
		{
			name: "streaming with tool use",
			script: `#!/bin/bash
echo '{"type":"system","subtype":"init"}'
echo '{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/test/path"}}]}}'
echo '{"type":"user","tool_use_result":"file contents"}'
echo '{"type":"result","result":"Done","is_error":false}'
exit 0`,
			config: ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 5 * time.Second,
			},
			wantOutput:     "Done",
			wantErr:        false,
			wantToolEvents: 2,
		},
		{
			name: "streaming with structured output",
			script: `#!/bin/bash
echo '{"type":"system","subtype":"init"}'
echo '{"type":"result","result":"text result","structured_output":{"key":"value"},"is_error":false}'
exit 0`,
			config: ExecuteConfig{
				Prompt:     "test prompt",
				JSONSchema: `{"type":"object"}`,
				Timeout:    5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "streaming execution failure",
			script: `#!/bin/bash
echo "error" >&2
exit 1`,
			config: ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 5 * time.Second,
			},
			wantErr:     true,
			errContains: "exit code 1",
		},
		{
			name: "streaming with error tool result",
			script: `#!/bin/bash
echo '{"type":"system","subtype":"init"}'
echo '{"type":"user","tool_use_result":"Error: something went wrong"}'
echo '{"type":"result","result":"Done","is_error":false}'
exit 0`,
			config: ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 5 * time.Second,
			},
			wantOutput:     "Done",
			wantErr:        false,
			wantToolEvents: 1,
		},
		{
			name: "streaming with dangerously skip permissions",
			script: `#!/bin/bash
if [[ "$*" == *"--dangerously-skip-permissions"* ]]; then
  echo '{"type":"result","result":"permissions skipped","is_error":false}'
else
  echo '{"type":"result","result":"permissions not skipped","is_error":false}'
fi
exit 0`,
			config: ExecuteConfig{
				Prompt:                     "test prompt",
				DangerouslySkipPermissions: true,
				Timeout:                    5 * time.Second,
			},
			wantOutput: "permissions skipped",
			wantErr:    false,
		},
		{
			name: "streaming with working directory",
			script: `#!/bin/bash
echo '{"type":"result","result":"'$(pwd)'","is_error":false}'
exit 0`,
			config: ExecuteConfig{
				Prompt:           "test prompt",
				WorkingDirectory: tmpDir,
				Timeout:          5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "streaming with environment variables",
			script: `#!/bin/bash
echo '{"type":"result","result":"env set","is_error":false}'
exit 0`,
			config: ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 5 * time.Second,
				Env: map[string]string{
					"TEST_VAR": "test_value",
				},
			},
			wantOutput: "env set",
			wantErr:    false,
		},
		{
			name: "streaming with empty lines in output",
			script: `#!/bin/bash
echo ''
echo '{"type":"result","result":"done","is_error":false}'
exit 0`,
			config: ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 5 * time.Second,
			},
			wantOutput: "done",
			wantErr:    false,
		},
		{
			name: "streaming with malformed JSON lines",
			script: `#!/bin/bash
echo 'not valid json'
echo '{"type":"result","result":"done","is_error":false}'
exit 0`,
			config: ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 5 * time.Second,
			},
			wantOutput: "done",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath := filepath.Join(tmpDir, "claude-stream-"+tt.name)
			err := os.WriteFile(scriptPath, []byte(tt.script), 0755)
			require.NoError(t, err)

			executor := NewClaudeExecutorWithPath(scriptPath, NewLogger(LogLevelNormal))
			ctx := context.Background()

			var toolEvents []ProgressEvent
			onProgress := func(event ProgressEvent) {
				toolEvents = append(toolEvents, event)
			}

			got, err := executor.ExecuteStreaming(ctx, tt.config, onProgress)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			if tt.wantOutput != "" {
				assert.Contains(t, got.Output, tt.wantOutput)
			}

			if tt.wantToolEvents > 0 {
				assert.Len(t, toolEvents, tt.wantToolEvents)
			}
		})
	}
}

func TestClaudeExecutor_ExecuteStreaming_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test in short mode")
	}

	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "claude-stream-slow")
	// Use a script that will be killed by SIGTERM/SIGKILL
	script := `#!/bin/bash
trap 'exit 1' TERM INT
while true; do sleep 0.01; done`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	executor := NewClaudeExecutorWithPath(scriptPath, NewLogger(LogLevelNormal))
	ctx := context.Background()

	config := ExecuteConfig{
		Prompt:  "test prompt",
		Timeout: 20 * time.Millisecond,
	}

	got, err := executor.ExecuteStreaming(ctx, config, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.NotNil(t, got)
}

func TestNewClaudeExecutorWithRunner(t *testing.T) {
	mockRunner := new(MockCommandRunner)
	logger := NewLogger(LogLevelNormal)

	executor := NewClaudeExecutorWithRunner("/custom/claude", mockRunner, logger)

	require.NotNil(t, executor)
	ce, ok := executor.(*claudeExecutor)
	require.True(t, ok)
	assert.Equal(t, "/custom/claude", ce.claudePath)
	assert.Equal(t, mockRunner, ce.cmdRunner)
	assert.Equal(t, logger, ce.logger)
}

func TestClaudeExecutor_Execute_ClaudeNotFound(t *testing.T) {
	executor := &claudeExecutor{
		claudePath: "",
		cmdRunner:  command.NewRunner(),
	}

	// Temporarily override PATH to ensure claude isn't found
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent")
	defer os.Setenv("PATH", oldPath)

	ctx := context.Background()
	config := ExecuteConfig{
		Prompt: "test",
	}

	_, err := executor.Execute(ctx, config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "claude CLI not found")
}

func TestClaudeExecutor_ExecuteStreaming_ClaudeNotFound(t *testing.T) {
	executor := &claudeExecutor{
		claudePath: "",
		cmdRunner:  command.NewRunner(),
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent")
	defer os.Setenv("PATH", oldPath)

	ctx := context.Background()
	config := ExecuteConfig{
		Prompt: "test",
	}

	_, err := executor.ExecuteStreaming(ctx, config, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "claude CLI not found")
}

func TestClaudeExecutor_ExecuteStreaming_WithWorkingDirAndEnv(t *testing.T) {
	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "claude-test")
	script := `#!/bin/bash
echo '{"type":"result","result":"success","is_error":false}'
exit 0`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	executor := NewClaudeExecutorWithPath(scriptPath, NewLogger(LogLevelNormal))
	ctx := context.Background()

	config := ExecuteConfig{
		Prompt:           "test prompt",
		WorkingDirectory: tmpDir,
		Timeout:          5 * time.Second,
		Env: map[string]string{
			"TEST_VAR": "test_value",
		},
	}

	got, err := executor.ExecuteStreaming(ctx, config, nil)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Contains(t, got.Output, "success")
}

func TestClaudeExecutor_Execute_NoTimeout(t *testing.T) {
	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "claude-fast")
	script := `#!/bin/bash
echo "quick response"
exit 0`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	executor := NewClaudeExecutorWithPath(scriptPath, NewLogger(LogLevelNormal))
	ctx := context.Background()

	// No timeout set - should complete quickly
	config := ExecuteConfig{
		Prompt: "test prompt",
	}

	got, err := executor.Execute(ctx, config)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Contains(t, got.Output, "quick response")
}

func TestClaudeExecutor_ExecuteStreaming_NoTimeout(t *testing.T) {
	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "claude-fast-stream")
	script := `#!/bin/bash
echo '{"type":"result","result":"quick","is_error":false}'
exit 0`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	executor := NewClaudeExecutorWithPath(scriptPath, NewLogger(LogLevelNormal))
	ctx := context.Background()

	// No timeout set
	config := ExecuteConfig{
		Prompt: "test prompt",
	}

	got, err := executor.ExecuteStreaming(ctx, config, nil)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Contains(t, got.Output, "quick")
}

func TestExtractToolInputSummary(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		input    string
		want     string
	}{
		{
			name:     "extracts file path from Read tool",
			toolName: "Read",
			input:    `{"file_path": "/path/to/file.go"}`,
			want:     "/path/to/file.go",
		},
		{
			name:     "extracts file path from Edit tool",
			toolName: "Edit",
			input:    `{"file_path": "/path/to/edit.go", "old_string": "foo"}`,
			want:     "/path/to/edit.go",
		},
		{
			name:     "extracts file path from Write tool",
			toolName: "Write",
			input:    `{"file_path": "/path/to/write.go", "content": "test"}`,
			want:     "/path/to/write.go",
		},
		{
			name:     "extracts pattern from Glob tool",
			toolName: "Glob",
			input:    `{"pattern": "*.go"}`,
			want:     "*.go",
		},
		{
			name:     "extracts pattern from Grep tool",
			toolName: "Grep",
			input:    `{"pattern": "TODO"}`,
			want:     "TODO",
		},
		{
			name:     "extracts command from Bash tool",
			toolName: "Bash",
			input:    `{"command": "ls -la"}`,
			want:     "ls -la",
		},
		{
			name:     "extracts description from Task tool",
			toolName: "Task",
			input:    `{"description": "Run tests"}`,
			want:     "Run tests",
		},
		{
			name:     "returns empty for unknown tool",
			toolName: "UnknownTool",
			input:    `{"some_field": "value"}`,
			want:     "",
		},
		{
			name:     "returns empty for invalid JSON",
			toolName: "Read",
			input:    `invalid json`,
			want:     "",
		},
		{
			name:     "returns empty for nil input",
			toolName: "Read",
			input:    "",
			want:     "",
		},
		{
			name:     "returns empty when field is not a string",
			toolName: "Read",
			input:    `{"file_path": 123}`,
			want:     "",
		},
		{
			name:     "returns empty when field is missing",
			toolName: "Read",
			input:    `{"other_field": "value"}`,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input []byte
			if tt.input != "" {
				input = []byte(tt.input)
			}
			got := extractToolInputSummary(tt.toolName, input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClaudeExecutor_ExecuteStreaming_ScannerError(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		script      string
		wantErr     bool
		errContains string
	}{
		{
			name: "handles scanner error with very large line",
			// Generate a line that exceeds the scanner buffer (1MB = 1024*1024)
			// Use dd to create 1.5MB line without newlines (faster than python)
			script: `#!/bin/bash
dd if=/dev/zero bs=1500000 count=1 2>/dev/null | tr '\0' 'x'
exit 0`,
			wantErr:     true,
			errContains: "error reading stdout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath := filepath.Join(tmpDir, "claude-scanner-error")
			err := os.WriteFile(scriptPath, []byte(tt.script), 0755)
			require.NoError(t, err)

			executor := NewClaudeExecutorWithPath(scriptPath, NewLogger(LogLevelNormal))
			ctx := context.Background()

			config := ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 5 * time.Second,
			}

			got, err := executor.ExecuteStreaming(ctx, config, nil)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				require.NotNil(t, got)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestClaudeExecutor_ExecuteStreaming_NilFinalChunk(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		script     string
		wantOutput string
	}{
		{
			name: "handles nil finalChunk - no result type in output",
			script: `#!/bin/bash
echo '{"type":"system","subtype":"init"}'
echo '{"type":"assistant","message":{"content":[{"type":"text","text":"Hello"}]}}'
exit 0`,
			wantOutput: "",
		},
		{
			name: "handles output with only assistant messages",
			script: `#!/bin/bash
echo '{"type":"assistant","message":{"content":[{"type":"text","text":"Processing"}]}}'
echo '{"type":"assistant","message":{"content":[{"type":"text","text":"Done"}]}}'
exit 0`,
			wantOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath := filepath.Join(tmpDir, "claude-nil-chunk")
			err := os.WriteFile(scriptPath, []byte(tt.script), 0755)
			require.NoError(t, err)

			executor := NewClaudeExecutorWithPath(scriptPath, NewLogger(LogLevelNormal))
			ctx := context.Background()

			config := ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 5 * time.Second,
			}

			got, err := executor.ExecuteStreaming(ctx, config, nil)

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantOutput, got.Output)
			assert.Equal(t, 0, got.ExitCode)
		})
	}
}

func TestClaudeExecutor_ExecuteStreaming_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		script      string
		wantErr     bool
		errContains string
	}{
		{
			name: "handles context cancellation during streaming",
			script: `#!/bin/bash
trap 'exit 1' TERM INT
echo '{"type":"system","subtype":"init"}'
sleep 0.05
echo '{"type":"result","result":"should not reach here","is_error":false}'
exit 0`,
			wantErr:     true,
			errContains: "", // Error can be "context canceled" or "exit code -1" depending on timing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath := filepath.Join(tmpDir, "claude-cancel")
			err := os.WriteFile(scriptPath, []byte(tt.script), 0755)
			require.NoError(t, err)

			executor := NewClaudeExecutorWithPath(scriptPath, NewLogger(LogLevelNormal))
			ctx, cancel := context.WithCancel(context.Background())

			// Start execution in goroutine
			var got *ExecuteResult
			var execErr error
			done := make(chan struct{})
			go func() {
				got, execErr = executor.ExecuteStreaming(ctx, ExecuteConfig{Prompt: "test prompt"}, nil)
				close(done)
			}()

			// Give the script time to start and echo init message
			time.Sleep(1 * time.Millisecond)
			cancel() // cancel while script is sleeping

			<-done

			if tt.wantErr {
				require.Error(t, execErr)
				if tt.errContains != "" {
					assert.Contains(t, execErr.Error(), tt.errContains)
				}
				require.NotNil(t, got)
				return
			}

			require.NoError(t, execErr)
		})
	}
}

func TestClaudeExecutor_ExecuteStreaming_PromptTooLong(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		script      string
		wantErr     error
		errContains string
	}{
		{
			name: "detects 'Prompt is too long' error in streaming",
			script: `#!/bin/bash
echo "Error: Prompt is too long" >&2
exit 1`,
			wantErr:     ErrPromptTooLong,
			errContains: "exit code 1",
		},
		{
			name: "detects 'Prompt is too long' with context in streaming",
			script: `#!/bin/bash
echo "Failed: Prompt is too long for the model" >&2
exit 1`,
			wantErr:     ErrPromptTooLong,
			errContains: "exit code 1",
		},
		{
			name: "does not match similar errors in streaming",
			script: `#!/bin/bash
echo "Error: Prompt is short" >&2
exit 1`,
			wantErr:     ErrClaude,
			errContains: "exit code 1",
		},
		{
			name: "case sensitive - does not match different case in streaming",
			script: `#!/bin/bash
echo "Error: prompt is too long" >&2
exit 1`,
			wantErr:     ErrClaude,
			errContains: "exit code 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath := filepath.Join(tmpDir, "claude-streaming-prompt-too-long")
			err := os.WriteFile(scriptPath, []byte(tt.script), 0755)
			require.NoError(t, err)

			executor := NewClaudeExecutorWithPath(scriptPath, NewLogger(LogLevelNormal))
			ctx := context.Background()

			config := ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 5 * time.Second,
			}

			got, err := executor.ExecuteStreaming(ctx, config, nil)

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
			if tt.errContains != "" {
				assert.Contains(t, err.Error(), tt.errContains)
			}
			require.NotNil(t, got)
		})
	}
}

func TestClaudeExecutor_Execute_WithTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test in short mode")
	}

	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		script      string
		timeout     time.Duration
		wantErr     bool
		errContains string
	}{
		{
			name: "respects timeout setting in Execute",
			script: `#!/bin/bash
sleep 0.05
echo "completed"
exit 0`,
			timeout:     20 * time.Millisecond,
			wantErr:     true,
			errContains: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath := filepath.Join(tmpDir, "claude-timeout-test")
			err := os.WriteFile(scriptPath, []byte(tt.script), 0755)
			require.NoError(t, err)

			executor := NewClaudeExecutorWithPath(scriptPath, NewLogger(LogLevelNormal))
			ctx := context.Background()

			config := ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: tt.timeout,
			}

			got, err := executor.Execute(ctx, config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				require.NotNil(t, got)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestClaudeExecutor_Execute_NonZeroExitCode(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		script      string
		exitCode    int
		wantErr     bool
		errContains string
	}{
		{
			name: "handles exit code 1",
			script: `#!/bin/bash
echo "error occurred" >&2
exit 1`,
			exitCode:    1,
			wantErr:     true,
			errContains: "exit code 1",
		},
		{
			name: "handles exit code 2",
			script: `#!/bin/bash
echo "critical error" >&2
exit 2`,
			exitCode:    2,
			wantErr:     true,
			errContains: "exit code 2",
		},
		{
			name: "handles exit code 127",
			script: `#!/bin/bash
echo "command not found" >&2
exit 127`,
			exitCode:    127,
			wantErr:     true,
			errContains: "exit code 127",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath := filepath.Join(tmpDir, "claude-exit-code")
			err := os.WriteFile(scriptPath, []byte(tt.script), 0755)
			require.NoError(t, err)

			executor := NewClaudeExecutorWithPath(scriptPath, NewLogger(LogLevelNormal))
			ctx := context.Background()

			config := ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 5 * time.Second,
			}

			got, err := executor.Execute(ctx, config)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.exitCode, got.ExitCode)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestClaudeExecutor_Execute_PromptTooLong(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		script      string
		wantErr     error
		errContains string
	}{
		{
			name: "detects 'Prompt is too long' error",
			script: `#!/bin/bash
echo "Error: Prompt is too long" >&2
exit 1`,
			wantErr:     ErrPromptTooLong,
			errContains: "exit code 1",
		},
		{
			name: "detects 'Prompt is too long' with context",
			script: `#!/bin/bash
echo "Failed to execute: Prompt is too long (max 100k tokens)" >&2
exit 1`,
			wantErr:     ErrPromptTooLong,
			errContains: "exit code 1",
		},
		{
			name: "does not match similar errors",
			script: `#!/bin/bash
echo "Error: Prompt too short" >&2
exit 1`,
			wantErr:     ErrClaude,
			errContains: "exit code 1",
		},
		{
			name: "case sensitive - does not match different case",
			script: `#!/bin/bash
echo "Error: PROMPT IS TOO LONG" >&2
exit 1`,
			wantErr:     ErrClaude,
			errContains: "exit code 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath := filepath.Join(tmpDir, "claude-prompt-too-long")
			err := os.WriteFile(scriptPath, []byte(tt.script), 0755)
			require.NoError(t, err)

			executor := NewClaudeExecutorWithPath(scriptPath, NewLogger(LogLevelNormal))
			ctx := context.Background()

			config := ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 5 * time.Second,
			}

			got, err := executor.Execute(ctx, config)

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
			if tt.errContains != "" {
				assert.Contains(t, err.Error(), tt.errContains)
			}
			require.NotNil(t, got)
		})
	}
}
