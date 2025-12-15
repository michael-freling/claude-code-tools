package workflow

import (
	"strings"
	"testing"
	"text/template"

	"github.com/michael-freling/claude-code-tools/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPromptGenerator(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successfully creates generator with embedded templates",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPromptGenerator()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestPromptGenerator_GeneratePlanningPrompt(t *testing.T) {
	tests := []struct {
		name        string
		wfType      WorkflowType
		description string
		feedback    []string
		wantErr     bool
		wantContain []string
	}{
		{
			name:        "generates planning prompt for feature without feedback",
			wfType:      WorkflowTypeFeature,
			description: "add JWT authentication",
			feedback:    nil,
			wantErr:     false,
			wantContain: []string{
				"Type: feature",
				"Description: add JWT authentication",
				"Output your plan as JSON",
			},
		},
		{
			name:        "generates planning prompt for fix without feedback",
			wfType:      WorkflowTypeFix,
			description: "fix login timeout",
			feedback:    nil,
			wantErr:     false,
			wantContain: []string{
				"Type: fix",
				"Description: fix login timeout",
				"Output your plan as JSON",
			},
		},
		{
			name:        "generates planning prompt with feedback",
			wfType:      WorkflowTypeFeature,
			description: "add user authentication",
			feedback:    []string{"use refresh tokens", "add logout endpoint"},
			wantErr:     false,
			wantContain: []string{
				"Type: feature",
				"Description: add user authentication",
				"Previous Feedback",
				"use refresh tokens",
				"add logout endpoint",
			},
		},
		{
			name:        "generates planning prompt with empty feedback slice",
			wfType:      WorkflowTypeFeature,
			description: "add feature",
			feedback:    []string{},
			wantErr:     false,
			wantContain: []string{
				"Type: feature",
				"Description: add feature",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewPromptGenerator()
			require.NoError(t, err)

			got, err := pg.GeneratePlanningPrompt(tt.wfType, tt.description, tt.feedback)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, got)

			for _, want := range tt.wantContain {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestPromptGenerator_GenerateImplementationPrompt(t *testing.T) {
	tests := []struct {
		name        string
		plan        *Plan
		wantErr     bool
		errContains string
		wantContain []string
	}{
		{
			name: "generates implementation prompt with complete plan",
			plan: &Plan{
				Summary:     "Add JWT authentication",
				ContextType: "feature",
				Architecture: Architecture{
					Overview:   "Implement JWT-based auth",
					Components: []string{"auth service", "middleware"},
				},
				Phases: []PlanPhase{
					{
						Name:           "Database Schema",
						Description:    "Add users table",
						EstimatedFiles: 3,
						EstimatedLines: 50,
					},
				},
				WorkStreams: []WorkStream{
					{
						Name:  "Backend",
						Tasks: []string{"DB schema", "Auth service"},
					},
				},
				Risks:      []string{"Token expiration handling"},
				Complexity: "medium",
			},
			wantErr: false,
			wantContain: []string{
				"Add JWT authentication",
				"Implement JWT-based auth",
				"auth service",
				"middleware",
				"Database Schema",
				"Add users table",
				"Backend",
				"DB schema",
				"Token expiration handling",
				"Output Format",
			},
		},
		{
			name:        "returns error when plan is nil",
			plan:        nil,
			wantErr:     true,
			errContains: "plan cannot be nil",
		},
		{
			name: "generates prompt with minimal plan",
			plan: &Plan{
				Summary:     "Simple fix",
				ContextType: "fix",
				Architecture: Architecture{
					Overview:   "Fix bug",
					Components: []string{},
				},
				Phases:      []PlanPhase{},
				WorkStreams: []WorkStream{},
				Risks:       []string{},
				Complexity:  "small",
			},
			wantErr: false,
			wantContain: []string{
				"Simple fix",
				"Fix bug",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewPromptGenerator()
			require.NoError(t, err)

			got, err := pg.GenerateImplementationPrompt(tt.plan)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, got)

			for _, want := range tt.wantContain {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestPromptGenerator_GenerateRefactoringPrompt(t *testing.T) {
	tests := []struct {
		name        string
		plan        *Plan
		wantErr     bool
		errContains string
		wantContain []string
	}{
		{
			name: "generates refactoring prompt with complete plan",
			plan: &Plan{
				Summary:     "Add JWT authentication",
				ContextType: "feature",
				Architecture: Architecture{
					Overview:   "Implement JWT-based auth",
					Components: []string{"auth service", "middleware"},
				},
			},
			wantErr: false,
			wantContain: []string{
				"Add JWT authentication",
				"Implement JWT-based auth",
				"auth service",
				"middleware",
				"Refactor to improve",
				"Output Format",
			},
		},
		{
			name:        "returns error when plan is nil",
			plan:        nil,
			wantErr:     true,
			errContains: "plan cannot be nil",
		},
		{
			name: "generates prompt with minimal plan",
			plan: &Plan{
				Summary:     "Simple feature",
				ContextType: "feature",
				Architecture: Architecture{
					Overview:   "Add feature",
					Components: []string{},
				},
			},
			wantErr: false,
			wantContain: []string{
				"Simple feature",
				"Add feature",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewPromptGenerator()
			require.NoError(t, err)

			got, err := pg.GenerateRefactoringPrompt(tt.plan)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, got)

			for _, want := range tt.wantContain {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestPromptGenerator_GeneratePRSplitPrompt(t *testing.T) {
	tests := []struct {
		name        string
		metrics     *PRMetrics
		commits     []command.Commit
		wantErr     bool
		errContains string
		wantContain []string
	}{
		{
			name: "generates PR split prompt with complete metrics and commits",
			metrics: &PRMetrics{
				LinesChanged:  423,
				FilesChanged:  12,
				FilesAdded:    []string{"auth.go", "middleware.go"},
				FilesModified: []string{"main.go", "routes.go"},
				FilesDeleted:  []string{"old_auth.go"},
			},
			commits: []command.Commit{
				{Hash: "abc123", Subject: "Add authentication"},
				{Hash: "def456", Subject: "Add middleware"},
			},
			wantErr: false,
			wantContain: []string{
				"Lines Changed: 423",
				"Files Changed: 12",
				"auth.go",
				"middleware.go",
				"main.go",
				"routes.go",
				"old_auth.go",
				"abc123",
				"Add authentication",
				"def456",
				"Add middleware",
				"Required JSON Response Format",
				"CRITICAL REQUIREMENTS",
				"NEVER return an empty childPRs array",
				"childPRs array MUST contain at least ONE child PR",
				"childPRs array MUST have at least 1 element",
			},
		},
		{
			name:        "returns error when metrics is nil",
			metrics:     nil,
			commits:     []command.Commit{},
			wantErr:     true,
			errContains: "metrics cannot be nil",
		},
		{
			name: "generates prompt with minimal metrics and no commits",
			metrics: &PRMetrics{
				LinesChanged: 50,
				FilesChanged: 3,
			},
			commits: []command.Commit{},
			wantErr: false,
			wantContain: []string{
				"Lines Changed: 50",
				"Files Changed: 3",
			},
		},
		{
			name: "generates prompt with only added files and commits",
			metrics: &PRMetrics{
				LinesChanged: 100,
				FilesChanged: 5,
				FilesAdded:   []string{"file1.go", "file2.go"},
			},
			commits: []command.Commit{
				{Hash: "ghi789", Subject: "Add new files"},
			},
			wantErr: false,
			wantContain: []string{
				"Lines Changed: 100",
				"Files Changed: 5",
				"Files Added",
				"file1.go",
				"file2.go",
				"ghi789",
				"Add new files",
			},
		},
		{
			name: "prompt includes fallback guidance for small changes",
			metrics: &PRMetrics{
				LinesChanged: 50,
				FilesChanged: 2,
			},
			commits: []command.Commit{
				{Hash: "abc123", Subject: "Small fix"},
			},
			wantErr: false,
			wantContain: []string{
				"If the changes are too small to logically split into multiple PRs",
				"Create a SINGLE child PR containing all the changes",
				"[Part 1/1]",
				"This is the correct approach rather than returning empty childPRs",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewPromptGenerator()
			require.NoError(t, err)

			got, err := pg.GeneratePRSplitPrompt(tt.metrics, tt.commits)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, got)

			for _, want := range tt.wantContain {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestPromptGenerator_TemplateLoadingErrors(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successfully loads all templates",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewPromptGenerator()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, pg)

			generator := pg.(*promptGenerator)
			assert.NotNil(t, generator.templates["planning.tmpl"])
			assert.NotNil(t, generator.templates["implementation.tmpl"])
			assert.NotNil(t, generator.templates["refactoring.tmpl"])
			assert.NotNil(t, generator.templates["pr-split.tmpl"])
		})
	}
}

func TestPromptGenerator_AllMethodsReturnValidPrompts(t *testing.T) {
	pg, err := NewPromptGenerator()
	require.NoError(t, err)

	testPlan := &Plan{
		Summary:     "Test plan",
		ContextType: "feature",
		Architecture: Architecture{
			Overview:   "Test architecture",
			Components: []string{"component1"},
		},
		Phases: []PlanPhase{
			{
				Name:           "Phase 1",
				Description:    "Test phase",
				EstimatedFiles: 1,
				EstimatedLines: 10,
			},
		},
		WorkStreams: []WorkStream{
			{
				Name:  "Stream 1",
				Tasks: []string{"task1"},
			},
		},
		Risks:      []string{"risk1"},
		Complexity: "small",
	}

	testMetrics := &PRMetrics{
		LinesChanged: 100,
		FilesChanged: 5,
	}

	tests := []struct {
		name    string
		genFunc func() (string, error)
	}{
		{
			name: "planning prompt",
			genFunc: func() (string, error) {
				return pg.GeneratePlanningPrompt(WorkflowTypeFeature, "test", nil)
			},
		},
		{
			name: "implementation prompt",
			genFunc: func() (string, error) {
				return pg.GenerateImplementationPrompt(testPlan)
			},
		},
		{
			name: "refactoring prompt",
			genFunc: func() (string, error) {
				return pg.GenerateRefactoringPrompt(testPlan)
			},
		},
		{
			name: "PR split prompt",
			genFunc: func() (string, error) {
				return pg.GeneratePRSplitPrompt(testMetrics, []command.Commit{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := tt.genFunc()
			require.NoError(t, err)
			assert.NotEmpty(t, prompt)
			assert.Greater(t, len(strings.TrimSpace(prompt)), 10)
		})
	}
}

func TestPromptGenerator_GenerateFixCIPrompt(t *testing.T) {
	tests := []struct {
		name        string
		failures    string
		wantErr     bool
		errContains string
		wantContain []string
	}{
		{
			name:     "valid failures string returns prompt containing failures",
			failures: "Test failed: TestFoo\nExpected: 1\nGot: 2",
			wantErr:  false,
			wantContain: []string{
				"Test failed: TestFoo",
				"Expected: 1",
				"Got: 2",
			},
		},
		{
			name:        "empty failures string returns error",
			failures:    "",
			wantErr:     true,
			errContains: "failures cannot be empty",
		},
		{
			name:     "failures with special characters works correctly",
			failures: "Error: syntax error near '&&' in file test_*.go",
			wantErr:  false,
			wantContain: []string{
				"Error: syntax error near '&&' in file test_*.go",
			},
		},
		{
			name: "failures with newlines preserves newlines in output",
			failures: `line 1
line 2
line 3`,
			wantErr: false,
			wantContain: []string{
				"line 1",
				"line 2",
				"line 3",
			},
		},
		{
			name: "multi-line failure output all lines present in prompt",
			failures: `=== RUN   TestExample
--- FAIL: TestExample (0.00s)
    example_test.go:10: Expected true, got false
FAIL
exit status 1`,
			wantErr: false,
			wantContain: []string{
				"=== RUN   TestExample",
				"--- FAIL: TestExample (0.00s)",
				"example_test.go:10: Expected true, got false",
				"FAIL",
				"exit status 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewPromptGenerator()
			require.NoError(t, err)

			got, err := pg.GenerateFixCIPrompt(tt.failures)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, got)

			for _, want := range tt.wantContain {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestPromptGenerator_TemplateNotLoadedError(t *testing.T) {
	tests := []struct {
		name        string
		genFunc     func(*promptGenerator) (string, error)
		errContains string
	}{
		{
			name: "fix-ci template not loaded returns error",
			genFunc: func(pg *promptGenerator) (string, error) {
				return pg.GenerateFixCIPrompt("test failure")
			},
			errContains: "fix-ci template not loaded",
		},
		{
			name: "planning template not loaded returns error",
			genFunc: func(pg *promptGenerator) (string, error) {
				return pg.GeneratePlanningPrompt(WorkflowTypeFeature, "test", nil)
			},
			errContains: "planning template not loaded",
		},
		{
			name: "implementation template not loaded returns error",
			genFunc: func(pg *promptGenerator) (string, error) {
				return pg.GenerateImplementationPrompt(&Plan{Summary: "test"})
			},
			errContains: "implementation template not loaded",
		},
		{
			name: "refactoring template not loaded returns error",
			genFunc: func(pg *promptGenerator) (string, error) {
				return pg.GenerateRefactoringPrompt(&Plan{Summary: "test"})
			},
			errContains: "refactoring template not loaded",
		},
		{
			name: "pr-split template not loaded returns error",
			genFunc: func(pg *promptGenerator) (string, error) {
				return pg.GeneratePRSplitPrompt(&PRMetrics{LinesChanged: 100, FilesChanged: 5}, []command.Commit{})
			},
			errContains: "pr-split template not loaded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := &promptGenerator{
				templates: make(map[string]*template.Template),
			}

			got, err := tt.genFunc(pg)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
			assert.Empty(t, got)
		})
	}
}

func TestPromptGenerator_TemplateExecutionError(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		templateText string
		genFunc      func(*promptGenerator) (string, error)
		errContains  string
	}{
		{
			name:         "planning template execution error with invalid field reference",
			templateName: "planning.tmpl",
			templateText: "{{.NonExistentField}}",
			genFunc: func(pg *promptGenerator) (string, error) {
				return pg.GeneratePlanningPrompt(WorkflowTypeFeature, "test", nil)
			},
			errContains: "failed to execute planning template",
		},
		{
			name:         "implementation template execution error with invalid field reference",
			templateName: "implementation.tmpl",
			templateText: "{{.InvalidField}}",
			genFunc: func(pg *promptGenerator) (string, error) {
				return pg.GenerateImplementationPrompt(&Plan{Summary: "test"})
			},
			errContains: "failed to execute implementation template",
		},
		{
			name:         "refactoring template execution error with invalid field reference",
			templateName: "refactoring.tmpl",
			templateText: "{{.BadField}}",
			genFunc: func(pg *promptGenerator) (string, error) {
				return pg.GenerateRefactoringPrompt(&Plan{Summary: "test"})
			},
			errContains: "failed to execute refactoring template",
		},
		{
			name:         "pr-split template execution error with invalid field reference",
			templateName: "pr-split.tmpl",
			templateText: "{{.WrongField}}",
			genFunc: func(pg *promptGenerator) (string, error) {
				return pg.GeneratePRSplitPrompt(&PRMetrics{LinesChanged: 100, FilesChanged: 5}, []command.Commit{})
			},
			errContains: "failed to execute pr-split template",
		},
		{
			name:         "fix-ci template execution error with invalid function call",
			templateName: "fix-ci.tmpl",
			templateText: "{{.NonExistentMethod}}",
			genFunc: func(pg *promptGenerator) (string, error) {
				return pg.GenerateFixCIPrompt("test failure")
			},
			errContains: "failed to execute fix-ci template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := &promptGenerator{
				templates: make(map[string]*template.Template),
			}

			tmpl, err := template.New(tt.templateName).Parse(tt.templateText)
			require.NoError(t, err)
			pg.templates[tt.templateName] = tmpl

			got, err := tt.genFunc(pg)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
			assert.Empty(t, got)
		})
	}
}

func TestPromptGenerator_GenerateFixCIPrompt_TemplateContent(t *testing.T) {
	tests := []struct {
		name        string
		failures    string
		wantContain []string
	}{
		{
			name:     "prompt includes CI failure header",
			failures: "test error",
			wantContain: []string{
				"CI checks have failed",
				"CI Failure Output",
			},
		},
		{
			name:     "prompt includes instructions section",
			failures: "build failed",
			wantContain: []string{
				"Instructions",
				"Analyze the CI failure output",
				"For FAILED jobs: Fix all issues reported by the CI system",
				"DO NOT skip or ignore actual errors",
			},
		},
		{
			name:     "prompt includes output format section",
			failures: "lint error",
			wantContain: []string{
				"Output Format",
				"filesChanged",
				"linesAdded",
				"linesRemoved",
				"summary",
				"nextSteps",
			},
		},
		{
			name:     "prompt includes common CI failures list",
			failures: "test failure",
			wantContain: []string{
				"Common CI failures",
				"Unit tests failing",
				"Integration tests failing",
				"Build failures",
				"Linting issues",
				"Race conditions",
			},
		},
		{
			name:     "prompt includes important reminders",
			failures: "error occurred",
			wantContain: []string{
				"IMPORTANT",
				"Fix ALL actual CI failures (failed jobs)",
				"Ensure all tests pass locally",
				"Do not skip or disable failing tests",
				"Address root causes, not symptoms",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewPromptGenerator()
			require.NoError(t, err)

			got, err := pg.GenerateFixCIPrompt(tt.failures)

			require.NoError(t, err)
			assert.NotEmpty(t, got)

			for _, want := range tt.wantContain {
				assert.Contains(t, got, want, "prompt should contain: %s", want)
			}

			assert.Contains(t, got, tt.failures, "prompt should contain the failure message")
		})
	}
}

func TestPromptGenerator_GenerateFixCIPrompt_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		failures    string
		wantErr     bool
		errContains string
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:        "whitespace-only failures returns error",
			failures:    "   \t\n   ",
			wantErr:     true,
			errContains: "failures cannot be empty",
		},
		{
			name:     "single character failure works",
			failures: "X",
			wantErr:  false,
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "X")
				assert.Contains(t, output, "CI Failure Output")
			},
		},
		{
			name:     "very long failure message is preserved",
			failures: strings.Repeat("Error: test failed with long message. ", 100),
			wantErr:  false,
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "Error: test failed with long message.")
				assert.Greater(t, len(output), 1000)
			},
		},
		{
			name:     "failures with tabs and mixed whitespace preserved",
			failures: "Error:\tfailed\n\t\tat line 42",
			wantErr:  false,
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "Error:\tfailed")
				assert.Contains(t, output, "\t\tat line 42")
			},
		},
		{
			name:     "failures with unicode characters work correctly",
			failures: "Error: テスト失敗 ❌ Failed test 🚨",
			wantErr:  false,
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "テスト失敗")
				assert.Contains(t, output, "❌")
				assert.Contains(t, output, "🚨")
			},
		},
		{
			name:     "failures with code snippets preserved correctly",
			failures: "Error in code:\n```go\nfunc test() {\n\treturn nil\n}\n```",
			wantErr:  false,
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "```go")
				assert.Contains(t, output, "func test() {")
				assert.Contains(t, output, "\treturn nil")
			},
		},
		{
			name:     "failures with JSON data preserved",
			failures: `{"error": "test failed", "details": {"line": 42, "file": "test.go"}}`,
			wantErr:  false,
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, `"error": "test failed"`)
				assert.Contains(t, output, `"line": 42`)
				assert.Contains(t, output, `"file": "test.go"`)
			},
		},
		{
			name:     "failures with ANSI color codes work",
			failures: "\033[31mError:\033[0m test failed\n\033[33mWarning:\033[0m check this",
			wantErr:  false,
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "\033[31mError:\033[0m")
				assert.Contains(t, output, "\033[33mWarning:\033[0m")
			},
		},
		{
			name:     "failures with backticks and quotes preserved",
			failures: `Error: can't parse 'value' with "quotes" and ` + "`backticks`",
			wantErr:  false,
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "can't parse")
				assert.Contains(t, output, "'value'")
				assert.Contains(t, output, `"quotes"`)
				assert.Contains(t, output, "`backticks`")
			},
		},
		{
			name:     "failures with URLs preserved",
			failures: "Error: failed to fetch https://api.example.com/v1/resource?key=value&other=param",
			wantErr:  false,
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "https://api.example.com/v1/resource?key=value&other=param")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewPromptGenerator()
			require.NoError(t, err)

			got, err := pg.GenerateFixCIPrompt(tt.failures)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, got)

			if tt.checkOutput != nil {
				tt.checkOutput(t, got)
			}
		})
	}
}

