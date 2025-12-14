//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/michael-freling/claude-code-tools/internal/workflow"
	"github.com/michael-freling/claude-code-tools/test/e2e/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor_ArgumentConstruction(t *testing.T) {
	tests := []struct {
		name         string
		config       workflow.ExecuteConfig
		wantContains []string
		wantNotContains []string
		wantPromptLast bool
	}{
		{
			name: "basic execution always includes --print",
			config: workflow.ExecuteConfig{
				Prompt: "test prompt",
			},
			wantContains: []string{"--print"},
			wantPromptLast: true,
		},
		{
			name: "streaming mode includes stream-json and verbose",
			config: workflow.ExecuteConfig{
				Prompt: "test prompt",
			},
			wantContains: []string{
				"--print",
				"--output-format",
				"stream-json",
				"--verbose",
			},
			wantPromptLast: true,
		},
		{
			name: "dangerously skip permissions flag",
			config: workflow.ExecuteConfig{
				Prompt: "test prompt",
				DangerouslySkipPermissions: true,
			},
			wantContains: []string{
				"--print",
				"--dangerously-skip-permissions",
			},
			wantPromptLast: true,
		},
		{
			name: "json schema flag",
			config: workflow.ExecuteConfig{
				Prompt: "test prompt",
				JSONSchema: `{"type": "object"}`,
			},
			wantContains: []string{
				"--print",
				"--json-schema",
			},
			wantPromptLast: true,
		},
		{
			name: "all flags together",
			config: workflow.ExecuteConfig{
				Prompt: "test prompt",
				DangerouslySkipPermissions: true,
				JSONSchema: `{"type": "object"}`,
			},
			wantContains: []string{
				"--print",
				"--output-format",
				"stream-json",
				"--verbose",
				"--dangerously-skip-permissions",
				"--json-schema",
			},
			wantPromptLast: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := helpers.NewMockClaudeBuilder(t).
				WithStreamingResponse("test result", map[string]string{"summary": "test"})
			claudePath := mock.Build()

			logger := workflow.NewLogger(workflow.LogLevelNormal)
			executor := workflow.NewClaudeExecutorWithPath(claudePath, logger)

			ctx := context.Background()
			_, err := executor.ExecuteStreaming(ctx, tt.config, nil)
			require.NoError(t, err)

			args := mock.GetCapturedArgs()
			require.NotEmpty(t, args, "expected arguments to be captured")

			for _, want := range tt.wantContains {
				assert.Contains(t, args, want, "expected args to contain %q", want)
			}

			for _, notWant := range tt.wantNotContains {
				assert.NotContains(t, args, notWant, "expected args to not contain %q", notWant)
			}

			if tt.wantPromptLast {
				require.NotEmpty(t, args)
				lastArg := args[len(args)-1]
				assert.Equal(t, tt.config.Prompt, lastArg, "prompt should be the last argument")
			}
		})
	}
}

func TestPromptGeneration_PlanningPhase(t *testing.T) {
	tests := []struct {
		name             string
		workflowType     workflow.WorkflowType
		description      string
		feedback         []string
		wantContainsDesc bool
		wantContainsType bool
	}{
		{
			name:             "feature workflow",
			workflowType:     workflow.WorkflowTypeFeature,
			description:      "Add user authentication",
			wantContainsDesc: true,
			wantContainsType: true,
		},
		{
			name:             "fix workflow",
			workflowType:     workflow.WorkflowTypeFix,
			description:      "Fix memory leak in parser",
			wantContainsDesc: true,
			wantContainsType: true,
		},
		{
			name:             "with feedback",
			workflowType:     workflow.WorkflowTypeFeature,
			description:      "Add logging",
			feedback:         []string{"Please use structured logging", "Add tests"},
			wantContainsDesc: true,
			wantContainsType: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := workflow.NewPromptGenerator()
			require.NoError(t, err)

			prompt, err := gen.GeneratePlanningPrompt(tt.workflowType, tt.description, tt.feedback)
			require.NoError(t, err)
			assert.NotEmpty(t, prompt)

			if tt.wantContainsDesc {
				assert.Contains(t, prompt, tt.description, "prompt should contain description")
			}

			if tt.wantContainsType {
				assert.Contains(t, prompt, string(tt.workflowType), "prompt should contain workflow type")
			}

			for _, feedback := range tt.feedback {
				assert.Contains(t, prompt, feedback, "prompt should contain feedback: %s", feedback)
			}
		})
	}
}

