package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			got := NewClaudeExecutor()
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
			got := NewClaudeExecutorWithPath(tt.claudePath)
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
		wantErr    bool
	}{
		{
			name:       "returns custom path when set",
			claudePath: "/usr/local/bin/claude",
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
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.claudePath, got)
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
