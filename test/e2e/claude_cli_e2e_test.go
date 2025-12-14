//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/michael-freling/claude-code-tools/internal/workflow"
	"github.com/michael-freling/claude-code-tools/test/e2e/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealClaude_SimpleExecution tests basic Claude CLI execution
func TestRealClaude_SimpleExecution(t *testing.T) {
	helpers.RequireClaude(t)

	tests := []struct {
		name           string
		prompt         string
		wantInResponse string
		timeout        time.Duration
	}{
		{
			name:           "simple ping pong",
			prompt:         "Reply with exactly: PONG",
			wantInResponse: "PONG",
			timeout:        30 * time.Second,
		},
		{
			name:           "simple math calculation",
			prompt:         "What is 2+2? Reply with just the number.",
			wantInResponse: "4",
			timeout:        30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := workflow.NewLogger(workflow.LogLevelNormal)
			executor := workflow.NewClaudeExecutor(logger)

			ctx := context.Background()
			result, err := executor.Execute(ctx, workflow.ExecuteConfig{
				Prompt:  tt.prompt,
				Timeout: tt.timeout,
			})

			require.NoError(t, err, "claude execution should succeed")
			require.NotNil(t, result)
			assert.Equal(t, 0, result.ExitCode, "exit code should be 0")
			assert.NotEmpty(t, result.Output, "output should not be empty")
			assert.Contains(t, result.Output, tt.wantInResponse, "response should contain expected content")
			assert.Greater(t, result.Duration, time.Duration(0), "duration should be positive")
		})
	}
}

// TestRealClaude_StreamingMode tests Claude CLI execution with streaming
func TestRealClaude_StreamingMode(t *testing.T) {
	helpers.RequireClaude(t)

	tests := []struct {
		name              string
		prompt            string
		wantInResponse    string
		wantProgressEvent bool
		timeout           time.Duration
	}{
		{
			name:              "streaming with simple response",
			prompt:            "Reply with: Hello from streaming mode",
			wantInResponse:    "Hello",
			wantProgressEvent: true,
			timeout:           30 * time.Second,
		},
		{
			name:              "streaming with calculation",
			prompt:            "Calculate 10 * 10 and explain the result briefly",
			wantInResponse:    "100",
			wantProgressEvent: true,
			timeout:           30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := workflow.NewLogger(workflow.LogLevelNormal)
			executor := workflow.NewClaudeExecutor(logger)

			progressReceived := false
			onProgress := func(event workflow.ProgressEvent) {
				progressReceived = true
				// Verify progress event has valid structure
				assert.NotEmpty(t, event.Type, "progress event should have a type")
			}

			ctx := context.Background()
			result, err := executor.ExecuteStreaming(ctx, workflow.ExecuteConfig{
				Prompt:  tt.prompt,
				Timeout: tt.timeout,
			}, onProgress)

			require.NoError(t, err, "claude streaming execution should succeed")
			require.NotNil(t, result)
			assert.Equal(t, 0, result.ExitCode, "exit code should be 0")
			assert.NotEmpty(t, result.Output, "output should not be empty")
			assert.Contains(t, result.Output, tt.wantInResponse, "response should contain expected content")

			if tt.wantProgressEvent {
				assert.True(t, progressReceived, "should receive progress events")
			}
		})
	}
}

// TestRealClaude_JSONSchema tests Claude CLI with structured output
func TestRealClaude_JSONSchema(t *testing.T) {
	helpers.RequireClaude(t)

	tests := []struct {
		name           string
		prompt         string
		schema         string
		wantFields     []string
		validateResult func(t *testing.T, data map[string]interface{})
		timeout        time.Duration
	}{
		{
			name:   "simple object schema",
			prompt: "Provide a summary with title and content about Go programming",
			schema: `{
				"type": "object",
				"properties": {
					"title": {"type": "string"},
					"content": {"type": "string"}
				},
				"required": ["title", "content"]
			}`,
			wantFields: []string{"title", "content"},
			validateResult: func(t *testing.T, data map[string]interface{}) {
				assert.NotEmpty(t, data["title"], "title should not be empty")
				assert.NotEmpty(t, data["content"], "content should not be empty")
			},
			timeout: 30 * time.Second,
		},
		{
			name:   "object with array",
			prompt: "List three programming languages with their types (compiled/interpreted)",
			schema: `{
				"type": "object",
				"properties": {
					"languages": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"name": {"type": "string"},
								"type": {"type": "string"}
							},
							"required": ["name", "type"]
						}
					}
				},
				"required": ["languages"]
			}`,
			wantFields: []string{"languages"},
			validateResult: func(t *testing.T, data map[string]interface{}) {
				languages, ok := data["languages"].([]interface{})
				require.True(t, ok, "languages should be an array")
				assert.GreaterOrEqual(t, len(languages), 1, "should have at least one language")

				if len(languages) > 0 {
					firstLang, ok := languages[0].(map[string]interface{})
					require.True(t, ok, "language entry should be an object")
					assert.NotEmpty(t, firstLang["name"], "language name should not be empty")
					assert.NotEmpty(t, firstLang["type"], "language type should not be empty")
				}
			},
			timeout: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := workflow.NewLogger(workflow.LogLevelNormal)
			executor := workflow.NewClaudeExecutor(logger)

			ctx := context.Background()
			result, err := executor.Execute(ctx, workflow.ExecuteConfig{
				Prompt:     tt.prompt,
				JSONSchema: tt.schema,
				Timeout:    tt.timeout,
			})

			require.NoError(t, err, "claude execution with schema should succeed")
			require.NotNil(t, result)
			assert.Equal(t, 0, result.ExitCode, "exit code should be 0")
			assert.NotEmpty(t, result.Output, "output should not be empty")

			// Parse the Claude CLI JSON envelope first
			var envelope map[string]interface{}
			err = json.Unmarshal([]byte(result.Output), &envelope)
			require.NoError(t, err, "output should be valid JSON")

			// Extract structured_output from the envelope
			structuredOutput, ok := envelope["structured_output"]
			require.True(t, ok, "output should contain structured_output field")
			require.NotNil(t, structuredOutput, "structured_output should not be nil")

			// Convert structured_output to map for validation
			var data map[string]interface{}
			structuredBytes, err := json.Marshal(structuredOutput)
			require.NoError(t, err, "structured_output should be marshallable")
			err = json.Unmarshal(structuredBytes, &data)
			require.NoError(t, err, "structured_output should be valid JSON")

			// Check for required fields
			for _, field := range tt.wantFields {
				assert.Contains(t, data, field, "output should contain field %s", field)
			}

			// Run custom validation if provided
			if tt.validateResult != nil {
				tt.validateResult(t, data)
			}
		})
	}
}