func TestWorkflow_PlanningPhase_WithMockClaude(t *testing.T) {
	tests := []struct {
		name            string
		workflowType    workflow.WorkflowType
		description     string
		planJSON        map[string]interface{}
		wantError       bool
		wantPhaseParsed bool
	}{
		{
			name:         "valid plan",
			workflowType: workflow.WorkflowTypeFeature,
			description:  "Add user authentication",
			planJSON: map[string]interface{}{
				"summary":     "Implement user authentication with JWT",
				"contextType": "feature",
				"architecture": map[string]interface{}{
					"overview":   "Add JWT-based authentication",
					"components": []string{"auth middleware", "user model"},
				},
				"phases": []map[string]interface{}{
					{
						"name":           "Setup",
						"description":    "Setup authentication infrastructure",
						"estimatedFiles": 3,
						"estimatedLines": 150,
					},
				},
				"workStreams": []map[string]interface{}{
					{
						"name":  "Core auth",
						"tasks": []string{"JWT generation", "Token validation"},
					},
				},
				"risks":                []string{"Token expiration handling"},
				"complexity":           "medium",
				"estimatedTotalLines":  150,
				"estimatedTotalFiles":  3,
			},
			wantPhaseParsed: true,
		},
		{
			name:         "minimal valid plan",
			workflowType: workflow.WorkflowTypeFix,
			description:  "Fix memory leak",
			planJSON: map[string]interface{}{
				"summary":     "Fix memory leak in parser",
				"contextType": "fix",
				"phases": []map[string]interface{}{
					{
						"name":           "Fix",
						"description":    "Fix the leak",
						"estimatedFiles": 1,
						"estimatedLines": 10,
					},
				},
				"complexity": "small",
			},
			wantPhaseParsed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			mock := helpers.NewMockClaudeBuilder(t).
				WithStreamingResponse("Plan created", tt.planJSON)
			claudePath := mock.Build()

			config := workflow.DefaultConfig(tmpDir)
			config.ClaudePath = claudePath
			config.Timeouts.Planning = 10 * time.Second

			orchestrator, err := workflow.NewOrchestratorWithConfig(config)
			require.NoError(t, err)

			orchestrator.SetConfirmFunc(func(plan *workflow.Plan) (bool, string, error) {
				return false, "", workflow.ErrUserCancelled
			})

			ctx := context.Background()
			workflowName := "test-workflow"
			err = orchestrator.Start(ctx, workflowName, tt.description, tt.workflowType)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			if tt.wantPhaseParsed {
				stateManager := workflow.NewStateManager(tmpDir)
				plan, err := stateManager.LoadPlan(workflowName)
				require.NoError(t, err)
				assert.Equal(t, tt.planJSON["summary"], plan.Summary)
				assert.NotEmpty(t, plan.Phases)
			}

			args := mock.GetCapturedArgs()
			assert.Contains(t, args, "--print", "should include --print flag")
			assert.Contains(t, args, "stream-json", "should include stream-json format")
			assert.Contains(t, args, "--json-schema", "should include --json-schema flag")

			// Verify prompt was provided as last argument
			require.NotEmpty(t, args, "expected arguments to be captured")
			lastArg := args[len(args)-1]
			assert.NotEmpty(t, lastArg, "prompt should not be empty")
		})
	}
}

func TestWorkflow_PromptTooLong_Error(t *testing.T) {
	tests := []struct {
		name          string
		stderrMsg     string
		exitCode      int
		wantErrType   error
	}{
		{
			name:        "prompt too long error detected",
			stderrMsg:   "Prompt is too long",
			exitCode:    1,
			wantErrType: workflow.ErrPromptTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := helpers.NewMockClaudeBuilder(t).
				WithExitCode(tt.exitCode).
				WithStderr(tt.stderrMsg)
			claudePath := mock.Build()

			logger := workflow.NewLogger(workflow.LogLevelNormal)
			executor := workflow.NewClaudeExecutorWithPath(claudePath, logger)

			ctx := context.Background()
			_, err := executor.ExecuteStreaming(ctx, workflow.ExecuteConfig{
				Prompt: "test prompt",
			}, nil)

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErrType)
		})
	}
}