func TestPromptGenerator_LoadTemplates_ErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "loadTemplates succeeds with valid embedded templates",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := &promptGenerator{
				templates: make(map[string]*template.Template),
			}

			err := pg.loadTemplates()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, pg.templates, 10)
			assert.NotNil(t, pg.templates["planning.tmpl"])
			assert.NotNil(t, pg.templates["implementation.tmpl"])
			assert.NotNil(t, pg.templates["refactoring.tmpl"])
			assert.NotNil(t, pg.templates["pr-split.tmpl"])
			assert.NotNil(t, pg.templates["fix-ci.tmpl"])
			assert.NotNil(t, pg.templates["create-pr.tmpl"])
			assert.NotNil(t, pg.templates["planning-simplified.tmpl"])
			assert.NotNil(t, pg.templates["implementation-simplified.tmpl"])
			assert.NotNil(t, pg.templates["refactoring-simplified.tmpl"])
			assert.NotNil(t, pg.templates["pr-split-simplified.tmpl"])
		})
	}
}

func TestPromptGenerator_GenerateFixCIPrompt_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name        string
		failures    string
		wantContain []string
	}{
		{
			name: "go test failure with stack trace",
			failures: `=== RUN   TestCalculate
--- FAIL: TestCalculate (0.00s)
    calculator_test.go:15:
        	Error Trace:	calculator_test.go:15
        	Error:      	Not equal:
        	            	expected: 4
        	            	actual  : 5
        	Test:       	TestCalculate
FAIL
FAIL	github.com/example/calculator	0.002s`,
			wantContain: []string{
				"=== RUN   TestCalculate",
				"--- FAIL: TestCalculate",
				"calculator_test.go:15",
				"expected: 4",
				"actual  : 5",
				"github.com/example/calculator",
			},
		},
		{
			name: "go build failure with compiler error",
			failures: `# github.com/example/app
./main.go:42:2: undefined: nonExistentFunction
./main.go:43:15: cannot use "string" (untyped string constant) as int value in assignment
./main.go:44:9: syntax error: unexpected newline, expecting comma or }`,
			wantContain: []string{
				"github.com/example/app",
				"main.go:42:2: undefined: nonExistentFunction",
				"main.go:43:15: cannot use",
				"syntax error",
			},
		},
		{
			name: "golangci-lint failure",
			failures: `main.go:10:2: SA4006: this value of err is never used (staticcheck)
	err := doSomething()
	^
handlers.go:25:1: ST1003: should not use underscores in Go names; func get_user should be getUser (stylecheck)
service.go:15:15: Error return value is not checked (errcheck)`,
			wantContain: []string{
				"SA4006",
				"this value of err is never used",
				"ST1003",
				"should not use underscores",
				"Error return value is not checked",
			},
		},
		{
			name: "race detector failure",
			failures: `==================
WARNING: DATA RACE
Read at 0x00c0000b6010 by goroutine 8:
  main.updateCounter()
      /path/to/main.go:45 +0x3a

Previous write at 0x00c0000b6010 by goroutine 7:
  main.updateCounter()
      /path/to/main.go:45 +0x52

Goroutine 8 (running) created at:
  main.main()
      /path/to/main.go:30 +0x7e
==================`,
			wantContain: []string{
				"WARNING: DATA RACE",
				"Read at 0x00c0000b6010 by goroutine 8",
				"Previous write at 0x00c0000b6010",
				"main.updateCounter()",
			},
		},
		{
			name: "coverage threshold failure",
			failures: `coverage: 45.2% of statements
FAIL	coverage threshold not met: got 45.2%, want >= 80.0%
exit status 1`,
			wantContain: []string{
				"coverage: 45.2% of statements",
				"coverage threshold not met",
				"got 45.2%, want >= 80.0%",
			},
		},
		{
			name: "docker build failure",
			failures: `Step 5/12 : RUN go build -o /app/server
 ---> Running in 8a9b7c6d5e4f
# github.com/example/app
./main.go:10:2: undefined: missingPackage
The command '/bin/sh -c go build -o /app/server' returned a non-zero code: 2`,
			wantContain: []string{
				"Step 5/12",
				"RUN go build -o /app/server",
				"undefined: missingPackage",
				"returned a non-zero code: 2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewPromptGenerator()
			require.NoError(t, err)

			got, err := pg.GenerateFixCIPrompt(tt.failures)

			require.NoError(t, err)
			assert.NotEmpty(t, got)

			for _, want := range tt.wantContain {
				assert.Contains(t, got, want, "prompt should contain: %s", want)
			}

			assert.Contains(t, got, "CI Failure Output")
			assert.Contains(t, got, "Output Format")
		})
	}
}

