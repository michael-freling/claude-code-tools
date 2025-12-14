package workflow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStateManager(t *testing.T) {
	tests := []struct {
		name    string
		baseDir string
	}{
		{
			name:    "creates state manager with base directory",
			baseDir: "/tmp/workflows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewStateManager(tt.baseDir)
			assert.NotNil(t, got)
		})
	}
}

func TestFileStateManager_WorkflowDir(t *testing.T) {
	tests := []struct {
		name         string
		baseDir      string
		workflowName string
		want         string
	}{
		{
			name:         "returns correct workflow directory",
			baseDir:      "/tmp/workflows",
			workflowName: "test-workflow",
			want:         "/tmp/workflows/test-workflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewStateManager(tt.baseDir)
			got := sm.WorkflowDir(tt.workflowName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFileStateManager_EnsureWorkflowDir(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "creates workflow directory successfully",
			workflowName: "test-workflow",
			wantErr:      false,
		},
		{
			name:         "creates workflow directory with hyphens",
			workflowName: "my-test-workflow",
			wantErr:      false,
		},
		{
			name:         "returns error for invalid workflow name",
			workflowName: "../invalid",
			wantErr:      true,
			errContains:  "invalid workflow name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			err := sm.EnsureWorkflowDir(tt.workflowName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)

			workflowDir := sm.WorkflowDir(tt.workflowName)
			assert.DirExists(t, workflowDir)

			phasesDir := filepath.Join(workflowDir, "phases")
			assert.DirExists(t, phasesDir)
		})
	}
}

func TestFileStateManager_WorkflowExists(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		setup        func(sm StateManager)
		want         bool
	}{
		{
			name:         "returns false for non-existent workflow",
			workflowName: "non-existent",
			setup:        func(sm StateManager) {},
			want:         false,
		},
		{
			name:         "returns true for existing workflow",
			workflowName: "existing",
			setup: func(sm StateManager) {
				sm.InitState("existing", "test description", WorkflowTypeFeature)
			},
			want: true,
		},
		{
			name:         "returns false for invalid workflow name",
			workflowName: "../invalid",
			setup:        func(sm StateManager) {},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			tt.setup(sm)

			got := sm.WorkflowExists(tt.workflowName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFileStateManager_InitState(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		description  string
		wfType       WorkflowType
		wantErr      bool
		errContains  string
	}{
		{
			name:         "initializes state for feature workflow",
			workflowName: "auth-feature",
			description:  "add authentication",
			wfType:       WorkflowTypeFeature,
			wantErr:      false,
		},
		{
			name:         "initializes state for fix workflow",
			workflowName: "login-fix",
			description:  "fix login bug",
			wfType:       WorkflowTypeFix,
			wantErr:      false,
		},
		{
			name:         "returns error for invalid workflow name",
			workflowName: "../invalid",
			description:  "test",
			wfType:       WorkflowTypeFeature,
			wantErr:      true,
			errContains:  "invalid workflow name",
		},
		{
			name:         "returns error for invalid workflow type",
			workflowName: "test",
			description:  "test",
			wfType:       WorkflowType("invalid"),
			wantErr:      true,
			errContains:  "invalid workflow type",
		},
		{
			name:         "returns error for empty description",
			workflowName: "test",
			description:  "",
			wfType:       WorkflowTypeFeature,
			wantErr:      true,
			errContains:  "cannot be empty",
		},
		{
			name:         "returns error for existing workflow",
			workflowName: "existing",
			description:  "test",
			wfType:       WorkflowTypeFeature,
			wantErr:      true,
			errContains:  "workflow already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			if tt.name == "returns error for existing workflow" {
				sm.InitState(tt.workflowName, tt.description, tt.wfType)
			}

			got, err := sm.InitState(tt.workflowName, tt.description, tt.wfType)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			assert.Equal(t, "1.0", got.Version)
			assert.Equal(t, tt.workflowName, got.Name)
			assert.Equal(t, tt.wfType, got.Type)
			assert.Equal(t, tt.description, got.Description)
			assert.Equal(t, PhasePlanning, got.CurrentPhase)
			assert.NotZero(t, got.CreatedAt)
			assert.NotZero(t, got.UpdatedAt)

			require.NotNil(t, got.Phases)
			assert.Len(t, got.Phases, 5)

			assert.Equal(t, StatusInProgress, got.Phases[PhasePlanning].Status)
			assert.Equal(t, StatusPending, got.Phases[PhaseConfirmation].Status)
			assert.Equal(t, StatusPending, got.Phases[PhaseImplementation].Status)
			assert.Equal(t, StatusPending, got.Phases[PhaseRefactoring].Status)
			assert.Equal(t, StatusPending, got.Phases[PhasePRSplit].Status)
		})
	}
}

func TestFileStateManager_SaveAndLoadState(t *testing.T) {
	tests := []struct {
		name  string
		state *WorkflowState
	}{
		{
			name: "saves and loads state successfully",
			state: &WorkflowState{
				Version:      "1.0",
				Name:         "test-workflow",
				Type:         WorkflowTypeFeature,
				Description:  "test description",
				CurrentPhase: PhasePlanning,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
				Phases: map[Phase]*PhaseState{
					PhasePlanning: {
						Status:   StatusInProgress,
						Attempts: 1,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			err := sm.SaveState(tt.state.Name, tt.state)
			require.NoError(t, err)

			got, err := sm.LoadState(tt.state.Name)
			require.NoError(t, err)
			require.NotNil(t, got)

			assert.Equal(t, tt.state.Version, got.Version)
			assert.Equal(t, tt.state.Name, got.Name)
			assert.Equal(t, tt.state.Type, got.Type)
			assert.Equal(t, tt.state.Description, got.Description)
			assert.Equal(t, tt.state.CurrentPhase, got.CurrentPhase)
		})
	}
}

func TestFileStateManager_SaveState_Errors(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		state        *WorkflowState
		setup        func(tmpDir string)
		wantErr      bool
		errContains  string
	}{
		{
			name:         "returns error for invalid workflow name",
			workflowName: "../invalid",
			state: &WorkflowState{
				Version:      "1.0",
				Name:         "../invalid",
				Type:         WorkflowTypeFeature,
				Description:  "test",
				CurrentPhase: PhasePlanning,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
				Phases:       make(map[Phase]*PhaseState),
			},
			setup:       func(tmpDir string) {},
			wantErr:     true,
			errContains: "invalid workflow name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if tt.setup != nil {
				tt.setup(tmpDir)
			}
			sm := NewStateManager(tmpDir)

			err := sm.SaveState(tt.workflowName, tt.state)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestFileStateManager_SaveState_UpdatesTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStateManager(tmpDir)

	currentTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	sm.SetTimeProvider(func() time.Time { return currentTime })

	workflowName := "test-workflow"
	state, err := sm.InitState(workflowName, "test", WorkflowTypeFeature)
	require.NoError(t, err)

	originalTime := state.UpdatedAt
	assert.Equal(t, currentTime, originalTime)

	currentTime = currentTime.Add(1 * time.Second)
	state.CurrentPhase = PhaseImplementation
	err = sm.SaveState(workflowName, state)
	require.NoError(t, err)

	assert.True(t, state.UpdatedAt.After(originalTime))
	assert.Equal(t, currentTime, state.UpdatedAt)
}
func TestFileStateManager_LoadState_Errors(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		setup        func(tmpDir string)
		wantErr      bool
		errContains  string
	}{
		{
			name:         "returns error for invalid workflow name",
			workflowName: "../invalid",
			setup:        func(tmpDir string) {},
			wantErr:      true,
			errContains:  "invalid workflow name",
		},
		{
			name:         "returns error for non-existent workflow",
			workflowName: "non-existent",
			setup:        func(tmpDir string) {},
			wantErr:      true,
			errContains:  "workflow not found",
		},
		{
			name:         "returns error for corrupted state file",
			workflowName: "corrupted",
			setup: func(tmpDir string) {
				workflowDir := filepath.Join(tmpDir, "corrupted")
				os.MkdirAll(workflowDir, 0755)
				os.WriteFile(filepath.Join(workflowDir, "state.json"), []byte("invalid json"), 0644)
			},
			wantErr:     true,
			errContains: "state file corrupted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			tt.setup(tmpDir)

			got, err := sm.LoadState(tt.workflowName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestFileStateManager_SaveAndLoadPlan(t *testing.T) {
	tests := []struct {
		name string
		plan *Plan
	}{
		{
			name: "saves and loads plan successfully",
			plan: &Plan{
				Summary:     "test summary",
				ContextType: "feature",
				Architecture: Architecture{
					Overview:   "test overview",
					Components: []string{"component1", "component2"},
				},
				Phases: []PlanPhase{
					{
						Name:           "phase1",
						Description:    "test phase",
						EstimatedFiles: 5,
						EstimatedLines: 100,
					},
				},
				Complexity:          "medium",
				EstimatedTotalLines: 100,
				EstimatedTotalFiles: 5,
			},
		},
		{
			name: "saves and loads plan with minimal fields",
			plan: &Plan{
				Summary:     "minimal plan",
				ContextType: "fix",
				Architecture: Architecture{
					Overview:   "fix overview",
					Components: []string{},
				},
				Phases:              []PlanPhase{},
				Complexity:          "small",
				EstimatedTotalLines: 10,
				EstimatedTotalFiles: 1,
			},
		},
		{
			name: "saves and loads plan with many phases",
			plan: &Plan{
				Summary:     "complex plan",
				ContextType: "feature",
				Architecture: Architecture{
					Overview:   "complex architecture",
					Components: []string{"auth", "db", "api", "ui"},
				},
				Phases: []PlanPhase{
					{
						Name:           "database",
						Description:    "setup database",
						EstimatedFiles: 3,
						EstimatedLines: 50,
					},
					{
						Name:           "api",
						Description:    "implement api",
						EstimatedFiles: 10,
						EstimatedLines: 200,
					},
					{
						Name:           "ui",
						Description:    "build ui",
						EstimatedFiles: 15,
						EstimatedLines: 300,
					},
				},
				Complexity:          "high",
				EstimatedTotalLines: 550,
				EstimatedTotalFiles: 28,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			workflowName := "test-workflow"
			sm.InitState(workflowName, "test", WorkflowTypeFeature)

			err := sm.SavePlan(workflowName, tt.plan)
			require.NoError(t, err)

			got, err := sm.LoadPlan(workflowName)
			require.NoError(t, err)
			require.NotNil(t, got)

			assert.Equal(t, tt.plan.Summary, got.Summary)
			assert.Equal(t, tt.plan.ContextType, got.ContextType)
			assert.Equal(t, tt.plan.Complexity, got.Complexity)
			assert.Equal(t, len(tt.plan.Phases), len(got.Phases))
		})
	}
}

func TestFileStateManager_LoadPlan_Errors(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		setup        func(tmpDir string)
		wantErr      bool
		errContains  string
	}{
		{
			name:         "returns error for invalid workflow name",
			workflowName: "../invalid",
			setup:        func(tmpDir string) {},
			wantErr:      true,
			errContains:  "invalid workflow name",
		},
		{
			name:         "returns error for non-existent plan file",
			workflowName: "test-workflow",
			setup: func(tmpDir string) {
				sm := NewStateManager(tmpDir)
				sm.InitState("test-workflow", "test", WorkflowTypeFeature)
			},
			wantErr:     true,
			errContains: "failed to read plan file",
		},
		{
			name:         "returns error for corrupted plan file",
			workflowName: "corrupted",
			setup: func(tmpDir string) {
				workflowDir := filepath.Join(tmpDir, "corrupted")
				os.MkdirAll(workflowDir, 0755)
				os.WriteFile(filepath.Join(workflowDir, "plan.json"), []byte("invalid json"), 0644)
			},
			wantErr:     true,
			errContains: "failed to unmarshal plan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(tmpDir)
			sm := NewStateManager(tmpDir)

			got, err := sm.LoadPlan(tt.workflowName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestFileStateManager_SavePlan_Errors(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		plan         *Plan
		setup        func(tmpDir string) string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "returns error for invalid workflow name",
			workflowName: "../invalid",
			plan:         &Plan{Summary: "test"},
			wantErr:      true,
			errContains:  "invalid workflow name",
		},
		{
			name:         "successfully saves valid plan",
			workflowName: "valid-workflow",
			plan:         &Plan{Summary: "test plan", Complexity: "medium"},
			wantErr:      false,
		},
		{
			name:         "successfully creates workflow directory if needed",
			workflowName: "new-workflow",
			plan:         &Plan{Summary: "test", Complexity: "low"},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tmpDir string
			if tt.setup != nil {
				tmpDir = tt.setup(t.TempDir())
			} else {
				tmpDir = t.TempDir()
			}
			sm := NewStateManager(tmpDir)

			if !tt.wantErr {
				_, err := sm.InitState(tt.workflowName, "test", WorkflowTypeFeature)
				require.NoError(t, err)
			}

			err := sm.SavePlan(tt.workflowName, tt.plan)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestFileStateManager_SavePlanMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
	}{
		{
			name:     "saves plan markdown successfully",
			markdown: "# Test Plan\n\nThis is a test plan.",
		},
		{
			name:     "saves empty markdown",
			markdown: "",
		},
		{
			name: "saves complex markdown with formatting",
			markdown: `# Feature Implementation Plan

## Overview
This plan outlines the implementation of authentication.

## Components
- User service
- Auth middleware
- Database schema

## Phases
1. Database setup
2. API implementation
3. Frontend integration

## Risks
- Token expiration handling
- Password security

## Timeline
Estimated: 2 weeks`,
		},
		{
			name:     "saves markdown with special characters",
			markdown: "# Plan\n\n- Item 1 `code`\n- Item 2 **bold**\n- Item 3 *italic*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			workflowName := "test-workflow"
			sm.InitState(workflowName, "test", WorkflowTypeFeature)

			err := sm.SavePlanMarkdown(workflowName, tt.markdown)
			require.NoError(t, err)

			planPath := filepath.Join(sm.WorkflowDir(workflowName), "plan.md")
			assert.FileExists(t, planPath)

			content, err := os.ReadFile(planPath)
			require.NoError(t, err)
			assert.Equal(t, tt.markdown, string(content))
		})
	}
}

func TestFileStateManager_SavePlanMarkdown_Errors(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		markdown     string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "returns error for invalid workflow name",
			workflowName: "../invalid",
			markdown:     "test",
			wantErr:      true,
			errContains:  "invalid workflow name",
		},
		{
			name:         "successfully saves valid markdown",
			workflowName: "valid-workflow",
			markdown:     "# Test Plan\n\nThis is a test",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			if !tt.wantErr {
				_, err := sm.InitState(tt.workflowName, "test", WorkflowTypeFeature)
				require.NoError(t, err)
			}

			err := sm.SavePlanMarkdown(tt.workflowName, tt.markdown)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestFileStateManager_SaveAndLoadPhaseOutput(t *testing.T) {
	tests := []struct {
		name  string
		phase Phase
		data  interface{}
	}{
		{
			name:  "saves and loads implementation summary",
			phase: PhaseImplementation,
			data: &ImplementationSummary{
				FilesChanged: []string{"file1.go", "file2.go"},
				LinesAdded:   100,
				LinesRemoved: 50,
				TestsAdded:   5,
				Summary:      "test summary",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			workflowName := "test-workflow"
			sm.InitState(workflowName, "test", WorkflowTypeFeature)

			err := sm.SavePhaseOutput(workflowName, tt.phase, tt.data)
			require.NoError(t, err)

			var got ImplementationSummary
			err = sm.LoadPhaseOutput(workflowName, tt.phase, &got)
			require.NoError(t, err)

			expected := tt.data.(*ImplementationSummary)
			assert.Equal(t, expected.Summary, got.Summary)
			assert.Equal(t, expected.LinesAdded, got.LinesAdded)
		})
	}
}

func TestFileStateManager_SavePhaseOutput_VariousPhases(t *testing.T) {
	tests := []struct {
		name  string
		phase Phase
		data  interface{}
	}{
		{
			name:  "saves planning phase output",
			phase: PhasePlanning,
			data: &Plan{
				Summary:             "planning summary",
				ContextType:         "feature",
				Complexity:          "high",
				EstimatedTotalLines: 500,
				EstimatedTotalFiles: 10,
			},
		},
		{
			name:  "saves refactoring phase output",
			phase: PhaseRefactoring,
			data: &RefactoringSummary{
				FilesChanged:     []string{"main.go", "utils.go"},
				ImprovementsMade: []string{"improved error handling", "reduced complexity"},
				Summary:          "refactoring complete",
			},
		},
		{
			name:  "saves PR split phase output",
			phase: PhasePRSplit,
			data: &PRSplitResult{
				ParentPR: PRInfo{
					Number:      123,
					URL:         "https://github.com/org/repo/pull/123",
					Title:       "Parent PR",
					Description: "Main changes",
				},
				ChildPRs: []PRInfo{
					{
						Number:      124,
						URL:         "https://github.com/org/repo/pull/124",
						Title:       "Child PR 1",
						Description: "First part",
					},
				},
				Summary: "Split into parent and child PRs",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			workflowName := "test-workflow"
			_, err := sm.InitState(workflowName, "test", WorkflowTypeFeature)
			require.NoError(t, err)

			err = sm.SavePhaseOutput(workflowName, tt.phase, tt.data)
			require.NoError(t, err)

			phaseFile := filepath.Join(sm.WorkflowDir(workflowName), "phases", string(tt.phase)+".json")
			assert.FileExists(t, phaseFile)
		})
	}
}

func TestFileStateManager_SavePhaseOutput_Errors(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		phase        Phase
		data         interface{}
		wantErr      bool
		errContains  string
	}{
		{
			name:         "returns error for invalid workflow name",
			workflowName: "../invalid",
			phase:        PhaseImplementation,
			data:         &ImplementationSummary{Summary: "test"},
			wantErr:      true,
			errContains:  "invalid workflow name",
		},
		{
			name:         "successfully saves valid phase output",
			workflowName: "valid-workflow",
			phase:        PhaseImplementation,
			data:         &ImplementationSummary{Summary: "test summary", LinesAdded: 100},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			if !tt.wantErr {
				_, err := sm.InitState(tt.workflowName, "test", WorkflowTypeFeature)
				require.NoError(t, err)
			}

			err := sm.SavePhaseOutput(tt.workflowName, tt.phase, tt.data)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestFileStateManager_LoadPhaseOutput_Errors(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		phase        Phase
		setup        func(tmpDir string)
		wantErr      bool
		errContains  string
	}{
		{
			name:         "returns error for invalid workflow name",
			workflowName: "../invalid",
			phase:        PhaseImplementation,
			setup:        func(tmpDir string) {},
			wantErr:      true,
			errContains:  "invalid workflow name",
		},
		{
			name:         "returns error for non-existent phase output",
			workflowName: "test-workflow",
			phase:        PhaseImplementation,
			setup: func(tmpDir string) {
				sm := NewStateManager(tmpDir)
				sm.InitState("test-workflow", "test", WorkflowTypeFeature)
			},
			wantErr:     true,
			errContains: "failed to read phase output",
		},
		{
			name:         "returns error for corrupted phase output",
			workflowName: "corrupted",
			phase:        PhaseImplementation,
			setup: func(tmpDir string) {
				workflowDir := filepath.Join(tmpDir, "corrupted")
				phasesDir := filepath.Join(workflowDir, "phases")
				os.MkdirAll(phasesDir, 0755)
				os.WriteFile(filepath.Join(phasesDir, "IMPLEMENTATION.json"), []byte("invalid json"), 0644)
			},
			wantErr:     true,
			errContains: "failed to unmarshal phase output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(tmpDir)
			sm := NewStateManager(tmpDir)

			var got ImplementationSummary
			err := sm.LoadPhaseOutput(tt.workflowName, tt.phase, &got)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestFileStateManager_ListWorkflows(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(sm StateManager)
		wantCount int
	}{
		{
			name: "returns empty list for no workflows",
			setup: func(sm StateManager) {
			},
			wantCount: 0,
		},
		{
			name: "returns list of workflows",
			setup: func(sm StateManager) {
				sm.InitState("workflow1", "test1", WorkflowTypeFeature)
				sm.InitState("workflow2", "test2", WorkflowTypeFix)
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			tt.setup(sm)

			got, err := sm.ListWorkflows()
			require.NoError(t, err)
			assert.Len(t, got, tt.wantCount)
		})
	}
}

func TestFileStateManager_DeleteWorkflow(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		setup        func(sm StateManager)
		wantErr      bool
		errContains  string
	}{
		{
			name:         "deletes workflow successfully",
			workflowName: "test-workflow",
			setup: func(sm StateManager) {
				sm.InitState("test-workflow", "test", WorkflowTypeFeature)
			},
			wantErr: false,
		},
		{
			name:         "returns error for invalid workflow name",
			workflowName: "../invalid",
			setup:        func(sm StateManager) {},
			wantErr:      true,
			errContains:  "invalid workflow name",
		},
		{
			name:         "returns error for non-existent workflow",
			workflowName: "non-existent",
			setup:        func(sm StateManager) {},
			wantErr:      true,
			errContains:  "workflow not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			tt.setup(sm)

			err := sm.DeleteWorkflow(tt.workflowName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.False(t, sm.WorkflowExists(tt.workflowName))
		})
	}
}

func TestFileStateManager_SaveRawOutput(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		phase        Phase
		output       string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "saves raw output successfully",
			workflowName: "test-workflow",
			phase:        PhasePlanning,
			output:       "This is raw Claude output that failed to parse as JSON",
			wantErr:      false,
		},
		{
			name:         "saves empty output",
			workflowName: "test-workflow",
			phase:        PhaseImplementation,
			output:       "",
			wantErr:      false,
		},
		{
			name:         "returns error for invalid workflow name",
			workflowName: "../invalid",
			phase:        PhasePlanning,
			output:       "test",
			wantErr:      true,
			errContains:  "invalid workflow name",
		},
		{
			name:         "saves raw output for refactoring phase",
			workflowName: "test-workflow",
			phase:        PhaseRefactoring,
			output:       "Refactoring output",
			wantErr:      false,
		},
		{
			name:         "saves raw output for pr-split phase",
			workflowName: "test-workflow",
			phase:        PhasePRSplit,
			output:       "PR split output",
			wantErr:      false,
		},
		{
			name:         "saves raw output for confirmation phase",
			workflowName: "test-workflow",
			phase:        PhaseConfirmation,
			output:       "Confirmation output",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			if !tt.wantErr {
				_, err := sm.InitState(tt.workflowName, "test", WorkflowTypeFeature)
				require.NoError(t, err)
			}

			err := sm.SaveRawOutput(tt.workflowName, tt.phase, tt.output)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)

			rawFile := filepath.Join(sm.WorkflowDir(tt.workflowName), "phases", string(tt.phase)+"_raw.txt")
			assert.FileExists(t, rawFile)

			content, err := os.ReadFile(rawFile)
			require.NoError(t, err)
			assert.Equal(t, tt.output, string(content))
		})
	}
}

func TestFileStateManager_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStateManager(tmpDir)

	workflowName := "test-workflow"
	state, err := sm.InitState(workflowName, "test", WorkflowTypeFeature)
	require.NoError(t, err)

	err = sm.SaveState(workflowName, state)
	require.NoError(t, err)

	fsm, ok := sm.(*fileStateManager)
	require.True(t, ok)

	lock1, err := fsm.lock(workflowName)
	require.NoError(t, err)
	require.NotNil(t, lock1)

	lock2, err := fsm.lock(workflowName)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrStateLocked))
	assert.Nil(t, lock2)

	err = fsm.unlock(workflowName)
	require.NoError(t, err)

	lock3, err := fsm.lock(workflowName)
	require.NoError(t, err)
	require.NotNil(t, lock3)

	err = fsm.unlock(workflowName)
	require.NoError(t, err)
}

func TestFileStateManager_ListWorkflows_WithCorruptedState(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStateManager(tmpDir)

	_, err := sm.InitState("good-workflow", "test", WorkflowTypeFeature)
	require.NoError(t, err)

	corruptedDir := filepath.Join(tmpDir, "corrupted-workflow")
	err = os.MkdirAll(corruptedDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(corruptedDir, "state.json"), []byte("invalid json"), 0644)
	require.NoError(t, err)

	workflows, err := sm.ListWorkflows()
	require.NoError(t, err)

	assert.Len(t, workflows, 1)
	assert.Equal(t, "good-workflow", workflows[0].Name)
}

func TestFileStateManager_AtomicWrite_Error(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStateManager(tmpDir)
	fsm, ok := sm.(*fileStateManager)
	require.True(t, ok)

	invalidPath := filepath.Join("/proc/invalid/path/that/cannot/exist", "file.json")
	err := fsm.atomicWrite(invalidPath, []byte("test"))
	assert.Error(t, err)
}

func TestFileStateManager_AtomicWrite_Success(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStateManager(tmpDir)
	fsm, ok := sm.(*fileStateManager)
	require.True(t, ok)

	testPath := filepath.Join(tmpDir, "subdir", "test.json")
	testData := []byte("test data")

	err := fsm.atomicWrite(testPath, testData)
	require.NoError(t, err)

	content, err := os.ReadFile(testPath)
	require.NoError(t, err)
	assert.Equal(t, testData, content)
}

func TestFileStateManager_Unlock_NoLock(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStateManager(tmpDir)
	fsm, ok := sm.(*fileStateManager)
	require.True(t, ok)

	err := fsm.unlock("non-existent-workflow")
	assert.NoError(t, err)
}

func TestFileStateManager_ListWorkflows_WithNonDirectoryEntries(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStateManager(tmpDir)

	_, err := sm.InitState("workflow1", "test1", WorkflowTypeFeature)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "regular-file.txt"), []byte("test"), 0644)
	require.NoError(t, err)

	workflows, err := sm.ListWorkflows()
	require.NoError(t, err)

	assert.Len(t, workflows, 1)
	assert.Equal(t, "workflow1", workflows[0].Name)
}

func TestFileStateManager_EnsureWorkflowDir_CreateDirError(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "successfully creates workflow directory",
			workflowName: "valid-workflow",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			err := sm.EnsureWorkflowDir(tt.workflowName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestFileStateManager_Lock_MultipleLocks(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStateManager(tmpDir)

	_, err := sm.InitState("test-workflow", "test", WorkflowTypeFeature)
	require.NoError(t, err)

	fsm, ok := sm.(*fileStateManager)
	require.True(t, ok)

	lock1, err := fsm.lock("test-workflow")
	require.NoError(t, err)
	require.NotNil(t, lock1)

	err = fsm.unlock("test-workflow")
	require.NoError(t, err)

	err = fsm.unlock("test-workflow")
	require.NoError(t, err)

	lock2, err := fsm.lock("test-workflow")
	require.NoError(t, err)
	require.NotNil(t, lock2)

	err = fsm.unlock("test-workflow")
	require.NoError(t, err)

	err = lock1.Close()
	require.NoError(t, err)

	err = lock2.Close()
	require.NoError(t, err)
}

func TestFileStateManager_ListWorkflows_StatusDetection(t *testing.T) {
	tests := []struct {
		name       string
		setupState func(*WorkflowState)
		wantStatus string
	}{
		{
			name: "completed workflow",
			setupState: func(s *WorkflowState) {
				s.CurrentPhase = PhaseCompleted
			},
			wantStatus: "completed",
		},
		{
			name: "failed workflow",
			setupState: func(s *WorkflowState) {
				s.CurrentPhase = PhaseFailed
			},
			wantStatus: "failed",
		},
		{
			name: "workflow with error",
			setupState: func(s *WorkflowState) {
				s.Error = &WorkflowError{
					Message: "test error",
					Phase:   PhasePlanning,
				}
			},
			wantStatus: "failed",
		},
		{
			name: "in progress workflow",
			setupState: func(s *WorkflowState) {
				s.CurrentPhase = PhaseImplementation
			},
			wantStatus: "in_progress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			state, err := sm.InitState("test-workflow", "test", WorkflowTypeFeature)
			require.NoError(t, err)

			tt.setupState(state)
			err = sm.SaveState("test-workflow", state)
			require.NoError(t, err)

			workflows, err := sm.ListWorkflows()
			require.NoError(t, err)

			assert.Len(t, workflows, 1)
			assert.Equal(t, tt.wantStatus, workflows[0].Status)
		})
	}
}

func TestFileStateManager_SavePrompt(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		phase        Phase
		attempt      int
		prompt       string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "saves prompt successfully for planning phase attempt 1",
			workflowName: "test-workflow",
			phase:        PhasePlanning,
			attempt:      1,
			prompt:       "This is a test prompt for planning",
			wantErr:      false,
		},
		{
			name:         "saves prompt successfully for implementation phase attempt 2",
			workflowName: "test-workflow",
			phase:        PhaseImplementation,
			attempt:      2,
			prompt:       "This is a test prompt for implementation on second attempt",
			wantErr:      false,
		},
		{
			name:         "returns error for empty prompt",
			workflowName: "test-workflow",
			phase:        PhaseRefactoring,
			attempt:      1,
			prompt:       "",
			wantErr:      true,
			errContains:  "prompt cannot be empty",
		},
		{
			name:         "saves very long prompt",
			workflowName: "test-workflow",
			phase:        PhasePlanning,
			attempt:      3,
			prompt:       string(make([]byte, 100000)),
			wantErr:      false,
		},
		{
			name:         "returns error for invalid workflow name",
			workflowName: "../invalid",
			phase:        PhasePlanning,
			attempt:      1,
			prompt:       "test",
			wantErr:      true,
			errContains:  "invalid workflow name",
		},
		{
			name:         "returns error for zero attempt",
			workflowName: "test-workflow",
			phase:        PhasePlanning,
			attempt:      0,
			prompt:       "test",
			wantErr:      true,
			errContains:  "attempt must be positive",
		},
		{
			name:         "returns error for negative attempt",
			workflowName: "test-workflow",
			phase:        PhasePlanning,
			attempt:      -1,
			prompt:       "test",
			wantErr:      true,
			errContains:  "attempt must be positive",
		},
		{
			name:         "saves prompt for confirmation phase",
			workflowName: "test-workflow",
			phase:        PhaseConfirmation,
			attempt:      1,
			prompt:       "Confirmation prompt",
			wantErr:      false,
		},
		{
			name:         "saves prompt for pr-split phase",
			workflowName: "test-workflow",
			phase:        PhasePRSplit,
			attempt:      5,
			prompt:       "PR split prompt",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sm := NewStateManager(tmpDir)

			if !tt.wantErr {
				_, err := sm.InitState(tt.workflowName, "test", WorkflowTypeFeature)
				require.NoError(t, err)
			}

			gotPath, err := sm.SavePrompt(tt.workflowName, tt.phase, tt.attempt, tt.prompt)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Empty(t, gotPath)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, gotPath)

			expectedFilename := fmt.Sprintf("%s_attempt%d.txt", string(tt.phase), tt.attempt)
			expectedPath := filepath.Join(sm.WorkflowDir(tt.workflowName), "prompts", expectedFilename)
			assert.Equal(t, expectedPath, gotPath)

			assert.FileExists(t, gotPath)

			content, err := os.ReadFile(gotPath)
			require.NoError(t, err)
			assert.Equal(t, tt.prompt, string(content))
		})
	}
}

