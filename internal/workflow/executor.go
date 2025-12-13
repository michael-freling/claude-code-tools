package workflow

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/michael-freling/claude-code-tools/internal/command"
)

// ClaudeExecutor interface allows mocking of Claude CLI invocation
type ClaudeExecutor interface {
	Execute(ctx context.Context, config ExecuteConfig) (*ExecuteResult, error)
	ExecuteStreaming(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error)
}

// ProgressEvent represents a progress update during Claude execution
type ProgressEvent struct {
	Type      string // "tool_use", "tool_result", "text", "thinking"
	ToolName  string // For tool_use: Read, Edit, Grep, etc.
	ToolInput string // Brief description of tool input (e.g., file path)
	Text      string // For text/thinking: the content
	IsError   bool   // Whether this is an error result
}

// StreamChunk represents a parsed stream-json chunk from Claude CLI
type StreamChunk struct {
	Type             string          `json:"type"`              // system, assistant, user, result
	Subtype          string          `json:"subtype"`           // init, success, etc.
	Message          *MessageChunk   `json:"message"`           // For assistant/user types
	ToolUseResult    string          `json:"tool_use_result"`   // For user type (tool results)
	Result           string          `json:"result"`            // For result type
	StructuredOutput json.RawMessage `json:"structured_output"` // For result type with schema
	IsError          bool            `json:"is_error"`
}