func TestPromptGenerator_GenerateCreatePRPrompt(t *testing.T) {
	tests := []struct {
		name        string
		ctx         *PRCreationContext
		wantErr     bool
		errContains string
		wantContain []string
	}{
		{
			name: "generates create PR prompt with valid context containing all fields",
			ctx: &PRCreationContext{
				WorkflowType: WorkflowTypeFeature,
				Branch:       "feature/add-authentication",
				BaseBranch:   "main",
				Description:  "Add JWT authentication",
			},
			wantErr: false,
			wantContain: []string{
				"Workflow Type: feature",
				"Branch: feature/add-authentication",
				"Base Branch: main",
				"Description: Add JWT authentication",
				"Decision Tree",
				"Step 1: Check for commits",
				"git log origin/main..HEAD --oneline",
				"Step 2: Check for existing PR",
				"gh pr list --head",
				"Step 3: Push branch to remote",
				"git push -u origin HEAD",
				"Step 4: Create the PR",
				"gh pr create",
				"Step 5: Verify PR creation",
				"Output Format",
				"prNumber",
				"status",
				"message",
				"created|exists|skipped|failed",
			},
		},
		{
			name: "generates create PR prompt for fix workflow",
			ctx: &PRCreationContext{
				WorkflowType: WorkflowTypeFix,
				Branch:       "fix/login-timeout",
				BaseBranch:   "main",
				Description:  "Fix login timeout issue",
			},
			wantErr: false,
			wantContain: []string{
				"Workflow Type: fix",
				"Branch: fix/login-timeout",
				"Base Branch: main",
				"Description: Fix login timeout issue",
				"For fix workflows: \"fix: <description>\"",
			},
		},
		{
			name: "generates create PR prompt with different base branch",
			ctx: &PRCreationContext{
				WorkflowType: WorkflowTypeFeature,
				Branch:       "feature/new-api",
				BaseBranch:   "develop",
				Description:  "Add new API endpoint",
			},
			wantErr: false,
			wantContain: []string{
				"Base Branch: develop",
				"git log origin/develop..HEAD --oneline",
			},
		},
		{
			name:        "returns error when context is nil",
			ctx:         nil,
			wantErr:     true,
			errContains: "context cannot be nil",
		},
		{
			name: "returns error when branch is empty",
			ctx: &PRCreationContext{
				WorkflowType: WorkflowTypeFeature,
				Branch:       "",
				BaseBranch:   "main",
				Description:  "test",
			},
			wantErr:     true,
			errContains: "branch cannot be empty",
		},
		{
			name: "returns error when base branch is empty",
			ctx: &PRCreationContext{
				WorkflowType: WorkflowTypeFeature,
				Branch:       "feature/test",
				BaseBranch:   "",
				Description:  "test",
			},
			wantErr:     true,
			errContains: "base branch cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewPromptGenerator()
			require.NoError(t, err)

			got, err := pg.GenerateCreatePRPrompt(tt.ctx)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, got)

			for _, want := range tt.wantContain {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestPromptGenerator_GenerateCreatePRPrompt_TemplateNotLoaded(t *testing.T) {
	tests := []struct {
		name        string
		ctx         *PRCreationContext
		errContains string
	}{
		{
			name: "create-pr template not loaded returns error",
			ctx: &PRCreationContext{
				WorkflowType: WorkflowTypeFeature,
				Branch:       "feature/test",
				BaseBranch:   "main",
				Description:  "test",
			},
			errContains: "create-pr template not loaded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := &promptGenerator{
				templates: make(map[string]*template.Template),
			}

			got, err := pg.GenerateCreatePRPrompt(tt.ctx)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
			assert.Empty(t, got)
		})
	}
}

func TestPromptGenerator_GenerateCreatePRPrompt_TemplateExecutionError(t *testing.T) {
	tests := []struct {
		name         string
		templateText string
		ctx          *PRCreationContext
		errContains  string
	}{
		{
			name:         "template execution error with invalid field reference",
			templateText: "{{.NonExistentField}}",
			ctx: &PRCreationContext{
				WorkflowType: WorkflowTypeFeature,
				Branch:       "test",
				BaseBranch:   "main",
				Description:  "test",
			},
			errContains: "failed to execute create-pr template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := &promptGenerator{
				templates: make(map[string]*template.Template),
			}

			tmpl, err := template.New("create-pr.tmpl").Parse(tt.templateText)
			require.NoError(t, err)
			pg.templates["create-pr.tmpl"] = tmpl

			got, err := pg.GenerateCreatePRPrompt(tt.ctx)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
			assert.Empty(t, got)
		})
	}
}