func TestExecutor_StreamingOutput_ParsesCorrectly(t *testing.T) {
	tests := []struct {
		name               string
		structuredOutput   interface{}
		streamingResult    string
		wantJSONValid      bool
		wantStructuredData bool
	}{
		{
			name: "structured output with plan schema",
			structuredOutput: map[string]interface{}{
				"summary":     "Test plan",
				"contextType": "feature",
				"phases": []map[string]interface{}{
					{
						"name":           "Test",
						"description":    "Test phase",
						"estimatedFiles": 1,
						"estimatedLines": 10,
					},
				},
				"complexity": "small",
			},
			streamingResult:    "Plan created successfully",
			wantJSONValid:      true,
			wantStructuredData: true,
		},
		{
			name: "implementation summary",
			structuredOutput: map[string]interface{}{
				"filesChanged": []string{"main.go", "test.go"},
				"linesAdded":   100,
				"linesRemoved": 50,
				"testsAdded":   5,
				"summary":      "Implemented feature X",
			},
			streamingResult:    "Implementation complete",
			wantJSONValid:      true,
			wantStructuredData: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := helpers.NewMockClaudeBuilder(t).
				WithStreamingResponse(tt.streamingResult, tt.structuredOutput)
			claudePath := mock.Build()

			logger := workflow.NewLogger(workflow.LogLevelNormal)
			executor := workflow.NewClaudeExecutorWithPath(claudePath, logger)

			ctx := context.Background()
			result, err := executor.ExecuteStreaming(ctx, workflow.ExecuteConfig{
				Prompt:     "test",
				JSONSchema: workflow.PlanSchema,
			}, nil)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.Output)

			if tt.wantJSONValid {
				var envelope map[string]interface{}
				err := json.Unmarshal([]byte(result.Output), &envelope)
				require.NoError(t, err, "output should be valid JSON")

				if tt.wantStructuredData {
					assert.Contains(t, envelope, "structured_output")
					assert.Contains(t, envelope, "result")
				}
			}

			parser := workflow.NewOutputParser()
			extractedJSON, err := parser.ExtractJSON(result.Output)
			require.NoError(t, err)
			assert.NotEmpty(t, extractedJSON)

			var parsed map[string]interface{}
			err = json.Unmarshal([]byte(extractedJSON), &parsed)
			require.NoError(t, err, "extracted JSON should be valid")
		})
	}
}

func TestExecutor_WorkingDirectory(t *testing.T) {
	tests := []struct {
		name    string
		setupWD func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "valid working directory",
			setupWD: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return tmpDir
			},
			wantErr: false,
		},
		{
			name: "empty working directory uses current",
			setupWD: func(t *testing.T) string {
				return ""
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workingDir := tt.setupWD(t)

			if workingDir != "" {
				testFile := filepath.Join(workingDir, "test.txt")
				err := os.WriteFile(testFile, []byte("test"), 0644)
				require.NoError(t, err)
			}

			mock := helpers.NewMockClaudeBuilder(t).
				WithStreamingResponse("success", map[string]string{"result": "ok"})
			claudePath := mock.Build()

			logger := workflow.NewLogger(workflow.LogLevelNormal)
			executor := workflow.NewClaudeExecutorWithPath(claudePath, logger)

			ctx := context.Background()
			result, err := executor.ExecuteStreaming(ctx, workflow.ExecuteConfig{
				Prompt:           "test",
				WorkingDirectory: workingDir,
			}, nil)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestExecutor_Timeout(t *testing.T) {
	tests := []struct {
		name       string
		timeout    time.Duration
		mockDelay  bool
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:    "no timeout completes successfully",
			timeout: 0,
			wantErr: false,
		},
		{
			name:    "sufficient timeout completes successfully",
			timeout: 5 * time.Second,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := helpers.NewMockClaudeBuilder(t).
				WithStreamingResponse("done", map[string]string{"status": "ok"})
			claudePath := mock.Build()

			logger := workflow.NewLogger(workflow.LogLevelNormal)
			executor := workflow.NewClaudeExecutorWithPath(claudePath, logger)

			ctx := context.Background()
			result, err := executor.ExecuteStreaming(ctx, workflow.ExecuteConfig{
				Prompt:  "test",
				Timeout: tt.timeout,
			}, nil)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}