// MessageChunk represents the message field in a stream chunk
type MessageChunk struct {
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents a content block in a message
type ContentBlock struct {
	Type  string          `json:"type"`  // text, tool_use
	Text  string          `json:"text"`  // For text type
	Name  string          `json:"name"`  // For tool_use type (tool name)
	Input json.RawMessage `json:"input"` // For tool_use type (tool args)
}

// ExecuteConfig holds configuration for executing Claude CLI
type ExecuteConfig struct {
	Prompt                     string
	WorkingDirectory           string
	Timeout                    time.Duration
	Env                        map[string]string
	JSONSchema                 string
	DangerouslySkipPermissions bool
	SessionID                  string // Session ID to resume (empty for new session)
	ForceNewSession            bool   // Force creating a new session even if SessionID is set
}

// ExecuteResult holds the result of Claude CLI execution
type ExecuteResult struct {
	Output    string // Parsed output (final result or structured output)
	RawOutput string // Raw streaming output for session ID parsing
	ExitCode  int
	Duration  time.Duration
	Error     error
}

// claudeExecutor implements ClaudeExecutor interface
type claudeExecutor struct {
	claudePath string
	cmdRunner  command.Runner
	logger     Logger
}

// NewClaudeExecutor creates executor with default settings
func NewClaudeExecutor(logger Logger) ClaudeExecutor {
	return &claudeExecutor{
		claudePath: "claude",
		cmdRunner:  command.NewRunner(),
		logger:     logger,
	}
}

// NewClaudeExecutorWithPath creates executor with custom claude path
func NewClaudeExecutorWithPath(claudePath string, logger Logger) ClaudeExecutor {
	return &claudeExecutor{
		claudePath: claudePath,
		cmdRunner:  command.NewRunner(),
		logger:     logger,
	}
}

// NewClaudeExecutorWithRunner creates executor with custom command runner (for testing)
func NewClaudeExecutorWithRunner(claudePath string, cmdRunner command.Runner, logger Logger) ClaudeExecutor {
	return &claudeExecutor{
		claudePath: claudePath,
		cmdRunner:  cmdRunner,
		logger:     logger,
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

	args := []string{"--print"}
	if config.DangerouslySkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}
	if config.JSONSchema != "" {
		args = append(args, "--output-format", "json", "--json-schema", config.JSONSchema)
	}
	args = append(args, config.Prompt)
	cmd := exec.CommandContext(ctx, claudePath, args...)

	if config.WorkingDirectory != "" {
		cmd.Dir = config.WorkingDirectory
	}

	if len(config.Env) > 0 {
		cmd.Env = append(cmd.Env, e.buildEnv(config.Env)...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = nil // Prevent subprocess from reading parent's stdin

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
			stderrStr := stderr.String()
			result.Error = fmt.Errorf("%s", stderrStr)

			if strings.Contains(stderrStr, "Prompt is too long") {
				return result, fmt.Errorf("claude execution failed with exit code %d: %w", result.ExitCode, ErrPromptTooLong)
			}

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

// ExecuteStreaming runs the Claude CLI with streaming output and progress callbacks
func (e *claudeExecutor) ExecuteStreaming(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
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

	// Build args for streaming mode: requires --verbose with stream-json
	args := []string{"--print", "--output-format", "stream-json", "--verbose"}
	if config.DangerouslySkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}
	if config.JSONSchema != "" {
		args = append(args, "--json-schema", config.JSONSchema)
	}
	// Add session resume args if session ID is provided and not forcing new session
	if !config.ForceNewSession && config.SessionID != "" {
		args = append(args, "--resume", config.SessionID)
		if e.logger != nil {
			e.logger.Verbose("Resuming Claude session: %s", config.SessionID)
		}
	}
	args = append(args, config.Prompt)
	cmd := exec.CommandContext(ctx, claudePath, args...)

	if e.logger != nil {
		e.logger.Verbose("Executing Claude CLI:")
		workingDir := config.WorkingDirectory
		if workingDir == "" {
			workingDir = "."
		}
		e.logger.Verbose("  Working directory: %s", workingDir)
		if config.Timeout > 0 {
			e.logger.Verbose("  Timeout: %s", config.Timeout)
		} else {
			e.logger.Verbose("  Timeout: none")
		}
	}

	if config.WorkingDirectory != "" {
		cmd.Dir = config.WorkingDirectory
	}

	if len(config.Env) > 0 {
		cmd.Env = append(cmd.Env, e.buildEnv(config.Env)...)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		result.Error = err
		return result, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		result.Error = err
		return result, fmt.Errorf("failed to start claude: %w", err)
	}

	// Read and parse streaming output
	scanner := bufio.NewScanner(stdout)
	// Increase buffer size for large JSON lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var finalChunk *StreamChunk
	toolCallCount := 0
	var allOutput strings.Builder

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Collect all output for session ID parsing
		allOutput.Write(line)
		allOutput.WriteByte('\n')

		var chunk StreamChunk
		if err := json.Unmarshal(line, &chunk); err != nil {
			// Skip malformed lines
			continue
		}

		switch chunk.Type {
		case "result":
			finalChunk = &chunk
		case "assistant":
			if chunk.Message != nil {
				for _, content := range chunk.Message.Content {
					switch content.Type {
					case "tool_use":
						toolCallCount++
						if onProgress != nil {
							onProgress(ProgressEvent{
								Type:      "tool_use",
								ToolName:  content.Name,
								ToolInput: extractToolInputSummary(content.Name, content.Input),
							})
						}
					case "text":
						if onProgress != nil && content.Text != "" {
							onProgress(ProgressEvent{
								Type: "text",
								Text: content.Text,
							})
						}
					}
				}
			}
		case "user":
			// Tool results
			if chunk.ToolUseResult != "" && onProgress != nil {
				isError := len(chunk.ToolUseResult) > 6 && chunk.ToolUseResult[:6] == "Error:"
				onProgress(ProgressEvent{
					Type:    "tool_result",
					Text:    chunk.ToolUseResult,
					IsError: isError,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		result.Error = err
		return result, fmt.Errorf("error reading stdout: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		result.Duration = time.Since(start)
		if ctx.Err() == context.DeadlineExceeded {
			result.Error = ErrClaudeTimeout
			return result, fmt.Errorf("claude execution timeout after %s: %w", result.Duration, ErrClaudeTimeout)
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			stderrStr := stderr.String()
			result.Error = fmt.Errorf("%s", stderrStr)

			if strings.Contains(stderrStr, "Prompt is too long") {
				return result, fmt.Errorf("claude execution failed with exit code %d: %w", result.ExitCode, ErrPromptTooLong)
			}

			return result, fmt.Errorf("claude execution failed with exit code %d: %w", result.ExitCode, ErrClaude)
		}

		result.Error = err
		return result, fmt.Errorf("failed to execute claude: %w", err)
	}

	result.Duration = time.Since(start)
	result.ExitCode = 0
	result.RawOutput = allOutput.String()

	// Extract output from final chunk
	if finalChunk != nil {
		if len(finalChunk.StructuredOutput) > 0 {
			// Wrap structured output in the expected envelope format
			envelope := map[string]interface{}{
				"type":              "result",
				"result":            finalChunk.Result,
				"structured_output": finalChunk.StructuredOutput,
				"is_error":          finalChunk.IsError,
			}
			envelopeBytes, err := json.Marshal(envelope)
			if err != nil {
				result.Error = err
				return result, fmt.Errorf("failed to marshal structured output envelope: %w", err)
			}
			result.Output = string(envelopeBytes)
		} else {
			result.Output = finalChunk.Result
		}
	}

	if e.logger != nil {
		charCount := len(result.Output)
		e.logger.Verbose("Claude response received (%s characters, %d tool calls)", formatNumber(charCount), toolCallCount)
	}

	return result, nil
}

// extractToolInputSummary extracts a brief summary of tool input for display
func extractToolInputSummary(toolName string, input json.RawMessage) string {
	if input == nil {
		return ""
	}

	var data map[string]interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		return ""
	}

	switch toolName {
	case "Read":
		if path, ok := data["file_path"].(string); ok {
			return path
		}
	case "Edit":
		if path, ok := data["file_path"].(string); ok {
			return path
		}
	case "Write":
		if path, ok := data["file_path"].(string); ok {
			return path
		}
	case "Glob":
		if pattern, ok := data["pattern"].(string); ok {
			return pattern
		}
	case "Grep":
		if pattern, ok := data["pattern"].(string); ok {
			return pattern
		}
	case "Bash":
		if cmd, ok := data["command"].(string); ok {
			return cmd
		}
	case "Task":
		if desc, ok := data["description"].(string); ok {
			return desc
		}
	}

	return ""
}

// formatNumber formats a number with thousand separators
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%s,%03d", formatNumber(n/1000), n%1000)
}