func TestPromptGenerator_GenerateSimplifiedPlanningPrompt(t *testing.T) {
	tests := []struct {
		name        string
		req         FeatureRequest
		attempt     int
		wantErr     bool
		wantContain []string
	}{
		{
			name: "generates simplified planning prompt for feature workflow",
			req: FeatureRequest{
				Type:        WorkflowTypeFeature,
				Description: "add authentication",
				Feedback:    nil,
			},
			attempt: 1,
			wantErr: false,
			wantContain: []string{
				"Type: feature",
				"Description: add authentication",
			},
		},
		{
			name: "generates simplified planning prompt for fix workflow",
			req: FeatureRequest{
				Type:        WorkflowTypeFix,
				Description: "fix login bug",
				Feedback:    nil,
			},
			attempt: 2,
			wantErr: false,
			wantContain: []string{
				"Type: fix",
				"Description: fix login bug",
			},
		},
		{
			name: "generates simplified planning prompt with feedback",
			req: FeatureRequest{
				Type:        WorkflowTypeFeature,
				Description: "add feature",
				Feedback:    []string{"use refresh tokens", "add logout"},
			},
			attempt: 1,
			wantErr: false,
			wantContain: []string{
				"Type: feature",
				"Description: add feature",
				"use refresh tokens",
				"add logout",
			},
		},
		{
			name: "generates simplified planning prompt on retry",
			req: FeatureRequest{
				Type:        WorkflowTypeFeature,
				Description: "test feature",
				Feedback:    nil,
			},
			attempt: 3,
			wantErr: false,
			wantContain: []string{
				"Type: feature",
				"Description: test feature",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewPromptGenerator()
			require.NoError(t, err)

			got, err := pg.GenerateSimplifiedPlanningPrompt(tt.req, tt.attempt)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, got)

			regularPrompt, err := pg.GeneratePlanningPrompt(tt.req.Type, tt.req.Description, tt.req.Feedback)
			require.NoError(t, err)
			assert.Less(t, len(got), len(regularPrompt), "simplified prompt should be smaller than regular prompt")

			for _, want := range tt.wantContain {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestPromptGenerator_GenerateSimplifiedImplementationPrompt(t *testing.T) {
	tests := []struct {
		name        string
		ctx         *WorkflowContext
		workStream  WorkStream
		attempt     int
		wantErr     bool
		wantContain []string
		checkTasks  func(t *testing.T, prompt string, originalTasks []string, attempt int)
	}{
		{
			name: "generates simplified implementation prompt with task truncation for attempt 2",
			ctx: &WorkflowContext{
				Plan: &Plan{
					Summary:     "Add authentication",
					ContextType: "feature",
					Architecture: Architecture{
						Overview:   "JWT auth",
						Components: []string{"auth service"},
					},
				},
			},
			workStream: WorkStream{
				Name: "Backend",
				Tasks: []string{
					"task 1",
					"task 2",
					"task 3",
					"task 4",
					"task 5",
					"task 6",
					"task 7",
				},
			},
			attempt: 2,
			wantErr: false,
			wantContain: []string{
				"Add authentication",
				"JWT auth",
			},
			checkTasks: func(t *testing.T, prompt string, originalTasks []string, attempt int) {
				assert.Contains(t, prompt, "task 3")
				assert.Contains(t, prompt, "task 4")
				assert.Contains(t, prompt, "task 5")
				assert.Contains(t, prompt, "task 6")
				assert.Contains(t, prompt, "task 7")
				assert.NotContains(t, prompt, "task 1")
				assert.NotContains(t, prompt, "task 2")
			},
		},
		{
			name: "generates simplified implementation prompt with task truncation for attempt 3+",
			ctx: &WorkflowContext{
				Plan: &Plan{
					Summary:     "Add feature",
					ContextType: "feature",
					Architecture: Architecture{
						Overview:   "Test",
						Components: []string{"component"},
					},
				},
			},
			workStream: WorkStream{
				Name: "Backend",
				Tasks: []string{
					"task 1",
					"task 2",
					"task 3",
					"task 4",
					"task 5",
				},
			},
			attempt: 3,
			wantErr: false,
			wantContain: []string{
				"Add feature",
			},
			checkTasks: func(t *testing.T, prompt string, originalTasks []string, attempt int) {
				assert.Contains(t, prompt, "task 3")
				assert.Contains(t, prompt, "task 4")
				assert.Contains(t, prompt, "task 5")
				assert.NotContains(t, prompt, "task 1")
				assert.NotContains(t, prompt, "task 2")
			},
		},
		{
			name: "generates simplified implementation prompt with few tasks keeps all",
			ctx: &WorkflowContext{
				Plan: &Plan{
					Summary:     "Simple feature",
					ContextType: "feature",
					Architecture: Architecture{
						Overview:   "Simple",
						Components: []string{"comp"},
					},
				},
			},
			workStream: WorkStream{
				Name:  "Backend",
				Tasks: []string{"task 1", "task 2"},
			},
			attempt: 2,
			wantErr: false,
			wantContain: []string{
				"Simple feature",
			},
			checkTasks: func(t *testing.T, prompt string, originalTasks []string, attempt int) {
				assert.Contains(t, prompt, "task 1")
				assert.Contains(t, prompt, "task 2")
			},
		},
		{
			name: "returns error when context is nil",
			ctx:  nil,
			workStream: WorkStream{
				Name:  "Backend",
				Tasks: []string{"task"},
			},
			attempt:     1,
			wantErr:     true,
			wantContain: nil,
		},
		{
			name: "returns error when plan is nil",
			ctx: &WorkflowContext{
				Plan: nil,
			},
			workStream: WorkStream{
				Name:  "Backend",
				Tasks: []string{"task"},
			},
			attempt:     1,
			wantErr:     true,
			wantContain: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewPromptGenerator()
			require.NoError(t, err)

			got, err := pg.GenerateSimplifiedImplementationPrompt(tt.ctx, tt.workStream, tt.attempt)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, got)

			if tt.ctx != nil && tt.ctx.Plan != nil {
				regularPrompt, err := pg.GenerateImplementationPrompt(tt.ctx.Plan)
				require.NoError(t, err)
				assert.Less(t, len(got), len(regularPrompt), "simplified prompt should be smaller than regular prompt")
			}

			for _, want := range tt.wantContain {
				assert.Contains(t, got, want)
			}

			if tt.checkTasks != nil {
				tt.checkTasks(t, got, tt.workStream.Tasks, tt.attempt)
			}
		})
	}
}

func TestPromptGenerator_GenerateSimplifiedRefactoringPrompt(t *testing.T) {
	tests := []struct {
		name        string
		ctx         *WorkflowContext
		attempt     int
		wantErr     bool
		wantContain []string
	}{
		{
			name: "generates simplified refactoring prompt",
			ctx: &WorkflowContext{
				Plan: &Plan{
					Summary:     "Add authentication",
					ContextType: "feature",
					Architecture: Architecture{
						Overview:   "JWT-based auth",
						Components: []string{"auth service", "middleware"},
					},
				},
			},
			attempt: 1,
			wantErr: false,
			wantContain: []string{
				"Add authentication",
				"JWT-based auth",
			},
		},
		{
			name: "generates simplified refactoring prompt on retry",
			ctx: &WorkflowContext{
				Plan: &Plan{
					Summary:     "Fix bug",
					ContextType: "fix",
					Architecture: Architecture{
						Overview:   "Bug fix",
						Components: []string{"component"},
					},
				},
			},
			attempt: 3,
			wantErr: false,
			wantContain: []string{
				"Fix bug",
				"Bug fix",
			},
		},
		{
			name:    "returns error when context is nil",
			ctx:     nil,
			attempt: 1,
			wantErr: true,
		},
		{
			name: "returns error when plan is nil",
			ctx: &WorkflowContext{
				Plan: nil,
			},
			attempt: 1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewPromptGenerator()
			require.NoError(t, err)

			got, err := pg.GenerateSimplifiedRefactoringPrompt(tt.ctx, tt.attempt)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, got)

			if tt.ctx != nil && tt.ctx.Plan != nil {
				regularPrompt, err := pg.GenerateRefactoringPrompt(tt.ctx.Plan)
				require.NoError(t, err)
				assert.Less(t, len(got), len(regularPrompt), "simplified prompt should be smaller than regular prompt")
			}

			for _, want := range tt.wantContain {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestPromptGenerator_GenerateSimplifiedPRSplitPrompt(t *testing.T) {
	tests := []struct {
		name          string
		ctx           *WorkflowContext
		attempt       int
		wantErr       bool
		wantContain   []string
		checkTruncate func(t *testing.T, prompt string, originalCommits []Commit)
	}{
		{
			name: "generates simplified PR split prompt with commit truncation",
			ctx: &WorkflowContext{
				Metrics: &PRMetrics{
					LinesChanged:  500,
					FilesChanged:  15,
					FilesAdded:    []string{"file1.go", "file2.go"},
					FilesModified: []string{"main.go"},
				},
				Commits: []Commit{
					{Hash: "commit1", Subject: "first commit"},
					{Hash: "commit2", Subject: "second commit"},
					{Hash: "commit3", Subject: "third commit"},
					{Hash: "commit4", Subject: "fourth commit"},
					{Hash: "commit5", Subject: "fifth commit"},
					{Hash: "commit6", Subject: "sixth commit"},
					{Hash: "commit7", Subject: "seventh commit"},
					{Hash: "commit8", Subject: "eighth commit"},
					{Hash: "commit9", Subject: "ninth commit"},
					{Hash: "commit10", Subject: "tenth commit"},
					{Hash: "commit11", Subject: "eleventh commit"},
					{Hash: "commit12", Subject: "twelfth commit"},
				},
			},
			attempt: 2,
			wantErr: false,
			wantContain: []string{
				"Lines: 500",
				"Files: 15",
			},
			checkTruncate: func(t *testing.T, prompt string, originalCommits []Commit) {
				assert.Contains(t, prompt, "- commit3 -")
				assert.Contains(t, prompt, "- commit12 -")
				assert.NotContains(t, prompt, "- commit1 -")
				assert.NotContains(t, prompt, "- commit2 -")
			},
		},
		{
			name: "generates simplified PR split prompt with few commits keeps all",
			ctx: &WorkflowContext{
				Metrics: &PRMetrics{
					LinesChanged: 100,
					FilesChanged: 5,
				},
				Commits: []Commit{
					{Hash: "abc123", Subject: "commit 1"},
					{Hash: "def456", Subject: "commit 2"},
					{Hash: "ghi789", Subject: "commit 3"},
				},
			},
			attempt: 1,
			wantErr: false,
			wantContain: []string{
				"Lines: 100",
				"Files: 5",
			},
			checkTruncate: func(t *testing.T, prompt string, originalCommits []Commit) {
				assert.Contains(t, prompt, "abc123")
				assert.Contains(t, prompt, "def456")
				assert.Contains(t, prompt, "ghi789")
			},
		},
		{
			name: "generates simplified PR split prompt with nil commits",
			ctx: &WorkflowContext{
				Metrics: &PRMetrics{
					LinesChanged: 50,
					FilesChanged: 2,
				},
				Commits: nil,
			},
			attempt: 1,
			wantErr: false,
			wantContain: []string{
				"Lines: 50",
				"Files: 2",
			},
		},
		{
			name:    "returns error when context is nil",
			ctx:     nil,
			attempt: 1,
			wantErr: true,
		},
		{
			name: "returns error when metrics is nil",
			ctx: &WorkflowContext{
				Metrics: nil,
			},
			attempt: 1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewPromptGenerator()
			require.NoError(t, err)

			got, err := pg.GenerateSimplifiedPRSplitPrompt(tt.ctx, tt.attempt)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, got)

			for _, want := range tt.wantContain {
				assert.Contains(t, got, want)
			}

			if tt.checkTruncate != nil && tt.ctx != nil {
				tt.checkTruncate(t, got, tt.ctx.Commits)
			}
		})
	}
}
