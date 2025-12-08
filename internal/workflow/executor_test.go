package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExecutor is a mock implementation of ClaudeExecutor for testing
type mockExecutor struct {
	executeFunc          func(ctx context.Context, config ExecuteConfig) (*ExecuteResult, error)
	executeStreamingFunc func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error)
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

func (m *mockExecutor) ExecuteStreaming(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
	if m.executeStreamingFunc != nil {
		return m.executeStreamingFunc(ctx, config, onProgress)
	}
	return &ExecuteResult{
		Output:   "mock streaming output",
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

func TestExtractToolInputSummary(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		input    []byte
		want     string
	}{
		{
			name:     "Read tool with file_path returns file path",
			toolName: "Read",
			input:    []byte(`{"file_path": "/home/user/test.go"}`),
			want:     "/home/user/test.go",
		},
		{
			name:     "Edit tool with file_path returns file path",
			toolName: "Edit",
			input:    []byte(`{"file_path": "/home/user/main.go"}`),
			want:     "/home/user/main.go",
		},
		{
			name:     "Write tool with file_path returns file path",
			toolName: "Write",
			input:    []byte(`{"file_path": "/home/user/output.txt"}`),
			want:     "/home/user/output.txt",
		},
		{
			name:     "Glob tool with pattern returns pattern",
			toolName: "Glob",
			input:    []byte(`{"pattern": "**/*.go"}`),
			want:     "**/*.go",
		},
		{
			name:     "Grep tool with pattern returns pattern",
			toolName: "Grep",
			input:    []byte(`{"pattern": "func.*Error"}`),
			want:     "func.*Error",
		},
		{
			name:     "Bash tool with command returns command",
			toolName: "Bash",
			input:    []byte(`{"command": "go test ./..."}`),
			want:     "go test ./...",
		},
		{
			name:     "Task tool with description returns description",
			toolName: "Task",
			input:    []byte(`{"description": "run tests"}`),
			want:     "run tests",
		},
		{
			name:     "unknown tool returns empty string",
			toolName: "UnknownTool",
			input:    []byte(`{"some_field": "value"}`),
			want:     "",
		},
		{
			name:     "nil input returns empty string",
			toolName: "Read",
			input:    nil,
			want:     "",
		},
		{
			name:     "invalid JSON input returns empty string",
			toolName: "Read",
			input:    []byte(`{invalid json`),
			want:     "",
		},
		{
			name:     "missing expected field returns empty string",
			toolName: "Read",
			input:    []byte(`{"other_field": "value"}`),
			want:     "",
		},
		{
			name:     "Read tool with non-string file_path returns empty string",
			toolName: "Read",
			input:    []byte(`{"file_path": 123}`),
			want:     "",
		},
		{
			name:     "Edit tool with non-string file_path returns empty string",
			toolName: "Edit",
			input:    []byte(`{"file_path": true}`),
			want:     "",
		},
		{
			name:     "Write tool with non-string file_path returns empty string",
			toolName: "Write",
			input:    []byte(`{"file_path": ["array"]}`),
			want:     "",
		},
		{
			name:     "Glob tool with non-string pattern returns empty string",
			toolName: "Glob",
			input:    []byte(`{"pattern": null}`),
			want:     "",
		},
		{
			name:     "Grep tool with non-string pattern returns empty string",
			toolName: "Grep",
			input:    []byte(`{"pattern": {"nested": "object"}}`),
			want:     "",
		},
		{
			name:     "Bash tool with non-string command returns empty string",
			toolName: "Bash",
			input:    []byte(`{"command": 456}`),
			want:     "",
		},
		{
			name:     "Task tool with non-string description returns empty string",
			toolName: "Task",
			input:    []byte(`{"description": false}`),
			want:     "",
		},
		{
			name:     "Read tool with additional fields extracts file_path",
			toolName: "Read",
			input:    []byte(`{"file_path": "/test.go", "other": "value", "limit": 100}`),
			want:     "/test.go",
		},
		{
			name:     "Bash tool with complex command string",
			toolName: "Bash",
			input:    []byte(`{"command": "cd /tmp && go test -v -race ./..."}`),
			want:     "cd /tmp && go test -v -race ./...",
		},
		{
			name:     "Grep tool with regex pattern containing special chars",
			toolName: "Grep",
			input:    []byte(`{"pattern": "func\\s+Test.*\\(t\\s+\\*testing\\.T\\)"}`),
			want:     "func\\s+Test.*\\(t\\s+\\*testing\\.T\\)",
		},
		{
			name:     "empty JSON object returns empty string",
			toolName: "Read",
			input:    []byte(`{}`),
			want:     "",
		},
		{
			name:     "empty string input returns empty string",
			toolName: "Read",
			input:    []byte(``),
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractToolInputSummary(tt.toolName, tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "string shorter than maxLen unchanged",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "string equal to maxLen unchanged",
			input:  "hello world",
			maxLen: 11,
			want:   "hello world",
		},
		{
			name:   "string longer than maxLen truncated with ellipsis",
			input:  "hello world this is a long string",
			maxLen: 15,
			want:   "hello world ...",
		},
		{
			name:   "empty string returns empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "maxLen of 3 returns ellipsis only",
			input:  "hello",
			maxLen: 3,
			want:   "...",
		},
		{
			name:   "maxLen of 4 returns single char plus ellipsis",
			input:  "hello",
			maxLen: 4,
			want:   "h...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMockExecutor_ExecuteStreaming_Success(t *testing.T) {
	tests := []struct {
		name              string
		config            ExecuteConfig
		mockFunc          func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error)
		wantOutput        string
		wantProgressCalls int
		wantErr           bool
	}{
		{
			name: "executes successfully with basic streaming",
			config: ExecuteConfig{
				Prompt:           "test prompt",
				WorkingDirectory: "/tmp",
				Timeout:          5 * time.Second,
			},
			mockFunc: func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
				if onProgress != nil {
					onProgress(ProgressEvent{
						Type:      "tool_use",
						ToolName:  "Read",
						ToolInput: "/test/file.go",
					})
					onProgress(ProgressEvent{
						Type: "text",
						Text: "Reading file...",
					})
				}
				return &ExecuteResult{
					Output:   "test output",
					ExitCode: 0,
					Duration: 50 * time.Millisecond,
				}, nil
			},
			wantOutput:        "test output",
			wantProgressCalls: 2,
			wantErr:           false,
		},
		{
			name: "executes successfully with JSON schema",
			config: ExecuteConfig{
				Prompt:     "test prompt",
				JSONSchema: `{"type": "object"}`,
				Timeout:    5 * time.Second,
			},
			mockFunc: func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
				if config.JSONSchema == "" {
					return nil, errors.New("expected JSONSchema to be set")
				}
				if onProgress != nil {
					onProgress(ProgressEvent{
						Type: "text",
						Text: "Processing...",
					})
				}
				return &ExecuteResult{
					Output:   `{"result": "success"}`,
					ExitCode: 0,
					Duration: 50 * time.Millisecond,
				}, nil
			},
			wantOutput:        `{"result": "success"}`,
			wantProgressCalls: 1,
			wantErr:           false,
		},
		{
			name: "handles multiple tool use events",
			config: ExecuteConfig{
				Prompt: "test prompt",
			},
			mockFunc: func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
				if onProgress != nil {
					onProgress(ProgressEvent{Type: "tool_use", ToolName: "Read", ToolInput: "/file1.go"})
					onProgress(ProgressEvent{Type: "tool_use", ToolName: "Edit", ToolInput: "/file2.go"})
					onProgress(ProgressEvent{Type: "tool_use", ToolName: "Bash", ToolInput: "go test"})
					onProgress(ProgressEvent{Type: "tool_result", Text: "Success"})
				}
				return &ExecuteResult{
					Output:   "completed",
					ExitCode: 0,
					Duration: 100 * time.Millisecond,
				}, nil
			},
			wantOutput:        "completed",
			wantProgressCalls: 4,
			wantErr:           false,
		},
		{
			name: "handles tool result with error",
			config: ExecuteConfig{
				Prompt: "test prompt",
			},
			mockFunc: func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
				if onProgress != nil {
					onProgress(ProgressEvent{
						Type:    "tool_result",
						Text:    "Error: file not found",
						IsError: true,
					})
				}
				return &ExecuteResult{
					Output:   "failed",
					ExitCode: 0,
					Duration: 50 * time.Millisecond,
				}, nil
			},
			wantOutput:        "failed",
			wantProgressCalls: 1,
			wantErr:           false,
		},
		{
			name: "handles execution timeout",
			config: ExecuteConfig{
				Prompt:  "test prompt",
				Timeout: 1 * time.Millisecond,
			},
			mockFunc: func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
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
			mockFunc: func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
				return &ExecuteResult{
					ExitCode: 1,
					Error:    errors.New("execution failed"),
				}, ErrClaude
			},
			wantErr: true,
		},
		{
			name: "executes without progress callback",
			config: ExecuteConfig{
				Prompt: "test prompt",
			},
			mockFunc: func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
				return &ExecuteResult{
					Output:   "output without progress",
					ExitCode: 0,
					Duration: 50 * time.Millisecond,
				}, nil
			},
			wantOutput:        "output without progress",
			wantProgressCalls: 0,
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progressCalls := 0
			var progressCallback func(ProgressEvent)
			if tt.wantProgressCalls > 0 {
				progressCallback = func(event ProgressEvent) {
					progressCalls++
				}
			}

			executor := &mockExecutor{
				executeStreamingFunc: tt.mockFunc,
			}

			ctx := context.Background()
			got, err := executor.ExecuteStreaming(ctx, tt.config, progressCallback)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantOutput, got.Output)
			assert.Equal(t, 0, got.ExitCode)
			assert.Equal(t, tt.wantProgressCalls, progressCalls)
		})
	}
}