func TestFileStateManager_SavePrompt_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStateManager(tmpDir)

	workflowName := "test-workflow"
	_, err := sm.InitState(workflowName, "test", WorkflowTypeFeature)
	require.NoError(t, err)

	promptsDir := filepath.Join(sm.WorkflowDir(workflowName), "prompts")
	_, err = os.Stat(promptsDir)
	assert.True(t, os.IsNotExist(err))

	_, err = sm.SavePrompt(workflowName, PhasePlanning, 1, "test prompt")
	require.NoError(t, err)

	assert.DirExists(t, promptsDir)
}

func TestFileStateManager_SavePrompt_Overwriting(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStateManager(tmpDir)

	workflowName := "test-workflow"
	_, err := sm.InitState(workflowName, "test", WorkflowTypeFeature)
	require.NoError(t, err)

	firstPrompt := "First prompt content"
	path1, err := sm.SavePrompt(workflowName, PhasePlanning, 1, firstPrompt)
	require.NoError(t, err)

	content1, err := os.ReadFile(path1)
	require.NoError(t, err)
	assert.Equal(t, firstPrompt, string(content1))

	secondPrompt := "Second prompt content that overwrites the first"
	path2, err := sm.SavePrompt(workflowName, PhasePlanning, 1, secondPrompt)
	require.NoError(t, err)

	assert.Equal(t, path1, path2)

	content2, err := os.ReadFile(path2)
	require.NoError(t, err)
	assert.Equal(t, secondPrompt, string(content2))
}

