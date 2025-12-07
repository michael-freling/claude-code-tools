package workflow

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// ClaudeExecutor interface allows mocking of Claude CLI invocation
type ClaudeExecutor interface {
	Execute(ctx context.Context, config ExecuteConfig) (*ExecuteResult, error)
}

// ExecuteConfig holds configuration for executing Claude CLI
type ExecuteConfig struct {
	Prompt           string
	WorkingDirectory string
	Timeout          time.Duration
	Env              map[string]string
}

// ExecuteResult holds the result of Claude CLI execution
type ExecuteResult struct {
	Output   string
	ExitCode int
	Duration time.Duration
	Error    error
}

// claudeExecutor implements ClaudeExecutor interface
type claudeExecutor struct {
	claudePath string
}

// NewClaudeExecutor creates executor with default settings
func NewClaudeExecutor() ClaudeExecutor {
	return &claudeExecutor{
		claudePath: "claude",
	}
}

// NewClaudeExecutorWithPath creates executor with custom claude path
func NewClaudeExecutorWithPath(claudePath string) ClaudeExecutor {
	return &claudeExecutor{
		claudePath: claudePath,
	}
}

// Execute runs the Claude CLI with the given configuration
func (e *claudeExecutor) Execute(ctx context.Context, config ExecuteConfig) (*ExecuteResult, error) {
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}

	start := time.Now()
	result := &ExecuteResult{}

	claudePath, err := e.findClaudePath()
	if err != nil {
		result.Error = err
		return result, fmt.Errorf("claude CLI not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, claudePath, "--print", config.Prompt)

	if config.WorkingDirectory != "" {
		cmd.Dir = config.WorkingDirectory
	}

	if len(config.Env) > 0 {
		cmd.Env = append(cmd.Env, e.buildEnv(config.Env)...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	result.Duration = time.Since(start)
	result.Output = stdout.String()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.Error = ErrClaudeTimeout
			return result, fmt.Errorf("claude execution timeout after %s: %w", result.Duration, ErrClaudeTimeout)
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Error = fmt.Errorf("%s", stderr.String())
			return result, fmt.Errorf("claude execution failed with exit code %d: %w", result.ExitCode, ErrClaude)
		}

		result.Error = err
		return result, fmt.Errorf("failed to execute claude: %w", err)
	}

	result.ExitCode = 0
	return result, nil
}

// findClaudePath locates the claude executable in PATH
func (e *claudeExecutor) findClaudePath() (string, error) {
	if e.claudePath != "" && e.claudePath != "claude" {
		return e.claudePath, nil
	}

	path, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude CLI not found in PATH: %w", ErrClaudeNotFound)
	}

	return path, nil
}

// buildEnv converts environment map to slice of KEY=VALUE strings
func (e *claudeExecutor) buildEnv(env map[string]string) []string {
	result := make([]string, 0, len(env))
	for key, value := range env {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}
	return result
}