func TestMockExecutor_ExecuteStreaming_ProgressEvents(t *testing.T) {
	tests := []struct {
		name          string
		events        []ProgressEvent
		wantEventType []string
	}{
		{
			name: "tracks all tool types",
			events: []ProgressEvent{
				{Type: "tool_use", ToolName: "Read", ToolInput: "/file.go"},
				{Type: "tool_use", ToolName: "Write", ToolInput: "/output.txt"},
				{Type: "tool_use", ToolName: "Edit", ToolInput: "/main.go"},
				{Type: "tool_use", ToolName: "Bash", ToolInput: "go build"},
				{Type: "tool_use", ToolName: "Grep", ToolInput: "pattern"},
				{Type: "tool_use", ToolName: "Glob", ToolInput: "**/*.go"},
			},
			wantEventType: []string{"tool_use", "tool_use", "tool_use", "tool_use", "tool_use", "tool_use"},
		},
		{
			name: "tracks text and thinking events",
			events: []ProgressEvent{
				{Type: "text", Text: "Analyzing code..."},
				{Type: "thinking", Text: "Considering approach..."},
				{Type: "text", Text: "Making changes..."},
			},
			wantEventType: []string{"text", "thinking", "text"},
		},
		{
			name: "tracks tool results",
			events: []ProgressEvent{
				{Type: "tool_result", Text: "File read successfully", IsError: false},
				{Type: "tool_result", Text: "Error: file not found", IsError: true},
			},
			wantEventType: []string{"tool_result", "tool_result"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedEvents []ProgressEvent
			progressCallback := func(event ProgressEvent) {
				capturedEvents = append(capturedEvents, event)
			}

			executor := &mockExecutor{
				executeStreamingFunc: func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
					if onProgress != nil {
						for _, event := range tt.events {
							onProgress(event)
						}
					}
					return &ExecuteResult{
						Output:   "success",
						ExitCode: 0,
					}, nil
				},
			}

			ctx := context.Background()
			config := ExecuteConfig{Prompt: "test"}

			_, err := executor.ExecuteStreaming(ctx, config, progressCallback)

			require.NoError(t, err)
			require.Len(t, capturedEvents, len(tt.wantEventType))

			for i, event := range capturedEvents {
				assert.Equal(t, tt.wantEventType[i], event.Type)
			}
		})
	}
}