func TestFileStateManager_SavePrompt_MultipleAttempts(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewStateManager(tmpDir)

	workflowName := "test-workflow"
	_, err := sm.InitState(workflowName, "test", WorkflowTypeFeature)
	require.NoError(t, err)

	path1, err := sm.SavePrompt(workflowName, PhasePlanning, 1, "Attempt 1")
	require.NoError(t, err)

	path2, err := sm.SavePrompt(workflowName, PhasePlanning, 2, "Attempt 2")
	require.NoError(t, err)

	path3, err := sm.SavePrompt(workflowName, PhasePlanning, 3, "Attempt 3")
	require.NoError(t, err)

	assert.NotEqual(t, path1, path2)
	assert.NotEqual(t, path2, path3)
	assert.NotEqual(t, path1, path3)

	assert.FileExists(t, path1)
	assert.FileExists(t, path2)
	assert.FileExists(t, path3)

	content1, err := os.ReadFile(path1)
	require.NoError(t, err)
	assert.Equal(t, "Attempt 1", string(content1))

	content2, err := os.ReadFile(path2)
	require.NoError(t, err)
	assert.Equal(t, "Attempt 2", string(content2))

	content3, err := os.ReadFile(path3)
	require.NoError(t, err)
	assert.Equal(t, "Attempt 3", string(content3))
}
