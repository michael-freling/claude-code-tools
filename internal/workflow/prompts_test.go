package workflow

import (
	"strings"
	"testing"

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
		wantErr     bool
		errContains string
		wantContain []string
	}{
		{
			name: "generates PR split prompt with complete metrics",
			metrics: &PRMetrics{
				LinesChanged:  423,
				FilesChanged:  12,
				FilesAdded:    []string{"auth.go", "middleware.go"},
				FilesModified: []string{"main.go", "routes.go"},
				FilesDeleted:  []string{"old_auth.go"},
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
				"Output Format",
			},
		},
		{
			name:        "returns error when metrics is nil",
			metrics:     nil,
			wantErr:     true,
			errContains: "metrics cannot be nil",
		},
		{
			name: "generates prompt with minimal metrics",
			metrics: &PRMetrics{
				LinesChanged: 50,
				FilesChanged: 3,
			},
			wantErr: false,
			wantContain: []string{
				"Lines Changed: 50",
				"Files Changed: 3",
			},
		},
		{
			name: "generates prompt with only added files",
			metrics: &PRMetrics{
				LinesChanged: 100,
				FilesChanged: 5,
				FilesAdded:   []string{"file1.go", "file2.go"},
			},
			wantErr: false,
			wantContain: []string{
				"Lines Changed: 100",
				"Files Changed: 5",
				"Files Added",
				"file1.go",
				"file2.go",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewPromptGenerator()
			require.NoError(t, err)

			got, err := pg.GeneratePRSplitPrompt(tt.metrics)

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
				return pg.GeneratePRSplitPrompt(testMetrics)
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