func TestMockExecutor_ExecuteStreaming_Timeout(t *testing.T) {
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
				executeStreamingFunc: func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
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

			_, err := executor.ExecuteStreaming(ctx, config, nil)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrClaudeTimeout)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestMockExecutor_ExecuteStreaming_WithEnv(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{
			name: "executes with environment variables",
			env: map[string]string{
				"TEST_VAR": "test_value",
				"DEBUG":    "true",
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
				executeStreamingFunc: func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
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

			got, err := executor.ExecuteStreaming(ctx, config, nil)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "success", got.Output)
		})
	}
}

func TestMockExecutor_ExecuteStreaming_ExitCode(t *testing.T) {
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
				executeStreamingFunc: func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
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

			got, err := executor.ExecuteStreaming(ctx, config, nil)

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

func TestMockExecutor_ExecuteStreaming_StructuredOutput(t *testing.T) {
	tests := []struct {
		name           string
		config         ExecuteConfig
		mockFunc       func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error)
		wantOutputJSON bool
		wantErr        bool
	}{
		{
			name: "returns structured output in envelope format",
			config: ExecuteConfig{
				Prompt:     "test prompt",
				JSONSchema: `{"type": "object", "properties": {"name": {"type": "string"}}}`,
			},
			mockFunc: func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
				envelope := `{"type":"result","result":"Success","structured_output":{"name":"test"},"is_error":false}`
				return &ExecuteResult{
					Output:   envelope,
					ExitCode: 0,
					Duration: 50 * time.Millisecond,
				}, nil
			},
			wantOutputJSON: true,
			wantErr:        false,
		},
		{
			name: "returns plain text when no structured output",
			config: ExecuteConfig{
				Prompt: "test prompt",
			},
			mockFunc: func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
				return &ExecuteResult{
					Output:   "plain text result",
					ExitCode: 0,
					Duration: 50 * time.Millisecond,
				}, nil
			},
			wantOutputJSON: false,
			wantErr:        false,
		},
		{
			name: "handles structured output with error flag",
			config: ExecuteConfig{
				Prompt:     "test prompt",
				JSONSchema: `{"type": "object"}`,
			},
			mockFunc: func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
				envelope := `{"type":"result","result":"Error occurred","structured_output":{},"is_error":true}`
				return &ExecuteResult{
					Output:   envelope,
					ExitCode: 0,
					Duration: 50 * time.Millisecond,
				}, nil
			},
			wantOutputJSON: true,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &mockExecutor{
				executeStreamingFunc: tt.mockFunc,
			}

			ctx := context.Background()
			got, err := executor.ExecuteStreaming(ctx, tt.config, nil)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.NotEmpty(t, got.Output)

			if tt.wantOutputJSON {
				var envelope map[string]interface{}
				err := json.Unmarshal([]byte(got.Output), &envelope)
				require.NoError(t, err)
				assert.Equal(t, "result", envelope["type"])
			}
		})
	}
}

