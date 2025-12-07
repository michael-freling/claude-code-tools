package workflow

import (
	"errors"
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