func TestMockExecutor_Execute_DangerouslySkipPermissions(t *testing.T) {
	tests := []struct {
		name                       string
		dangerouslySkipPermissions bool
		wantErr                    bool
	}{
		{
			name:                       "executes with DangerouslySkipPermissions enabled",
			dangerouslySkipPermissions: true,
			wantErr:                    false,
		},
		{
			name:                       "executes with DangerouslySkipPermissions disabled",
			dangerouslySkipPermissions: false,
			wantErr:                    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &mockExecutor{
				executeFunc: func(ctx context.Context, config ExecuteConfig) (*ExecuteResult, error) {
					assert.Equal(t, tt.dangerouslySkipPermissions, config.DangerouslySkipPermissions)
					return &ExecuteResult{
						Output:   "success",
						ExitCode: 0,
						Duration: 50 * time.Millisecond,
					}, nil
				},
			}

			ctx := context.Background()
			config := ExecuteConfig{
				Prompt:                     "test",
				DangerouslySkipPermissions: tt.dangerouslySkipPermissions,
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

func TestMockExecutor_ExecuteStreaming_DangerouslySkipPermissions(t *testing.T) {
	tests := []struct {
		name                       string
		dangerouslySkipPermissions bool
		wantErr                    bool
	}{
		{
			name:                       "executes with DangerouslySkipPermissions enabled",
			dangerouslySkipPermissions: true,
			wantErr:                    false,
		},
		{
			name:                       "executes with DangerouslySkipPermissions disabled",
			dangerouslySkipPermissions: false,
			wantErr:                    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &mockExecutor{
				executeStreamingFunc: func(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
					assert.Equal(t, tt.dangerouslySkipPermissions, config.DangerouslySkipPermissions)
					return &ExecuteResult{
						Output:   "success",
						ExitCode: 0,
						Duration: 50 * time.Millisecond,
					}, nil
				},
			}

			ctx := context.Background()
			config := ExecuteConfig{
				Prompt:                     "test",
				DangerouslySkipPermissions: tt.dangerouslySkipPermissions,
			}

			got, err := executor.ExecuteStreaming(ctx, config, nil)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "success", got.Output)
		})
	}
}
