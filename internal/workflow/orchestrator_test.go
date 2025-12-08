package workflow

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockClaudeExecutor is a mock implementation of ClaudeExecutor
type MockClaudeExecutor struct {
	mock.Mock
}

func (m *MockClaudeExecutor) Execute(ctx context.Context, config ExecuteConfig) (*ExecuteResult, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ExecuteResult), args.Error(1)
}

func (m *MockClaudeExecutor) ExecuteStreaming(ctx context.Context, config ExecuteConfig, onProgress func(ProgressEvent)) (*ExecuteResult, error) {
	args := m.Called(ctx, config, onProgress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ExecuteResult), args.Error(1)
}

// MockStateManager is a mock implementation of StateManager
type MockStateManager struct {
	mock.Mock
}

func (m *MockStateManager) EnsureWorkflowDir(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockStateManager) WorkflowExists(name string) bool {
	args := m.Called(name)
	return args.Bool(0)
}

func (m *MockStateManager) WorkflowDir(name string) string {
	args := m.Called(name)
	return args.String(0)
}

func (m *MockStateManager) LoadState(name string) (*WorkflowState, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WorkflowState), args.Error(1)
}

func (m *MockStateManager) SaveState(name string, state *WorkflowState) error {
	args := m.Called(name, state)
	return args.Error(0)
}

func (m *MockStateManager) InitState(name, description string, wfType WorkflowType) (*WorkflowState, error) {
	args := m.Called(name, description, wfType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WorkflowState), args.Error(1)
}

func (m *MockStateManager) SavePlan(name string, plan *Plan) error {
	args := m.Called(name, plan)
	return args.Error(0)
}

func (m *MockStateManager) LoadPlan(name string) (*Plan, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Plan), args.Error(1)
}

func (m *MockStateManager) SavePlanMarkdown(name string, markdown string) error {
	args := m.Called(name, markdown)
	return args.Error(0)
}

func (m *MockStateManager) SavePhaseOutput(name string, phase Phase, data interface{}) error {
	args := m.Called(name, phase, data)
	return args.Error(0)
}

func (m *MockStateManager) LoadPhaseOutput(name string, phase Phase, target interface{}) error {
	args := m.Called(name, phase, target)
	return args.Error(0)
}

func (m *MockStateManager) ListWorkflows() ([]WorkflowInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]WorkflowInfo), args.Error(1)
}

func (m *MockStateManager) DeleteWorkflow(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockStateManager) SaveRawOutput(name string, phase Phase, output string) error {
	args := m.Called(name, phase, output)
	return args.Error(0)
}

// MockPromptGenerator is a mock implementation of PromptGenerator
type MockPromptGenerator struct {
	mock.Mock
}

func (m *MockPromptGenerator) GeneratePlanningPrompt(wfType WorkflowType, description string, feedback []string) (string, error) {
	args := m.Called(wfType, description, feedback)
	return args.String(0), args.Error(1)
}

func (m *MockPromptGenerator) GenerateImplementationPrompt(plan *Plan) (string, error) {
	args := m.Called(plan)
	return args.String(0), args.Error(1)
}

func (m *MockPromptGenerator) GenerateRefactoringPrompt(plan *Plan) (string, error) {
	args := m.Called(plan)
	return args.String(0), args.Error(1)
}

func (m *MockPromptGenerator) GeneratePRSplitPrompt(metrics *PRMetrics) (string, error) {
	args := m.Called(metrics)
	return args.String(0), args.Error(1)
}

func (m *MockPromptGenerator) GenerateFixCIPrompt(failures string) (string, error) {
	args := m.Called(failures)
	return args.String(0), args.Error(1)
}

// MockCIChecker is a mock implementation of CIChecker
type MockCIChecker struct {
	mock.Mock
}

func (m *MockCIChecker) CheckCI(ctx context.Context, prNumber int) (*CIResult, error) {
	args := m.Called(ctx, prNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CIResult), args.Error(1)
}

func (m *MockCIChecker) WaitForCI(ctx context.Context, prNumber int, timeout time.Duration) (*CIResult, error) {
	args := m.Called(ctx, prNumber, timeout)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CIResult), args.Error(1)
}

func (m *MockCIChecker) WaitForCIWithOptions(ctx context.Context, prNumber int, timeout time.Duration, opts CheckCIOptions) (*CIResult, error) {
	args := m.Called(ctx, prNumber, timeout, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CIResult), args.Error(1)
}

func (m *MockCIChecker) WaitForCIWithProgress(ctx context.Context, prNumber int, timeout time.Duration, opts CheckCIOptions, onProgress CIProgressCallback) (*CIResult, error) {
	args := m.Called(ctx, prNumber, timeout, opts, onProgress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CIResult), args.Error(1)
}

// MockOutputParser is a mock implementation of OutputParser
type MockOutputParser struct {
	mock.Mock
}

func (m *MockOutputParser) ExtractJSON(output string) (string, error) {
	args := m.Called(output)
	return args.String(0), args.Error(1)
}

func (m *MockOutputParser) ParsePlan(jsonStr string) (*Plan, error) {
	args := m.Called(jsonStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Plan), args.Error(1)
}

func (m *MockOutputParser) ParseImplementationSummary(jsonStr string) (*ImplementationSummary, error) {
	args := m.Called(jsonStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ImplementationSummary), args.Error(1)
}

func (m *MockOutputParser) ParseRefactoringSummary(jsonStr string) (*RefactoringSummary, error) {
	args := m.Called(jsonStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RefactoringSummary), args.Error(1)
}

func (m *MockOutputParser) ParsePRSplitResult(jsonStr string) (*PRSplitResult, error) {
	args := m.Called(jsonStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PRSplitResult), args.Error(1)
}

// MockWorktreeManager is a mock implementation of WorktreeManager
type MockWorktreeManager struct {
	mock.Mock
}

func (m *MockWorktreeManager) CreateWorktree(workflowName string) (string, error) {
	args := m.Called(workflowName)
	return args.String(0), args.Error(1)
}

func (m *MockWorktreeManager) WorktreeExists(path string) bool {
	args := m.Called(path)
	return args.Bool(0)
}

func (m *MockWorktreeManager) DeleteWorktree(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func TestNewOrchestrator(t *testing.T) {
	tests := []struct {
		name    string
		baseDir string
		wantErr bool
	}{
		{
			name:    "creates orchestrator with valid baseDir",
			baseDir: "/tmp/workflows",
			wantErr: false,
		},
		{
			name:    "fails with empty baseDir",
			baseDir: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewOrchestrator(tt.baseDir)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
			assert.NotNil(t, got.stateManager)
			assert.NotNil(t, got.executor)
			assert.NotNil(t, got.promptGenerator)
			assert.NotNil(t, got.parser)
			assert.NotNil(t, got.config)
		})
	}
}

func TestNewOrchestratorWithConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "creates orchestrator with valid config",
			config:  DefaultConfig("/tmp/workflows"),
			wantErr: false,
		},
		{
			name:    "fails with nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "fails with empty baseDir",
			config: &Config{
				BaseDir: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewOrchestratorWithConfig(tt.config)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestOrchestrator_executePlanning(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockStateManager, *MockClaudeExecutor, *MockPromptGenerator, *MockOutputParser)
		wantErr       bool
		wantNextPhase Phase
	}{
		{
			name: "successfully generates plan",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				pg.On("GeneratePlanningPrompt", WorkflowTypeFeature, "test description", []string(nil)).Return("planning prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"test plan\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"test plan\"}", nil)
				op.On("ParsePlan", mock.Anything).Return(&Plan{Summary: "test plan"}, nil)
				sm.On("SavePlan", "test-workflow", mock.Anything).Return(nil)
				sm.On("SavePlanMarkdown", "test-workflow", mock.Anything).Return(nil)
				sm.On("SavePhaseOutput", "test-workflow", PhasePlanning, mock.Anything).Return(nil)
			},
			wantErr:       false,
			wantNextPhase: PhaseConfirmation,
		},
		{
			name: "fails when executor fails",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				pg.On("GeneratePlanningPrompt", WorkflowTypeFeature, "test description", []string(nil)).Return("planning prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return((*ExecuteResult)(nil), errors.New("execution failed"))
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			mockExec := new(MockClaudeExecutor)
			mockPG := new(MockPromptGenerator)
			mockOP := new(MockOutputParser)

			tt.setupMocks(mockSM, mockExec, mockPG, mockOP)

			o := &Orchestrator{
				stateManager:    mockSM,
				executor:        mockExec,
				promptGenerator: mockPG,
				parser:          mockOP,
				config:          DefaultConfig("/tmp/workflows"),
			}

			state := &WorkflowState{
				Name:         "test-workflow",
				Type:         WorkflowTypeFeature,
				Description:  "test description",
				CurrentPhase: PhasePlanning,
				Phases: map[Phase]*PhaseState{
					PhasePlanning:       {Status: StatusInProgress},
					PhaseConfirmation:   {Status: StatusPending},
					PhaseImplementation: {Status: StatusPending},
					PhaseRefactoring:    {Status: StatusPending},
					PhasePRSplit:        {Status: StatusPending},
				},
			}

			err := o.executePlanning(context.Background(), state)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantNextPhase, state.CurrentPhase)
			mockSM.AssertExpectations(t)
			mockExec.AssertExpectations(t)
			mockPG.AssertExpectations(t)
			mockOP.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_executeConfirmation(t *testing.T) {
	tests := []struct {
		name          string
		confirmFunc   func(plan *Plan) (bool, string, error)
		setupMocks    func(*MockStateManager)
		wantErr       bool
		wantNextPhase Phase
	}{
		{
			name: "user approves plan",
			confirmFunc: func(plan *Plan) (bool, string, error) {
				return true, "", nil
			},
			setupMocks: func(sm *MockStateManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil).Times(2)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
			},
			wantErr:       false,
			wantNextPhase: PhaseImplementation,
		},
		{
			name: "user rejects with feedback",
			confirmFunc: func(plan *Plan) (bool, string, error) {
				return false, "please add more tests", nil
			},
			setupMocks: func(sm *MockStateManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil).Times(2)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
			},
			wantErr:       false,
			wantNextPhase: PhasePlanning,
		},
		{
			name: "confirmation fails",
			confirmFunc: func(plan *Plan) (bool, string, error) {
				return false, "", errors.New("user cancelled")
			},
			setupMocks: func(sm *MockStateManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
		{
			name: "fails when LoadPlan fails",
			confirmFunc: func(plan *Plan) (bool, string, error) {
				return true, "", nil
			},
			setupMocks: func(sm *MockStateManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return((*Plan)(nil), errors.New("load plan failed"))
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			tt.setupMocks(mockSM)

			o := &Orchestrator{
				stateManager: mockSM,
				config:       DefaultConfig("/tmp/workflows"),
				confirmFunc:  tt.confirmFunc,
			}

			state := &WorkflowState{
				Name:         "test-workflow",
				CurrentPhase: PhaseConfirmation,
				Phases: map[Phase]*PhaseState{
					PhasePlanning:       {Status: StatusCompleted},
					PhaseConfirmation:   {Status: StatusInProgress},
					PhaseImplementation: {Status: StatusPending},
					PhaseRefactoring:    {Status: StatusPending},
					PhasePRSplit:        {Status: StatusPending},
				},
			}

			err := o.executeConfirmation(context.Background(), state)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantNextPhase, state.CurrentPhase)
			mockSM.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_executeImplementation(t *testing.T) {
	tests := []struct {
		name             string
		initialWorktree  string
		setupMocks       func(*MockStateManager, *MockClaudeExecutor, *MockPromptGenerator, *MockOutputParser, *MockCIChecker, *MockWorktreeManager)
		wantErr          bool
		wantNextPhase    Phase
		wantWorktreePath string
	}{
		{
			name:            "successfully implements plan with pre-commit passing",
			initialWorktree: "",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				wm.On("CreateWorktree", "test-workflow").Return("/tmp/worktrees/test-workflow", nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateImplementationPrompt", mock.Anything).Return("implementation prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.MatchedBy(func(config ExecuteConfig) bool {
					return config.WorkingDirectory == "/tmp/worktrees/test-workflow"
				}), mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"implemented\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"implemented\"}", nil)
				op.On("ParseImplementationSummary", mock.Anything).Return(&ImplementationSummary{Summary: "implemented"}, nil)
				sm.On("SavePhaseOutput", "test-workflow", PhaseImplementation, mock.Anything).Return(nil)
				ci.On("WaitForCIWithProgress", mock.Anything, 0, mock.Anything, mock.Anything, mock.Anything).Return(&CIResult{Passed: true, Status: "success"}, nil)
			},
			wantErr:          false,
			wantNextPhase:    PhaseRefactoring,
			wantWorktreePath: "/tmp/worktrees/test-workflow",
		},
		{
			name:            "skips worktree creation when WorktreePath already set (resume scenario)",
			initialWorktree: "/existing/worktree/path",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				// Note: CreateWorktree should NOT be called
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateImplementationPrompt", mock.Anything).Return("implementation prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.MatchedBy(func(config ExecuteConfig) bool {
					return config.WorkingDirectory == "/existing/worktree/path"
				}), mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"implemented\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"implemented\"}", nil)
				op.On("ParseImplementationSummary", mock.Anything).Return(&ImplementationSummary{Summary: "implemented"}, nil)
				sm.On("SavePhaseOutput", "test-workflow", PhaseImplementation, mock.Anything).Return(nil)
				ci.On("WaitForCIWithProgress", mock.Anything, 0, mock.Anything, mock.Anything, mock.Anything).Return(&CIResult{Passed: true, Status: "success"}, nil)
			},
			wantErr:          false,
			wantNextPhase:    PhaseRefactoring,
			wantWorktreePath: "/existing/worktree/path",
		},
		{
			name:            "fails when worktree creation fails",
			initialWorktree: "",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				wm.On("CreateWorktree", "test-workflow").Return("", errors.New("branch already exists"))
			},
			wantErr:          true,
			wantNextPhase:    PhaseFailed,
			wantWorktreePath: "",
		},
		{
			name:            "CI check uses current branch PR automatically",
			initialWorktree: "",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				wm.On("CreateWorktree", "test-workflow").Return("/tmp/worktrees/test-workflow", nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateImplementationPrompt", mock.Anything).Return("implementation prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"implemented\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"implemented\"}", nil)
				op.On("ParseImplementationSummary", mock.Anything).Return(&ImplementationSummary{Summary: "implemented"}, nil)
				sm.On("SavePhaseOutput", "test-workflow", PhaseImplementation, mock.Anything).Return(nil)
				// CI check uses 0 for PR number (auto-detect current branch)
				ci.On("WaitForCIWithProgress", mock.Anything, 0, mock.Anything, mock.Anything, mock.Anything).Return(&CIResult{Passed: true, Status: "success"}, nil)
			},
			wantErr:          false,
			wantNextPhase:    PhaseRefactoring,
			wantWorktreePath: "/tmp/worktrees/test-workflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			mockExec := new(MockClaudeExecutor)
			mockPG := new(MockPromptGenerator)
			mockOP := new(MockOutputParser)
			mockCI := new(MockCIChecker)
			mockWM := new(MockWorktreeManager)

			tt.setupMocks(mockSM, mockExec, mockPG, mockOP, mockCI, mockWM)

			o := &Orchestrator{
				stateManager:    mockSM,
				executor:        mockExec,
				promptGenerator: mockPG,
				parser:          mockOP,
				config:          DefaultConfig("/tmp/workflows"),
				worktreeManager: mockWM,
				ciCheckerFactory: func(workingDir string, checkInterval time.Duration, commandTimeout time.Duration) CIChecker {
					return mockCI
				},
			}

			state := &WorkflowState{
				Name:         "test-workflow",
				CurrentPhase: PhaseImplementation,
				WorktreePath: tt.initialWorktree,
				Phases: map[Phase]*PhaseState{
					PhasePlanning:       {Status: StatusCompleted},
					PhaseConfirmation:   {Status: StatusCompleted},
					PhaseImplementation: {Status: StatusInProgress},
					PhaseRefactoring:    {Status: StatusPending},
					PhasePRSplit:        {Status: StatusPending},
				},
			}

			err := o.executeImplementation(context.Background(), state)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantNextPhase, state.CurrentPhase)
			assert.Equal(t, tt.wantWorktreePath, state.WorktreePath)
			mockSM.AssertExpectations(t)
			mockExec.AssertExpectations(t)
			mockPG.AssertExpectations(t)
			mockOP.AssertExpectations(t)
			mockWM.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_executeImplementation_ErrorPaths(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockStateManager, *MockClaudeExecutor, *MockPromptGenerator, *MockOutputParser, *MockCIChecker, *MockWorktreeManager)
		wantErr    bool
	}{
		{
			name: "fails when LoadPlan fails",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				wm.On("CreateWorktree", "test-workflow").Return("/tmp/worktrees/test-workflow", nil)
				sm.On("LoadPlan", "test-workflow").Return((*Plan)(nil), errors.New("load plan failed"))
			},
			wantErr: true,
		},
		{
			name: "fails when ExtractJSON fails",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				wm.On("CreateWorktree", "test-workflow").Return("/tmp/worktrees/test-workflow", nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateImplementationPrompt", mock.Anything).Return("implementation prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "invalid json output",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("", errors.New("no JSON found"))
				sm.On("WorkflowDir", "test-workflow").Return("/tmp/workflows/test-workflow")
				sm.On("SaveRawOutput", "test-workflow", PhaseImplementation, mock.Anything).Return(nil)
			},
			wantErr: true,
		},
		{
			name: "fails when ParseImplementationSummary fails",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				wm.On("CreateWorktree", "test-workflow").Return("/tmp/worktrees/test-workflow", nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateImplementationPrompt", mock.Anything).Return("implementation prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"invalid\": \"schema\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"invalid\": \"schema\"}", nil)
				op.On("ParseImplementationSummary", mock.Anything).Return((*ImplementationSummary)(nil), errors.New("invalid schema"))
				sm.On("WorkflowDir", "test-workflow").Return("/tmp/workflows/test-workflow")
				sm.On("SaveRawOutput", "test-workflow", PhaseImplementation, mock.Anything).Return(nil)
			},
			wantErr: true,
		},
		{
			name: "fails when SavePhaseOutput fails",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				wm.On("CreateWorktree", "test-workflow").Return("/tmp/worktrees/test-workflow", nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateImplementationPrompt", mock.Anything).Return("implementation prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"implemented\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"implemented\"}", nil)
				op.On("ParseImplementationSummary", mock.Anything).Return(&ImplementationSummary{Summary: "implemented"}, nil)
				sm.On("SavePhaseOutput", "test-workflow", PhaseImplementation, mock.Anything).Return(errors.New("save failed"))
			},
			wantErr: true,
		},
		{
			name: "fails when CI check fails",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				wm.On("CreateWorktree", "test-workflow").Return("/tmp/worktrees/test-workflow", nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateImplementationPrompt", mock.Anything).Return("implementation prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"implemented\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"implemented\"}", nil)
				op.On("ParseImplementationSummary", mock.Anything).Return(&ImplementationSummary{Summary: "implemented"}, nil)
				sm.On("SavePhaseOutput", "test-workflow", PhaseImplementation, mock.Anything).Return(nil)
				ci.On("WaitForCIWithProgress", mock.Anything, 0, mock.Anything, mock.Anything, mock.Anything).Return((*CIResult)(nil), errors.New("CI check timeout"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			mockExec := new(MockClaudeExecutor)
			mockPG := new(MockPromptGenerator)
			mockOP := new(MockOutputParser)
			mockCI := new(MockCIChecker)
			mockWM := new(MockWorktreeManager)

			tt.setupMocks(mockSM, mockExec, mockPG, mockOP, mockCI, mockWM)

			o := &Orchestrator{
				stateManager:    mockSM,
				executor:        mockExec,
				promptGenerator: mockPG,
				parser:          mockOP,
				config:          DefaultConfig("/tmp/workflows"),
				worktreeManager: mockWM,
				ciCheckerFactory: func(workingDir string, checkInterval time.Duration, commandTimeout time.Duration) CIChecker {
					return mockCI
				},
			}

			state := &WorkflowState{
				Name:         "test-workflow",
				CurrentPhase: PhaseImplementation,
				Phases: map[Phase]*PhaseState{
					PhasePlanning:       {Status: StatusCompleted},
					PhaseConfirmation:   {Status: StatusCompleted},
					PhaseImplementation: {Status: StatusInProgress},
					PhaseRefactoring:    {Status: StatusPending},
					PhasePRSplit:        {Status: StatusPending},
				},
			}

			err := o.executeImplementation(context.Background(), state)

			require.Error(t, err)
			assert.Equal(t, PhaseFailed, state.CurrentPhase)
			mockSM.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_executeImplementation_CIRetryLoop(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockStateManager, *MockClaudeExecutor, *MockPromptGenerator, *MockOutputParser, *MockCIChecker, *MockWorktreeManager)
		wantErr       bool
		wantNextPhase Phase
	}{
		{
			name: "retries when CI fails and succeeds on second attempt",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				wm.On("CreateWorktree", "test-workflow").Return("/tmp/worktrees/test-workflow", nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)

				pg.On("GenerateImplementationPrompt", mock.Anything).Return("implementation prompt", nil).Once()
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"implemented\"}\n```",
					ExitCode: 0,
				}, nil).Once()
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"implemented\"}", nil).Once()
				op.On("ParseImplementationSummary", mock.Anything).Return(&ImplementationSummary{Summary: "implemented"}, nil).Once()
				sm.On("SavePhaseOutput", "test-workflow", PhaseImplementation, mock.Anything).Return(nil).Once()

				ci.On("WaitForCIWithProgress", mock.Anything, 0, mock.Anything, mock.Anything, mock.Anything).Return(&CIResult{
					Passed:     false,
					Status:     "failed",
					Output:     "test failed",
					FailedJobs: []string{"test-job"},
				}, nil).Once()

				pg.On("GenerateFixCIPrompt", mock.Anything).Return("fix CI prompt", nil).Times(2)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"fixed\"}\n```",
					ExitCode: 0,
				}, nil).Once()
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"fixed\"}", nil).Once()
				op.On("ParseImplementationSummary", mock.Anything).Return(&ImplementationSummary{Summary: "fixed"}, nil).Once()
				sm.On("SavePhaseOutput", "test-workflow", PhaseImplementation, mock.Anything).Return(nil).Once()

				ci.On("WaitForCIWithProgress", mock.Anything, 0, mock.Anything, mock.Anything, mock.Anything).Return(&CIResult{
					Passed: true,
					Status: "success",
				}, nil).Once()
			},
			wantErr:       false,
			wantNextPhase: PhaseRefactoring,
		},
		{
			name: "fails after exceeding max fix attempts",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				wm.On("CreateWorktree", "test-workflow").Return("/tmp/worktrees/test-workflow", nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)

				pg.On("GenerateImplementationPrompt", mock.Anything).Return("implementation prompt", nil).Once()
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"implemented\"}\n```",
					ExitCode: 0,
				}, nil).Times(2)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"implemented\"}", nil).Times(2)
				op.On("ParseImplementationSummary", mock.Anything).Return(&ImplementationSummary{Summary: "implemented"}, nil).Times(2)
				sm.On("SavePhaseOutput", "test-workflow", PhaseImplementation, mock.Anything).Return(nil).Times(2)

				ci.On("WaitForCIWithProgress", mock.Anything, 0, mock.Anything, mock.Anything, mock.Anything).Return(&CIResult{
					Passed:     false,
					Status:     "failed",
					Output:     "test failed",
					FailedJobs: []string{"test-job"},
				}, nil).Times(2)

				pg.On("GenerateFixCIPrompt", mock.Anything).Return("fix CI prompt", nil).Times(3)
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
		{
			name: "resumes from CI failure state",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)

				pg.On("GenerateFixCIPrompt", "CI check error: build failed").Return("fix CI prompt", nil).Once()
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"fixed\"}\n```",
					ExitCode: 0,
				}, nil).Once()
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"fixed\"}", nil).Once()
				op.On("ParseImplementationSummary", mock.Anything).Return(&ImplementationSummary{Summary: "fixed"}, nil).Once()
				sm.On("SavePhaseOutput", "test-workflow", PhaseImplementation, mock.Anything).Return(nil).Once()

				ci.On("WaitForCIWithProgress", mock.Anything, 0, mock.Anything, mock.Anything, mock.Anything).Return(&CIResult{
					Passed: true,
					Status: "success",
				}, nil).Once()
			},
			wantErr:       false,
			wantNextPhase: PhaseRefactoring,
		},
		{
			name: "fails when GenerateImplementationPrompt fails",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				wm.On("CreateWorktree", "test-workflow").Return("/tmp/worktrees/test-workflow", nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateImplementationPrompt", mock.Anything).Return("", errors.New("failed to generate prompt"))
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
		{
			name: "fails when GenerateFixCIPrompt fails during retry",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				wm.On("CreateWorktree", "test-workflow").Return("/tmp/worktrees/test-workflow", nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)

				pg.On("GenerateImplementationPrompt", mock.Anything).Return("implementation prompt", nil).Once()
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"implemented\"}\n```",
					ExitCode: 0,
				}, nil).Once()
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"implemented\"}", nil).Once()
				op.On("ParseImplementationSummary", mock.Anything).Return(&ImplementationSummary{Summary: "implemented"}, nil).Once()
				sm.On("SavePhaseOutput", "test-workflow", PhaseImplementation, mock.Anything).Return(nil).Once()

				ci.On("WaitForCIWithProgress", mock.Anything, 0, mock.Anything, mock.Anything, mock.Anything).Return(&CIResult{
					Passed:     false,
					Status:     "failed",
					Output:     "test failed",
					FailedJobs: []string{"test-job"},
				}, nil).Once()

				pg.On("GenerateFixCIPrompt", mock.Anything).Return("", errors.New("failed to generate fix prompt"))
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			mockExec := new(MockClaudeExecutor)
			mockPG := new(MockPromptGenerator)
			mockOP := new(MockOutputParser)
			mockCI := new(MockCIChecker)
			mockWM := new(MockWorktreeManager)

			tt.setupMocks(mockSM, mockExec, mockPG, mockOP, mockCI, mockWM)

			config := DefaultConfig("/tmp/workflows")
			config.MaxFixAttempts = 2

			o := &Orchestrator{
				stateManager:    mockSM,
				executor:        mockExec,
				promptGenerator: mockPG,
				parser:          mockOP,
				config:          config,
				worktreeManager: mockWM,
				ciCheckerFactory: func(workingDir string, checkInterval time.Duration, commandTimeout time.Duration) CIChecker {
					return mockCI
				},
			}

			state := &WorkflowState{
				Name:         "test-workflow",
				CurrentPhase: PhaseImplementation,
				WorktreePath: "",
				Phases: map[Phase]*PhaseState{
					PhasePlanning:       {Status: StatusCompleted},
					PhaseConfirmation:   {Status: StatusCompleted},
					PhaseImplementation: {Status: StatusInProgress},
					PhaseRefactoring:    {Status: StatusPending},
					PhasePRSplit:        {Status: StatusPending},
				},
			}

			if tt.name == "resumes from CI failure state" {
				state.WorktreePath = "/existing/worktree/path"
				state.Error = &WorkflowError{
					Message:     "CI check error: build failed",
					Phase:       PhaseImplementation,
					FailureType: FailureTypeCI,
					Recoverable: true,
				}
				state.Phases[PhaseImplementation].Feedback = []string{"CI check error: build failed"}
			}

			err := o.executeImplementation(context.Background(), state)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantNextPhase, state.CurrentPhase)
			mockSM.AssertExpectations(t)
			mockExec.AssertExpectations(t)
			mockPG.AssertExpectations(t)
			mockOP.AssertExpectations(t)
			mockCI.AssertExpectations(t)
			mockWM.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_executeRefactoring_ErrorPaths(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockStateManager, *MockClaudeExecutor, *MockPromptGenerator, *MockOutputParser, *MockCIChecker)
		wantErr    bool
	}{
		{
			name: "fails when LoadPlan fails",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return((*Plan)(nil), errors.New("load plan failed"))
			},
			wantErr: true,
		},
		{
			name: "fails when ExtractJSON fails",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateRefactoringPrompt", mock.Anything).Return("refactoring prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "invalid json output",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("", errors.New("no JSON found"))
				sm.On("WorkflowDir", "test-workflow").Return("/tmp/workflows/test-workflow")
				sm.On("SaveRawOutput", "test-workflow", PhaseRefactoring, mock.Anything).Return(nil)
			},
			wantErr: true,
		},
		{
			name: "fails when ParseRefactoringSummary fails",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateRefactoringPrompt", mock.Anything).Return("refactoring prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"invalid\": \"schema\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"invalid\": \"schema\"}", nil)
				op.On("ParseRefactoringSummary", mock.Anything).Return((*RefactoringSummary)(nil), errors.New("invalid schema"))
				sm.On("WorkflowDir", "test-workflow").Return("/tmp/workflows/test-workflow")
				sm.On("SaveRawOutput", "test-workflow", PhaseRefactoring, mock.Anything).Return(nil)
			},
			wantErr: true,
		},
		{
			name: "fails when SavePhaseOutput fails",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateRefactoringPrompt", mock.Anything).Return("refactoring prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"refactored\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"refactored\"}", nil)
				op.On("ParseRefactoringSummary", mock.Anything).Return(&RefactoringSummary{Summary: "refactored"}, nil)
				sm.On("SavePhaseOutput", "test-workflow", PhaseRefactoring, mock.Anything).Return(errors.New("save failed"))
			},
			wantErr: true,
		},
		{
			name: "fails when CI check fails",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateRefactoringPrompt", mock.Anything).Return("refactoring prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"refactored\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"refactored\"}", nil)
				op.On("ParseRefactoringSummary", mock.Anything).Return(&RefactoringSummary{Summary: "refactored"}, nil)
				sm.On("SavePhaseOutput", "test-workflow", PhaseRefactoring, mock.Anything).Return(nil)
				ci.On("WaitForCIWithProgress", mock.Anything, 0, mock.Anything, mock.Anything, mock.Anything).Return((*CIResult)(nil), errors.New("CI check timeout"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			mockExec := new(MockClaudeExecutor)
			mockPG := new(MockPromptGenerator)
			mockOP := new(MockOutputParser)
			mockCI := new(MockCIChecker)

			tt.setupMocks(mockSM, mockExec, mockPG, mockOP, mockCI)

			o := &Orchestrator{
				stateManager:    mockSM,
				executor:        mockExec,
				promptGenerator: mockPG,
				parser:          mockOP,
				config:          DefaultConfig("/tmp/workflows"),
				ciCheckerFactory: func(workingDir string, checkInterval time.Duration, commandTimeout time.Duration) CIChecker {
					return mockCI
				},
			}

			state := &WorkflowState{
				Name:         "test-workflow",
				CurrentPhase: PhaseRefactoring,
				Phases: map[Phase]*PhaseState{
					PhasePlanning:       {Status: StatusCompleted},
					PhaseConfirmation:   {Status: StatusCompleted},
					PhaseImplementation: {Status: StatusCompleted},
					PhaseRefactoring:    {Status: StatusInProgress},
					PhasePRSplit:        {Status: StatusPending},
				},
			}

			err := o.executeRefactoring(context.Background(), state)

			require.Error(t, err)
			assert.Equal(t, PhaseFailed, state.CurrentPhase)
			mockSM.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_executeRefactoring_CIRetryLoop(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockStateManager, *MockClaudeExecutor, *MockPromptGenerator, *MockOutputParser, *MockCIChecker)
		wantErr       bool
		wantNextPhase Phase
	}{
		// Note: Tests where CI passes after refactoring are skipped because they call getPRMetrics
		// which requires a git repository. See TestOrchestrator_executePhase for the happy path test.
		{
			name: "fails after exceeding max fix attempts",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)

				pg.On("GenerateRefactoringPrompt", mock.Anything).Return("refactoring prompt", nil).Once()
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"refactored\"}\n```",
					ExitCode: 0,
				}, nil).Times(2)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"refactored\"}", nil).Times(2)
				op.On("ParseRefactoringSummary", mock.Anything).Return(&RefactoringSummary{Summary: "refactored"}, nil).Times(2)
				sm.On("SavePhaseOutput", "test-workflow", PhaseRefactoring, mock.Anything).Return(nil).Times(2)

				ci.On("WaitForCIWithProgress", mock.Anything, 0, mock.Anything, mock.Anything, mock.Anything).Return(&CIResult{
					Passed:     false,
					Status:     "failed",
					Output:     "test failed",
					FailedJobs: []string{"test-job"},
				}, nil).Times(2)

				pg.On("GenerateFixCIPrompt", mock.Anything).Return("fix CI prompt", nil).Times(3)
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
		{
			name: "resumes from CI failure state and exceeds max attempts",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)

				// When resuming from CI failure, startAttempt=2, so with MaxFixAttempts=2, it only runs once
				pg.On("GenerateFixCIPrompt", "CI check error: build failed").Return("fix CI prompt", nil).Once()
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"fixed\"}\n```",
					ExitCode: 0,
				}, nil).Once()
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"fixed\"}", nil).Once()
				op.On("ParseRefactoringSummary", mock.Anything).Return(&RefactoringSummary{Summary: "fixed"}, nil).Once()
				sm.On("SavePhaseOutput", "test-workflow", PhaseRefactoring, mock.Anything).Return(nil).Once()

				// CI still fails, and since this is already attempt 2 (max), it should fail
				ci.On("WaitForCIWithProgress", mock.Anything, 0, mock.Anything, mock.Anything, mock.Anything).Return(&CIResult{
					Passed:     false,
					Status:     "failed",
					Output:     "still failing",
					FailedJobs: []string{"test-job"},
				}, nil).Once()

				// GenerateFixCIPrompt called once more before exceeding max attempts
				pg.On("GenerateFixCIPrompt", mock.Anything).Return("fix CI prompt", nil).Once()
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
		{
			name: "fails when GenerateRefactoringPrompt fails",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateRefactoringPrompt", mock.Anything).Return("", errors.New("failed to generate prompt"))
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
		{
			name: "fails when GenerateFixCIPrompt fails during retry",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)

				pg.On("GenerateRefactoringPrompt", mock.Anything).Return("refactoring prompt", nil).Once()
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"refactored\"}\n```",
					ExitCode: 0,
				}, nil).Once()
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"refactored\"}", nil).Once()
				op.On("ParseRefactoringSummary", mock.Anything).Return(&RefactoringSummary{Summary: "refactored"}, nil).Once()
				sm.On("SavePhaseOutput", "test-workflow", PhaseRefactoring, mock.Anything).Return(nil).Once()

				ci.On("WaitForCIWithProgress", mock.Anything, 0, mock.Anything, mock.Anything, mock.Anything).Return(&CIResult{
					Passed:     false,
					Status:     "failed",
					Output:     "test failed",
					FailedJobs: []string{"test-job"},
				}, nil).Once()

				pg.On("GenerateFixCIPrompt", mock.Anything).Return("", errors.New("failed to generate fix prompt"))
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			mockExec := new(MockClaudeExecutor)
			mockPG := new(MockPromptGenerator)
			mockOP := new(MockOutputParser)
			mockCI := new(MockCIChecker)

			tt.setupMocks(mockSM, mockExec, mockPG, mockOP, mockCI)

			config := DefaultConfig("/tmp/workflows")
			config.MaxFixAttempts = 2

			o := &Orchestrator{
				stateManager:    mockSM,
				executor:        mockExec,
				promptGenerator: mockPG,
				parser:          mockOP,
				config:          config,
				ciCheckerFactory: func(workingDir string, checkInterval time.Duration, commandTimeout time.Duration) CIChecker {
					return mockCI
				},
			}

			workDir, _ := os.Getwd()
			state := &WorkflowState{
				Name:         "test-workflow",
				CurrentPhase: PhaseRefactoring,
				WorktreePath: workDir,
				Phases: map[Phase]*PhaseState{
					PhasePlanning:       {Status: StatusCompleted},
					PhaseConfirmation:   {Status: StatusCompleted},
					PhaseImplementation: {Status: StatusCompleted},
					PhaseRefactoring:    {Status: StatusInProgress},
					PhasePRSplit:        {Status: StatusPending},
				},
			}

			if tt.name == "resumes from CI failure state and exceeds max attempts" {
				state.Error = &WorkflowError{
					Message:     "CI check error: build failed",
					Phase:       PhaseRefactoring,
					FailureType: FailureTypeCI,
					Recoverable: true,
				}
				state.Phases[PhaseRefactoring].Feedback = []string{"CI check error: build failed"}
			}

			err := o.executeRefactoring(context.Background(), state)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantNextPhase, state.CurrentPhase)
			mockSM.AssertExpectations(t)
			mockExec.AssertExpectations(t)
			mockPG.AssertExpectations(t)
			mockOP.AssertExpectations(t)
			mockCI.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_executePhase(t *testing.T) {
	tests := []struct {
		name       string
		phase      Phase
		setupMocks func(*MockStateManager, *MockClaudeExecutor, *MockPromptGenerator, *MockOutputParser, *MockCIChecker, *MockWorktreeManager)
		wantErr    bool
	}{
		{
			name:  "executes PhasePlanning",
			phase: PhasePlanning,
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				pg.On("GeneratePlanningPrompt", WorkflowTypeFeature, "test description", []string(nil)).Return("planning prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"test plan\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"test plan\"}", nil)
				op.On("ParsePlan", mock.Anything).Return(&Plan{Summary: "test plan"}, nil)
				sm.On("SavePlan", "test-workflow", mock.Anything).Return(nil)
				sm.On("SavePlanMarkdown", "test-workflow", mock.Anything).Return(nil)
				sm.On("SavePhaseOutput", "test-workflow", PhasePlanning, mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "executes PhaseConfirmation",
			phase: PhaseConfirmation,
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil).Times(2)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
			},
			wantErr: false,
		},
		{
			name:  "executes PhaseImplementation",
			phase: PhaseImplementation,
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				wm.On("CreateWorktree", "test-workflow").Return("/tmp/worktrees/test-workflow", nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateImplementationPrompt", mock.Anything).Return("implementation prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"implemented\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"implemented\"}", nil)
				op.On("ParseImplementationSummary", mock.Anything).Return(&ImplementationSummary{Summary: "implemented"}, nil)
				sm.On("SavePhaseOutput", "test-workflow", PhaseImplementation, mock.Anything).Return(nil)
				ci.On("WaitForCIWithProgress", mock.Anything, 0, mock.Anything, mock.Anything, mock.Anything).Return(&CIResult{Passed: true, Status: "success"}, nil)
			},
			wantErr: false,
		},
		// Note: PhaseRefactoring test is in TestOrchestrator_executeRefactoring since it requires git repo setup
		{
			name:  "executes PhasePRSplit",
			phase: PhasePRSplit,
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker, wm *MockWorktreeManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				pg.On("GeneratePRSplitPrompt", mock.Anything).Return("pr split prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"parent_pr\":{\"number\":1}}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"parent_pr\":{\"number\":1}}", nil)
				op.On("ParsePRSplitResult", mock.Anything).Return(&PRSplitResult{ParentPR: PRInfo{Number: 1}}, nil)
				sm.On("SavePhaseOutput", "test-workflow", PhasePRSplit, mock.Anything).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			mockExec := new(MockClaudeExecutor)
			mockPG := new(MockPromptGenerator)
			mockOP := new(MockOutputParser)
			mockCI := new(MockCIChecker)
			mockWM := new(MockWorktreeManager)

			tt.setupMocks(mockSM, mockExec, mockPG, mockOP, mockCI, mockWM)

			o := &Orchestrator{
				stateManager:    mockSM,
				executor:        mockExec,
				promptGenerator: mockPG,
				parser:          mockOP,
				config:          DefaultConfig("/tmp/workflows"),
				worktreeManager: mockWM,
				confirmFunc: func(plan *Plan) (bool, string, error) {
					return true, "", nil
				},
				ciCheckerFactory: func(workingDir string, checkInterval time.Duration, commandTimeout time.Duration) CIChecker {
					return mockCI
				},
			}

			state := &WorkflowState{
				Name:         "test-workflow",
				Type:         WorkflowTypeFeature,
				Description:  "test description",
				CurrentPhase: tt.phase,
				Phases: map[Phase]*PhaseState{
					PhasePlanning:       {Status: StatusCompleted},
					PhaseConfirmation:   {Status: StatusCompleted},
					PhaseImplementation: {Status: StatusCompleted},
					PhaseRefactoring:    {Status: StatusCompleted},
					PhasePRSplit:        {Status: StatusInProgress, Metrics: &PRMetrics{FilesChanged: 10, LinesChanged: 100}},
				},
			}

			err := o.executePhase(context.Background(), state)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			mockSM.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_executePhase_InvalidPhase(t *testing.T) {
	mockSM := new(MockStateManager)
	mockSM.On("SaveState", "test-workflow", mock.Anything).Return(nil)

	o := &Orchestrator{
		stateManager: mockSM,
		config:       DefaultConfig("/tmp/workflows"),
	}

	state := &WorkflowState{
		Name:         "test-workflow",
		CurrentPhase: "INVALID_PHASE",
		Phases: map[Phase]*PhaseState{
			"INVALID_PHASE": {Status: StatusInProgress},
		},
	}

	err := o.executePhase(context.Background(), state)

	require.Error(t, err)
	assert.Equal(t, PhaseFailed, state.CurrentPhase)
	mockSM.AssertExpectations(t)
}

func TestOrchestrator_Start(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockStateManager)
		wantErr    bool
	}{
		{
			name: "fails when InitState fails",
			setupMocks: func(sm *MockStateManager) {
				sm.On("WorkflowExists", "test-workflow").Return(false)
				sm.On("InitState", "test-workflow", "test description", WorkflowTypeFeature).Return((*WorkflowState)(nil), errors.New("init failed"))
			},
			wantErr: true,
		},
		{
			name: "deletes and restarts failed workflow",
			setupMocks: func(sm *MockStateManager) {
				sm.On("WorkflowExists", "test-workflow").Return(true)
				sm.On("LoadState", "test-workflow").Return(&WorkflowState{
					Name:         "test-workflow",
					CurrentPhase: PhaseFailed,
				}, nil)
				sm.On("DeleteWorkflow", "test-workflow").Return(nil)
				sm.On("InitState", "test-workflow", "test description", WorkflowTypeFeature).Return((*WorkflowState)(nil), errors.New("init failed"))
			},
			wantErr: true,
		},
		{
			name: "fails when workflow exists and not failed",
			setupMocks: func(sm *MockStateManager) {
				sm.On("WorkflowExists", "test-workflow").Return(true)
				sm.On("LoadState", "test-workflow").Return(&WorkflowState{
					Name:         "test-workflow",
					CurrentPhase: PhaseImplementation,
				}, nil)
				sm.On("InitState", "test-workflow", "test description", WorkflowTypeFeature).Return((*WorkflowState)(nil), ErrWorkflowExists)
			},
			wantErr: true,
		},
		{
			name: "fails when deleting failed workflow fails",
			setupMocks: func(sm *MockStateManager) {
				sm.On("WorkflowExists", "test-workflow").Return(true)
				sm.On("LoadState", "test-workflow").Return(&WorkflowState{
					Name:         "test-workflow",
					CurrentPhase: PhaseFailed,
				}, nil)
				sm.On("DeleteWorkflow", "test-workflow").Return(errors.New("delete failed"))
			},
			wantErr: true,
		},
		{
			name: "continues when LoadState fails for existing workflow",
			setupMocks: func(sm *MockStateManager) {
				sm.On("WorkflowExists", "test-workflow").Return(true)
				sm.On("LoadState", "test-workflow").Return((*WorkflowState)(nil), errors.New("load failed"))
				sm.On("InitState", "test-workflow", "test description", WorkflowTypeFeature).Return((*WorkflowState)(nil), ErrWorkflowExists)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			tt.setupMocks(mockSM)

			o := &Orchestrator{
				stateManager: mockSM,
				config:       DefaultConfig("/tmp/workflows"),
			}

			err := o.Start(context.Background(), "test-workflow", "test description", WorkflowTypeFeature)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			mockSM.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_transitionPhase(t *testing.T) {
	tests := []struct {
		name         string
		currentPhase Phase
		nextPhase    Phase
		setupMocks   func(*MockStateManager)
		wantErr      bool
	}{
		{
			name:         "transitions from planning to confirmation",
			currentPhase: PhasePlanning,
			nextPhase:    PhaseConfirmation,
			setupMocks: func(sm *MockStateManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name:         "transitions to completed",
			currentPhase: PhaseRefactoring,
			nextPhase:    PhaseCompleted,
			setupMocks: func(sm *MockStateManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name:         "returns error when SaveState fails",
			currentPhase: PhasePlanning,
			nextPhase:    PhaseConfirmation,
			setupMocks: func(sm *MockStateManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(errors.New("save failed"))
			},
			wantErr: true,
		},
		{
			name:         "transitions to failed",
			currentPhase: PhasePlanning,
			nextPhase:    PhaseFailed,
			setupMocks: func(sm *MockStateManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			tt.setupMocks(mockSM)

			o := &Orchestrator{
				stateManager: mockSM,
				config:       DefaultConfig("/tmp/workflows"),
			}

			state := &WorkflowState{
				Name:         "test-workflow",
				CurrentPhase: tt.currentPhase,
				Phases: map[Phase]*PhaseState{
					PhasePlanning:       {Status: StatusInProgress},
					PhaseConfirmation:   {Status: StatusPending},
					PhaseImplementation: {Status: StatusPending},
					PhaseRefactoring:    {Status: StatusPending},
					PhasePRSplit:        {Status: StatusPending},
				},
			}

			err := o.transitionPhase(state, tt.nextPhase)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.nextPhase, state.CurrentPhase)
			assert.Equal(t, StatusCompleted, state.Phases[tt.currentPhase].Status)

			if tt.nextPhase != PhaseCompleted && tt.nextPhase != PhaseFailed {
				assert.Equal(t, StatusInProgress, state.Phases[tt.nextPhase].Status)
			}

			mockSM.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_Resume(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockStateManager)
		wantErr    bool
		errMsg     string
	}{
		{
			name: "cannot resume completed workflow",
			setupMocks: func(sm *MockStateManager) {
				sm.On("LoadState", "test-workflow").Return(&WorkflowState{
					Name:         "test-workflow",
					CurrentPhase: PhaseCompleted,
				}, nil)
			},
			wantErr: true,
			errMsg:  "already completed",
		},
		{
			name: "cannot resume non-recoverable error",
			setupMocks: func(sm *MockStateManager) {
				sm.On("LoadState", "test-workflow").Return(&WorkflowState{
					Name:         "test-workflow",
					CurrentPhase: PhaseFailed,
					Error: &WorkflowError{
						Message:     "non-recoverable error",
						Recoverable: false,
					},
				}, nil)
			},
			wantErr: true,
			errMsg:  "non-recoverable error state",
		},
		{
			name: "fails when LoadState fails",
			setupMocks: func(sm *MockStateManager) {
				sm.On("LoadState", "test-workflow").Return((*WorkflowState)(nil), errors.New("load failed"))
			},
			wantErr: true,
			errMsg:  "failed to load workflow state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			tt.setupMocks(mockSM)

			o := &Orchestrator{
				stateManager: mockSM,
				config:       DefaultConfig("/tmp/workflows"),
			}

			err := o.Resume(context.Background(), "test-workflow")

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}

			require.NoError(t, err)
			mockSM.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_Resume_RestoresFailedPhase(t *testing.T) {
	tests := []struct {
		name                string
		initialState        *WorkflowState
		expectedPhase       Phase
		expectedPhaseStatus PhaseStatus
	}{
		{
			name: "restores phase from error.Phase when error exists",
			initialState: &WorkflowState{
				Name:         "test-workflow",
				CurrentPhase: PhaseFailed,
				Phases: map[Phase]*PhaseState{
					PhaseImplementation: {Status: StatusFailed},
					PhasePlanning:       {Status: StatusCompleted},
				},
				Error: &WorkflowError{
					Message:     "parse error",
					Phase:       PhaseImplementation,
					Recoverable: true,
				},
			},
			expectedPhase:       PhaseImplementation,
			expectedPhaseStatus: StatusInProgress,
		},
		{
			name: "finds failed phase when error is nil",
			initialState: &WorkflowState{
				Name:         "test-workflow",
				CurrentPhase: PhaseFailed,
				Phases: map[Phase]*PhaseState{
					PhaseImplementation: {Status: StatusFailed},
					PhasePlanning:       {Status: StatusCompleted},
				},
			},
			expectedPhase:       PhaseImplementation,
			expectedPhaseStatus: StatusInProgress,
		},
		{
			name: "finds in_progress phase when error is nil",
			initialState: &WorkflowState{
				Name:         "test-workflow",
				CurrentPhase: PhaseFailed,
				Phases: map[Phase]*PhaseState{
					PhaseRefactoring: {Status: StatusInProgress},
					PhasePlanning:    {Status: StatusCompleted},
				},
			},
			expectedPhase:       PhaseRefactoring,
			expectedPhaseStatus: StatusInProgress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			mockSM.On("LoadState", "test-workflow").Return(tt.initialState, nil)

			// Capture the saved state to verify
			var savedState *WorkflowState
			mockSM.On("SaveState", "test-workflow", mock.Anything).Run(func(args mock.Arguments) {
				savedState = args.Get(1).(*WorkflowState)
			}).Return(errors.New("stop execution for test"))

			o := &Orchestrator{
				stateManager: mockSM,
				config:       DefaultConfig("/tmp/workflows"),
			}

			// Resume will fail because SaveState returns error, but we verify state was correctly set
			err := o.Resume(context.Background(), "test-workflow")
			require.Error(t, err)

			// Verify the state was correctly modified before save
			assert.Equal(t, tt.expectedPhase, savedState.CurrentPhase)
			assert.Nil(t, savedState.Error)
			assert.Equal(t, tt.expectedPhaseStatus, savedState.Phases[tt.expectedPhase].Status)

			mockSM.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_List(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockStateManager)
		want       []WorkflowInfo
		wantErr    bool
	}{
		{
			name: "successfully lists workflows",
			setupMocks: func(sm *MockStateManager) {
				workflows := []WorkflowInfo{
					{
						Name:         "workflow-1",
						Type:         WorkflowTypeFeature,
						CurrentPhase: PhasePlanning,
						Status:       "in_progress",
					},
				}
				sm.On("ListWorkflows").Return(workflows, nil)
			},
			want: []WorkflowInfo{
				{
					Name:         "workflow-1",
					Type:         WorkflowTypeFeature,
					CurrentPhase: PhasePlanning,
					Status:       "in_progress",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			tt.setupMocks(mockSM)

			o := &Orchestrator{
				stateManager: mockSM,
			}

			got, err := o.List()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			mockSM.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_Clean(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockStateManager)
		want       []string
		wantErr    bool
	}{
		{
			name: "successfully cleans completed workflows",
			setupMocks: func(sm *MockStateManager) {
				workflows := []WorkflowInfo{
					{Name: "workflow-1", Status: "completed"},
					{Name: "workflow-2", Status: "in_progress"},
					{Name: "workflow-3", Status: "completed"},
				}
				sm.On("ListWorkflows").Return(workflows, nil)
				sm.On("DeleteWorkflow", "workflow-1").Return(nil)
				sm.On("DeleteWorkflow", "workflow-3").Return(nil)
			},
			want:    []string{"workflow-1", "workflow-3"},
			wantErr: false,
		},
		{
			name: "returns error when ListWorkflows fails",
			setupMocks: func(sm *MockStateManager) {
				sm.On("ListWorkflows").Return([]WorkflowInfo(nil), errors.New("list failed"))
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "continues on delete error and deletes other workflows",
			setupMocks: func(sm *MockStateManager) {
				workflows := []WorkflowInfo{
					{Name: "workflow-1", Status: "completed"},
					{Name: "workflow-2", Status: "completed"},
				}
				sm.On("ListWorkflows").Return(workflows, nil)
				sm.On("DeleteWorkflow", "workflow-1").Return(errors.New("delete failed"))
				sm.On("DeleteWorkflow", "workflow-2").Return(nil)
			},
			want:    []string{"workflow-2"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			tt.setupMocks(mockSM)

			o := &Orchestrator{
				stateManager: mockSM,
			}

			got, err := o.Clean()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			mockSM.AssertExpectations(t)
		})
	}
}

func TestIsRecoverableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error is not recoverable",
			err:  nil,
			want: false,
		},
		{
			name: "timeout error is recoverable",
			err:  errors.New("operation timeout"),
			want: true,
		},
		{
			name: "connection timeout is recoverable",
			err:  errors.New("connection timeout after 30s"),
			want: true,
		},
		{
			name: "claude execution timeout is recoverable",
			err:  errors.New("claude execution timeout"),
			want: true,
		},
		{
			name: "claude execution failed is recoverable",
			err:  errors.New("claude execution failed"),
			want: true,
		},
		{
			name: "claude execution failed with exit code is recoverable",
			err:  errors.New("claude execution failed with exit code 1"),
			want: true,
		},
		{
			name: "failed to parse JSON is recoverable",
			err:  errors.New("failed to parse JSON"),
			want: true,
		},
		{
			name: "failed to parse YAML is recoverable",
			err:  errors.New("failed to parse YAML"),
			want: true,
		},
		{
			name: "failed to parse response is recoverable",
			err:  errors.New("failed to parse response"),
			want: true,
		},
		{
			name: "invalid workflow name is not recoverable",
			err:  errors.New("invalid workflow name"),
			want: false,
		},
		{
			name: "invalid phase is not recoverable",
			err:  errors.New("invalid phase transition"),
			want: false,
		},
		{
			name: "invalid configuration is not recoverable",
			err:  errors.New("invalid configuration"),
			want: false,
		},
		{
			name: "invalid input is not recoverable",
			err:  errors.New("invalid input parameter"),
			want: false,
		},
		{
			name: "generic error is recoverable by default",
			err:  errors.New("something went wrong"),
			want: true,
		},
		{
			name: "network error is recoverable by default",
			err:  errors.New("network connection lost"),
			want: true,
		},
		{
			name: "temporary error is recoverable by default",
			err:  errors.New("temporary failure, please retry"),
			want: true,
		},
		{
			name: "timeout at start is recoverable",
			err:  errors.New("timeout at connection start"),
			want: true,
		},
		{
			name: "timeout at end is recoverable",
			err:  errors.New("operation ended with timeout"),
			want: true,
		},
		{
			name: "timeout in middle is recoverable",
			err:  errors.New("operation timeout during execution"),
			want: true,
		},
		{
			name: "context deadline exceeded is recoverable",
			err:  errors.New("context deadline exceeded timeout"),
			want: true,
		},
		{
			name: "invalid at start of message is not recoverable",
			err:  errors.New("invalid request parameters"),
			want: false,
		},
		{
			name: "invalid in middle of message is not recoverable",
			err:  errors.New("request has invalid data"),
			want: false,
		},
		{
			name: "invalid workflow name with context is not recoverable",
			err:  errors.New("invalid workflow name: cannot contain spaces"),
			want: false,
		},
		{
			name: "parse error in JSON is recoverable",
			err:  errors.New("failed to parse JSON response from API"),
			want: true,
		},
		{
			name: "parse error with details is recoverable",
			err:  errors.New("failed to parse output: unexpected character at position 42"),
			want: true,
		},
		{
			name: "claude execution failed with details is recoverable",
			err:  errors.New("claude execution failed: connection reset by peer"),
			want: true,
		},
		{
			name: "claude execution failed with status code is recoverable",
			err:  errors.New("claude execution failed: HTTP 503 Service Unavailable"),
			want: true,
		},
		{
			name: "database error is recoverable by default",
			err:  errors.New("database connection failed"),
			want: true,
		},
		{
			name: "file not found is recoverable by default",
			err:  errors.New("file not found: config.yaml"),
			want: true,
		},
		{
			name: "permission denied is recoverable by default",
			err:  errors.New("permission denied accessing file"),
			want: true,
		},
		{
			name: "invalid type is not recoverable",
			err:  errors.New("invalid type provided"),
			want: false,
		},
		{
			name: "invalid format is not recoverable",
			err:  errors.New("invalid format specified"),
			want: false,
		},
		{
			name: "invalid state is not recoverable",
			err:  errors.New("invalid state transition requested"),
			want: false,
		},
		{
			name: "disk full error is recoverable by default",
			err:  errors.New("no space left on device"),
			want: true,
		},
		{
			name: "out of memory error is recoverable by default",
			err:  errors.New("out of memory"),
			want: true,
		},
		{
			name: "rate limit error is recoverable by default",
			err:  errors.New("rate limit exceeded, retry after 60s"),
			want: true,
		},
		{
			name: "service unavailable is recoverable by default",
			err:  errors.New("service temporarily unavailable"),
			want: true,
		},
		{
			name: "error with timeout word capitalized is recoverable",
			err:  errors.New("Request Timeout occurred"),
			want: true,
		},
		{
			name: "error with parse word capitalized is recoverable",
			err:  errors.New("Failed to Parse the response"),
			want: true,
		},
		{
			name: "error with Invalid word capitalized is recoverable since check is case-sensitive",
			err:  errors.New("Invalid configuration detected"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRecoverableError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	baseDir := "/tmp/workflows"
	config := DefaultConfig(baseDir)

	assert.Equal(t, baseDir, config.BaseDir)
	assert.Equal(t, 100, config.MaxLines)
	assert.Equal(t, 10, config.MaxFiles)
	assert.Equal(t, "claude", config.ClaudePath)
	assert.Equal(t, 1*time.Hour, config.Timeouts.Planning)
	assert.Equal(t, 6*time.Hour, config.Timeouts.Implementation)
	assert.Equal(t, 6*time.Hour, config.Timeouts.Refactoring)
	assert.Equal(t, 1*time.Hour, config.Timeouts.PRSplit)
}

func TestOrchestrator_SetConfirmFunc(t *testing.T) {
	o := &Orchestrator{}
	customFunc := func(plan *Plan) (bool, string, error) {
		return true, "", nil
	}

	o.SetConfirmFunc(customFunc)
	assert.NotNil(t, o.confirmFunc)
}

func TestParseDiffStat(t *testing.T) {
	tests := []struct {
		name       string
		diffOutput string
		want       *PRMetrics
		wantErr    bool
	}{
		{
			name: "single file changed",
			diffOutput: ` main.go | 10 ++++++++++
 1 file changed, 10 insertions(+)`,
			want: &PRMetrics{
				LinesChanged:  10,
				FilesChanged:  1,
				FilesAdded:    []string{},
				FilesModified: []string{"main.go"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "multiple files changed",
			diffOutput: ` file1.go | 10 ++++++++++
 file2.go | 5 +++++
 file3.go | 3 +++
 3 files changed, 18 insertions(+)`,
			want: &PRMetrics{
				LinesChanged:  18,
				FilesChanged:  3,
				FilesAdded:    []string{},
				FilesModified: []string{"file1.go", "file2.go", "file3.go"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name:       "no changes",
			diffOutput: "",
			want: &PRMetrics{
				FilesAdded:    []string{},
				FilesModified: []string{},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "new file added",
			diffOutput: ` newfile.go (new) | 20 ++++++++++++++++++++
 1 file changed, 20 insertions(+)`,
			want: &PRMetrics{
				LinesChanged:  20,
				FilesChanged:  1,
				FilesAdded:    []string{"newfile.go"},
				FilesModified: []string{},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "file deleted",
			diffOutput: ` oldfile.go (gone) | 50 --------------------------------------------------
 1 file changed, 50 deletions(-)`,
			want: &PRMetrics{
				LinesChanged:  50,
				FilesChanged:  1,
				FilesAdded:    []string{},
				FilesModified: []string{},
				FilesDeleted:  []string{"oldfile.go"},
			},
			wantErr: false,
		},
		{
			name: "mixed changes with additions modifications and deletions",
			diffOutput: ` newfile.go (new) | 20 ++++++++++++++++++++
 existing.go | 10 ++++++++++
 oldfile.go (gone) | 15 ---------------
 3 files changed, 30 insertions(+), 15 deletions(-)`,
			want: &PRMetrics{
				LinesChanged:  30,
				FilesChanged:  3,
				FilesAdded:    []string{"newfile.go"},
				FilesModified: []string{"existing.go"},
				FilesDeleted:  []string{"oldfile.go"},
			},
			wantErr: false,
		},
		{
			name: "files with paths",
			diffOutput: ` internal/workflow/orchestrator.go | 25 +++++++++++++++++++
 cmd/main.go | 5 +++++
 2 files changed, 30 insertions(+)`,
			want: &PRMetrics{
				LinesChanged:  30,
				FilesChanged:  2,
				FilesAdded:    []string{},
				FilesModified: []string{"internal/workflow/orchestrator.go", "cmd/main.go"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "binary files",
			diffOutput: ` image.png | Bin 0 -> 1024 bytes
 data.bin  | Bin 2048 -> 4096 bytes
 2 files changed`,
			want: &PRMetrics{
				LinesChanged:  0,
				FilesChanged:  2,
				FilesAdded:    []string{},
				FilesModified: []string{"image.png", "data.bin"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "renamed files",
			diffOutput: ` old.go => new.go | 5 +++++
 1 file changed, 5 insertions(+)`,
			want: &PRMetrics{
				LinesChanged:  5,
				FilesChanged:  1,
				FilesAdded:    []string{},
				FilesModified: []string{"old.go"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "large number of changes",
			diffOutput: ` file1.go | 100 ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 file2.go | 50 +++++++++++++++++++++++++++++
 2 files changed, 150 insertions(+)`,
			want: &PRMetrics{
				LinesChanged:  150,
				FilesChanged:  2,
				FilesAdded:    []string{},
				FilesModified: []string{"file1.go", "file2.go"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "deletions only",
			diffOutput: ` file1.go | 10 ----------
 file2.go | 5 -----
 2 files changed, 15 deletions(-)`,
			want: &PRMetrics{
				LinesChanged:  15,
				FilesChanged:  2,
				FilesAdded:    []string{},
				FilesModified: []string{"file1.go", "file2.go"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "insertions and deletions combined",
			diffOutput: ` file1.go | 25 +++++++++++--------------
 1 file changed, 12 insertions(+), 13 deletions(-)`,
			want: &PRMetrics{
				LinesChanged:  12,
				FilesChanged:  1,
				FilesAdded:    []string{},
				FilesModified: []string{"file1.go"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "whitespace-only line",
			diffOutput: ` file1.go | 10 ++++++++++

 1 file changed, 10 insertions(+)`,
			want: &PRMetrics{
				LinesChanged:  10,
				FilesChanged:  1,
				FilesAdded:    []string{},
				FilesModified: []string{"file1.go"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name:       "only summary line no files",
			diffOutput: ` 0 files changed`,
			want: &PRMetrics{
				LinesChanged:  0,
				FilesChanged:  0,
				FilesAdded:    []string{},
				FilesModified: []string{},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "file with very long path",
			diffOutput: ` internal/very/deep/nested/path/to/some/file/in/the/project/structure/file.go | 5 +++++
 1 file changed, 5 insertions(+)`,
			want: &PRMetrics{
				LinesChanged:  5,
				FilesChanged:  1,
				FilesAdded:    []string{},
				FilesModified: []string{"internal/very/deep/nested/path/to/some/file/in/the/project/structure/file.go"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "file with dots in name",
			diffOutput: ` test.config.json | 3 +++
 1 file changed, 3 insertions(+)`,
			want: &PRMetrics{
				LinesChanged:  3,
				FilesChanged:  1,
				FilesAdded:    []string{},
				FilesModified: []string{"test.config.json"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "file with numbers in name",
			diffOutput: ` migration_001_initial.sql | 20 ++++++++++++++++++++
 1 file changed, 20 insertions(+)`,
			want: &PRMetrics{
				LinesChanged:  20,
				FilesChanged:  1,
				FilesAdded:    []string{},
				FilesModified: []string{"migration_001_initial.sql"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "summary with only files changed field",
			diffOutput: ` file1.go | 10 ++++++++++
 1 file changed`,
			want: &PRMetrics{
				LinesChanged:  0,
				FilesChanged:  1,
				FilesAdded:    []string{},
				FilesModified: []string{"file1.go"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "mixed new deleted and modified in complex scenario",
			diffOutput: ` new1.go (new) | 30 ++++++++++++++++++++++++++++++
 new2.go (new) | 15 +++++++++++++++
 existing1.go | 10 ++++++++++
 existing2.go | 5 -----
 old1.go (gone) | 25 -------------------------
 old2.go (gone) | 10 ----------
 6 files changed, 55 insertions(+), 40 deletions(-)`,
			want: &PRMetrics{
				LinesChanged:  55,
				FilesChanged:  6,
				FilesAdded:    []string{"new1.go", "new2.go"},
				FilesModified: []string{"existing1.go", "existing2.go"},
				FilesDeleted:  []string{"old1.go", "old2.go"},
			},
			wantErr: false,
		},
		{
			name: "trailing whitespace in diff output",
			diffOutput: ` file1.go | 10 ++++++++++
 1 file changed, 10 insertions(+)   `,
			want: &PRMetrics{
				LinesChanged:  10,
				FilesChanged:  1,
				FilesAdded:    []string{},
				FilesModified: []string{"file1.go"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name:       "single line output with no newline",
			diffOutput: ` 0 files changed`,
			want: &PRMetrics{
				LinesChanged:  0,
				FilesChanged:  0,
				FilesAdded:    []string{},
				FilesModified: []string{},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "summary line with non-numeric parts",
			diffOutput: ` file1.go | 10 ++++++++++
 abc files changed, def insertions(+)`,
			want: &PRMetrics{
				LinesChanged:  0,
				FilesChanged:  0,
				FilesAdded:    []string{},
				FilesModified: []string{"file1.go"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "summary with only 2 parts",
			diffOutput: ` file1.go | 10 ++++++++++
 1 file`,
			want: &PRMetrics{
				LinesChanged:  0,
				FilesChanged:  1,
				FilesAdded:    []string{},
				FilesModified: []string{"file1.go"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDiffStat(tt.diffOutput)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOrchestrator_executeRefactoring(t *testing.T) {
	tests := []struct {
		name            string
		initialWorktree string
		setupMocks      func(*MockStateManager, *MockClaudeExecutor, *MockPromptGenerator, *MockOutputParser, *MockCIChecker)
		wantErr         bool
		wantNextPhase   Phase
	}{
		{
			name:            "fails when executor fails",
			initialWorktree: "/tmp/worktrees/test-workflow",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateRefactoringPrompt", mock.Anything).Return("refactoring prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return((*ExecuteResult)(nil), errors.New("execution failed"))
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
		{
			name:            "fails when LoadPlan fails",
			initialWorktree: "/tmp/worktrees/test-workflow",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return((*Plan)(nil), errors.New("plan not found"))
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
		{
			name:            "fails when GenerateRefactoringPrompt fails",
			initialWorktree: "/tmp/worktrees/test-workflow",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateRefactoringPrompt", mock.Anything).Return("", errors.New("prompt generation failed"))
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
		{
			name:            "fails when ExtractJSON fails",
			initialWorktree: "/tmp/worktrees/test-workflow",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateRefactoringPrompt", mock.Anything).Return("refactoring prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "invalid output",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("", errors.New("no JSON found"))
				sm.On("SaveRawOutput", "test-workflow", PhaseRefactoring, "invalid output").Return(nil)
				sm.On("WorkflowDir", "test-workflow").Return("/tmp/workflows/test-workflow")
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
		{
			name:            "fails when ParseRefactoringSummary fails",
			initialWorktree: "/tmp/worktrees/test-workflow",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateRefactoringPrompt", mock.Anything).Return("refactoring prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"invalid\": true}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"invalid\": true}", nil)
				op.On("ParseRefactoringSummary", mock.Anything).Return((*RefactoringSummary)(nil), errors.New("invalid summary"))
				sm.On("SaveRawOutput", "test-workflow", PhaseRefactoring, "```json\n{\"invalid\": true}\n```").Return(nil)
				sm.On("WorkflowDir", "test-workflow").Return("/tmp/workflows/test-workflow")
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
		{
			name:            "fails when SavePhaseOutput fails",
			initialWorktree: "/tmp/worktrees/test-workflow",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateRefactoringPrompt", mock.Anything).Return("refactoring prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"refactored\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"refactored\"}", nil)
				op.On("ParseRefactoringSummary", mock.Anything).Return(&RefactoringSummary{Summary: "refactored"}, nil)
				sm.On("SavePhaseOutput", "test-workflow", PhaseRefactoring, mock.Anything).Return(errors.New("failed to save"))
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
		{
			name:            "fails when CI check fails",
			initialWorktree: "/tmp/worktrees/test-workflow",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser, ci *MockCIChecker) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateRefactoringPrompt", mock.Anything).Return("refactoring prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"refactored\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"refactored\"}", nil)
				op.On("ParseRefactoringSummary", mock.Anything).Return(&RefactoringSummary{Summary: "refactored"}, nil)
				sm.On("SavePhaseOutput", "test-workflow", PhaseRefactoring, mock.Anything).Return(nil)
				ci.On("WaitForCIWithProgress", mock.Anything, 0, mock.Anything, mock.Anything, mock.Anything).Return((*CIResult)(nil), errors.New("CI check error"))
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			mockExec := new(MockClaudeExecutor)
			mockPG := new(MockPromptGenerator)
			mockOP := new(MockOutputParser)
			mockCI := new(MockCIChecker)

			tt.setupMocks(mockSM, mockExec, mockPG, mockOP, mockCI)

			o := &Orchestrator{
				stateManager:    mockSM,
				executor:        mockExec,
				promptGenerator: mockPG,
				parser:          mockOP,
				config:          DefaultConfig("/tmp/workflows"),
				ciCheckerFactory: func(workingDir string, checkInterval time.Duration, commandTimeout time.Duration) CIChecker {
					return mockCI
				},
			}

			state := &WorkflowState{
				Name:         "test-workflow",
				CurrentPhase: PhaseRefactoring,
				WorktreePath: tt.initialWorktree,
				Phases: map[Phase]*PhaseState{
					PhasePlanning:       {Status: StatusCompleted},
					PhaseConfirmation:   {Status: StatusCompleted},
					PhaseImplementation: {Status: StatusCompleted},
					PhaseRefactoring:    {Status: StatusInProgress},
					PhasePRSplit:        {Status: StatusPending},
				},
			}

			err := o.executeRefactoring(context.Background(), state)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantNextPhase, state.CurrentPhase)
			mockSM.AssertExpectations(t)
			mockExec.AssertExpectations(t)
			mockPG.AssertExpectations(t)
			mockOP.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_executePRSplit(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockStateManager, *MockClaudeExecutor, *MockPromptGenerator, *MockOutputParser)
		wantErr       bool
		wantNextPhase Phase
	}{
		{
			name: "successfully splits PR",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				pg.On("GeneratePRSplitPrompt", mock.Anything).Return("pr-split prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"split complete\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"split complete\"}", nil)
				op.On("ParsePRSplitResult", mock.Anything).Return(&PRSplitResult{Summary: "split complete"}, nil)
				sm.On("SavePhaseOutput", "test-workflow", PhasePRSplit, mock.Anything).Return(nil)
			},
			wantErr:       false,
			wantNextPhase: PhaseCompleted,
		},
		{
			name: "fails when metrics not available",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
			},
			wantErr:       true,
			wantNextPhase: PhaseFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			mockExec := new(MockClaudeExecutor)
			mockPG := new(MockPromptGenerator)
			mockOP := new(MockOutputParser)

			tt.setupMocks(mockSM, mockExec, mockPG, mockOP)

			o := &Orchestrator{
				stateManager:    mockSM,
				executor:        mockExec,
				promptGenerator: mockPG,
				parser:          mockOP,
				config:          DefaultConfig("/tmp/workflows"),
			}

			metrics := &PRMetrics{
				LinesChanged: 150,
				FilesChanged: 15,
			}

			var metricsPtr *PRMetrics
			if tt.name == "successfully splits PR" {
				metricsPtr = metrics
			}

			state := &WorkflowState{
				Name:         "test-workflow",
				CurrentPhase: PhasePRSplit,
				Phases: map[Phase]*PhaseState{
					PhasePlanning:       {Status: StatusCompleted},
					PhaseConfirmation:   {Status: StatusCompleted},
					PhaseImplementation: {Status: StatusCompleted},
					PhaseRefactoring:    {Status: StatusCompleted},
					PhasePRSplit:        {Status: StatusInProgress, Metrics: metricsPtr},
				},
			}

			err := o.executePRSplit(context.Background(), state)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantNextPhase, state.CurrentPhase)
			mockSM.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_executePRSplit_CIFailureRetry(t *testing.T) {
	// This test verifies that when CI fails on a child PR, the retry uses
	// GenerateFixCIPrompt to generate a proper fix prompt (not raw error text)
	mockSM := new(MockStateManager)
	mockExec := new(MockClaudeExecutor)
	mockPG := new(MockPromptGenerator)
	mockOP := new(MockOutputParser)
	mockCI := new(MockCIChecker)

	// Setup: first attempt splits PR, CI fails on child PR
	// second attempt uses GenerateFixCIPrompt, CI passes
	mockSM.On("SaveState", "test-workflow", mock.Anything).Return(nil)

	// First attempt: GeneratePRSplitPrompt
	mockPG.On("GeneratePRSplitPrompt", mock.Anything).Return("pr-split prompt", nil).Once()

	// First execution returns PRs
	mockExec.On("ExecuteStreaming", mock.Anything, mock.MatchedBy(func(config ExecuteConfig) bool {
		return config.Prompt == "pr-split prompt"
	}), mock.Anything).Return(&ExecuteResult{
		Output:   `{"parentPR": {"number": 1}, "childPRs": [{"number": 2, "title": "Child PR"}]}`,
		ExitCode: 0,
	}, nil).Once()

	mockOP.On("ExtractJSON", mock.Anything).Return(`{"parentPR": {"number": 1}, "childPRs": [{"number": 2, "title": "Child PR"}]}`, nil)
	mockOP.On("ParsePRSplitResult", mock.Anything).Return(&PRSplitResult{
		ParentPR: PRInfo{Number: 1},
		ChildPRs: []PRInfo{{Number: 2, Title: "Child PR"}},
	}, nil)
	mockSM.On("SavePhaseOutput", "test-workflow", PhasePRSplit, mock.Anything).Return(nil)

	// First CI check fails
	mockCI.On("WaitForCIWithProgress", mock.Anything, 2, mock.Anything, mock.Anything, mock.Anything).Return(&CIResult{
		Passed:     false,
		Status:     "failure",
		FailedJobs: []string{"build"},
		Output:     "Build failed",
	}, nil).Once()

	// Second attempt: GenerateFixCIPrompt should be called (this is the fix we're testing)
	mockPG.On("GenerateFixCIPrompt", mock.Anything).Return("fix ci prompt", nil).Once()

	// Second execution with fix prompt
	mockExec.On("ExecuteStreaming", mock.Anything, mock.MatchedBy(func(config ExecuteConfig) bool {
		return config.Prompt == "fix ci prompt"
	}), mock.Anything).Return(&ExecuteResult{
		Output:   `{"parentPR": {"number": 1}, "childPRs": [{"number": 2, "title": "Child PR"}]}`,
		ExitCode: 0,
	}, nil).Once()

	// Second CI check passes
	mockCI.On("WaitForCIWithProgress", mock.Anything, 2, mock.Anything, mock.Anything, mock.Anything).Return(&CIResult{
		Passed: true,
		Status: "success",
	}, nil).Once()

	config := DefaultConfig("/tmp/workflows")
	config.MaxFixAttempts = 3

	o := &Orchestrator{
		stateManager:    mockSM,
		executor:        mockExec,
		promptGenerator: mockPG,
		parser:          mockOP,
		config:          config,
		ciCheckerFactory: func(workingDir string, checkInterval time.Duration, commandTimeout time.Duration) CIChecker {
			return mockCI
		},
	}

	state := &WorkflowState{
		Name:         "test-workflow",
		CurrentPhase: PhasePRSplit,
		Phases: map[Phase]*PhaseState{
			PhasePlanning:       {Status: StatusCompleted},
			PhaseConfirmation:   {Status: StatusCompleted},
			PhaseImplementation: {Status: StatusCompleted},
			PhaseRefactoring:    {Status: StatusCompleted},
			PhasePRSplit:        {Status: StatusInProgress, Metrics: &PRMetrics{LinesChanged: 150, FilesChanged: 15}},
		},
	}

	err := o.executePRSplit(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, PhaseCompleted, state.CurrentPhase)

	// Verify GenerateFixCIPrompt was called (not just raw error passed)
	mockPG.AssertCalled(t, "GenerateFixCIPrompt", mock.Anything)
	mockSM.AssertExpectations(t)
	mockExec.AssertExpectations(t)
	mockPG.AssertExpectations(t)
}

func TestOrchestrator_Status(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockStateManager)
		want       *WorkflowState
		wantErr    bool
	}{
		{
			name: "successfully returns workflow status",
			setupMocks: func(sm *MockStateManager) {
				state := &WorkflowState{
					Name:         "test-workflow",
					CurrentPhase: PhasePlanning,
				}
				sm.On("LoadState", "test-workflow").Return(state, nil)
			},
			want: &WorkflowState{
				Name:         "test-workflow",
				CurrentPhase: PhasePlanning,
			},
			wantErr: false,
		},
		{
			name: "fails when workflow not found",
			setupMocks: func(sm *MockStateManager) {
				sm.On("LoadState", "test-workflow").Return((*WorkflowState)(nil), ErrWorkflowNotFound)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			tt.setupMocks(mockSM)

			o := &Orchestrator{
				stateManager: mockSM,
			}

			got, err := o.Status("test-workflow")

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			mockSM.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_Delete(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockStateManager)
		wantErr    bool
	}{
		{
			name: "successfully deletes workflow",
			setupMocks: func(sm *MockStateManager) {
				sm.On("DeleteWorkflow", "test-workflow").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "fails when workflow not found",
			setupMocks: func(sm *MockStateManager) {
				sm.On("DeleteWorkflow", "test-workflow").Return(ErrWorkflowNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			tt.setupMocks(mockSM)

			o := &Orchestrator{
				stateManager: mockSM,
			}

			err := o.Delete("test-workflow")

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			mockSM.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_failWorkflow(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		setupMocks func(*MockStateManager)
	}{
		{
			name: "successfully transitions to failed state",
			err:  errors.New("test error"),
			setupMocks: func(sm *MockStateManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			tt.setupMocks(mockSM)

			o := &Orchestrator{
				stateManager: mockSM,
			}

			state := &WorkflowState{
				Name:         "test-workflow",
				CurrentPhase: PhasePlanning,
				Phases: map[Phase]*PhaseState{
					PhasePlanning: {Status: StatusInProgress},
				},
			}

			err := o.failWorkflow(state, tt.err)

			require.Error(t, err)
			assert.Equal(t, PhaseFailed, state.CurrentPhase)
			assert.NotNil(t, state.Error)
			assert.Equal(t, tt.err.Error(), state.Error.Message)
			assert.Equal(t, FailureTypeExecution, state.Error.FailureType)
			assert.Equal(t, StatusFailed, state.Phases[PhasePlanning].Status)
			mockSM.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_failWorkflowCI(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		setupMocks func(*MockStateManager)
	}{
		{
			name: "successfully transitions to failed state with CI failure type",
			err:  errors.New("ci check failed"),
			setupMocks: func(sm *MockStateManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			tt.setupMocks(mockSM)

			o := &Orchestrator{
				stateManager: mockSM,
			}

			state := &WorkflowState{
				Name:         "test-workflow",
				CurrentPhase: PhasePlanning,
				Phases: map[Phase]*PhaseState{
					PhasePlanning: {Status: StatusInProgress},
				},
			}

			err := o.failWorkflowCI(state, tt.err)

			require.Error(t, err)
			assert.Equal(t, PhaseFailed, state.CurrentPhase)
			assert.NotNil(t, state.Error)
			assert.Equal(t, tt.err.Error(), state.Error.Message)
			assert.Equal(t, FailureTypeCI, state.Error.FailureType)
			assert.Equal(t, StatusFailed, state.Phases[PhasePlanning].Status)
			mockSM.AssertExpectations(t)
		})
	}
}

func TestOrchestrator_failWorkflowWithType(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		failureType FailureType
		setupMocks  func(*MockStateManager)
	}{
		{
			name:        "successfully transitions to failed state with execution failure type",
			err:         errors.New("execution failed"),
			failureType: FailureTypeExecution,
			setupMocks: func(sm *MockStateManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
			},
		},
		{
			name:        "successfully transitions to failed state with CI failure type",
			err:         errors.New("ci check failed"),
			failureType: FailureTypeCI,
			setupMocks: func(sm *MockStateManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
			},
		},
		{
			name:        "handles save state error",
			err:         errors.New("original error"),
			failureType: FailureTypeExecution,
			setupMocks: func(sm *MockStateManager) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(errors.New("save failed"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			tt.setupMocks(mockSM)

			o := &Orchestrator{
				stateManager: mockSM,
			}

			state := &WorkflowState{
				Name:         "test-workflow",
				CurrentPhase: PhasePlanning,
				Phases: map[Phase]*PhaseState{
					PhasePlanning: {Status: StatusInProgress},
				},
			}

			err := o.failWorkflowWithType(state, tt.err, tt.failureType)

			require.Error(t, err)
			assert.Equal(t, PhaseFailed, state.CurrentPhase)
			assert.NotNil(t, state.Error)
			assert.Equal(t, tt.failureType, state.Error.FailureType)
			assert.Equal(t, StatusFailed, state.Phases[PhasePlanning].Status)

			if tt.name == "handles save state error" {
				assert.Contains(t, err.Error(), "failed to save failed state")
				assert.Contains(t, err.Error(), "original error")
			} else {
				assert.Equal(t, tt.err.Error(), err.Error())
			}

			mockSM.AssertExpectations(t)
		})
	}
}

func TestDefaultConfirmFunc(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantApproved bool
		wantFeedback string
		wantErr      bool
		wantErrMsg   string
	}{
		{
			name:         "approves with y",
			input:        "y\n",
			wantApproved: true,
			wantFeedback: "",
			wantErr:      false,
		},
		{
			name:         "approves with yes",
			input:        "yes\n",
			wantApproved: true,
			wantFeedback: "",
			wantErr:      false,
		},
		{
			name:         "approves with Y uppercase",
			input:        "Y\n",
			wantApproved: true,
			wantFeedback: "",
			wantErr:      false,
		},
		{
			name:         "approves with YES uppercase",
			input:        "YES\n",
			wantApproved: true,
			wantFeedback: "",
			wantErr:      false,
		},
		{
			name:         "rejects with n",
			input:        "n\n",
			wantApproved: false,
			wantFeedback: "",
			wantErr:      true,
			wantErrMsg:   "workflow cancelled by user",
		},
		{
			name:         "rejects with no",
			input:        "no\n",
			wantApproved: false,
			wantFeedback: "",
			wantErr:      true,
			wantErrMsg:   "workflow cancelled by user",
		},
		{
			name:         "handles feedback input directly",
			input:        "please add more tests\n",
			wantApproved: false,
			wantFeedback: "please add more tests",
			wantErr:      false,
		},
		{
			name:         "handles empty input then valid input",
			input:        "\ny\n",
			wantApproved: true,
			wantFeedback: "",
			wantErr:      false,
		},
		{
			name:         "handles whitespace-only input then valid input",
			input:        "   \ny\n",
			wantApproved: true,
			wantFeedback: "",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a pipe to simulate stdin
			r, w, err := os.Pipe()
			require.NoError(t, err)
			defer r.Close()

			// Save original stdin and restore after test
			oldStdin := os.Stdin
			os.Stdin = r
			defer func() { os.Stdin = oldStdin }()

			// Write test input in a goroutine
			go func() {
				defer w.Close()
				w.WriteString(tt.input)
			}()

			plan := &Plan{
				Summary: "Test plan summary",
				Phases: []PlanPhase{
					{
						Name:           "Phase 1",
						Description:    "Test phase",
						EstimatedFiles: 1,
						EstimatedLines: 10,
					},
				},
			}

			approved, feedback, err := defaultConfirmFunc(plan)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantApproved, approved)
			assert.Equal(t, tt.wantFeedback, feedback)
		})
	}
}

func TestGetCIChecker(t *testing.T) {
	tests := []struct {
		name             string
		ciCheckerFactory func(workingDir string, checkInterval time.Duration, commandTimeout time.Duration) CIChecker
		workingDir       string
		wantMock         bool
	}{
		{
			name: "uses factory when set",
			ciCheckerFactory: func(workingDir string, checkInterval time.Duration, commandTimeout time.Duration) CIChecker {
				mockCI := new(MockCIChecker)
				return mockCI
			},
			workingDir: "/tmp/worktree",
			wantMock:   true,
		},
		{
			name:             "creates real checker when factory is nil",
			ciCheckerFactory: nil,
			workingDir:       "/tmp/worktree",
			wantMock:         false,
		},
		{
			name: "passes correct working directory to factory",
			ciCheckerFactory: func(workingDir string, checkInterval time.Duration, commandTimeout time.Duration) CIChecker {
				assert.Equal(t, "/custom/worktree/path", workingDir)
				mockCI := new(MockCIChecker)
				return mockCI
			},
			workingDir: "/custom/worktree/path",
			wantMock:   true,
		},
		{
			name: "passes config check interval to factory",
			ciCheckerFactory: func(workingDir string, checkInterval time.Duration, commandTimeout time.Duration) CIChecker {
				assert.Equal(t, 45*time.Second, checkInterval)
				mockCI := new(MockCIChecker)
				return mockCI
			},
			workingDir: "/tmp/worktree",
			wantMock:   true,
		},
		{
			name: "passes config command timeout to factory",
			ciCheckerFactory: func(workingDir string, checkInterval time.Duration, commandTimeout time.Duration) CIChecker {
				assert.Equal(t, 3*time.Minute, commandTimeout)
				mockCI := new(MockCIChecker)
				return mockCI
			},
			workingDir: "/tmp/worktree",
			wantMock:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig("/tmp/workflows")
			if tt.name == "passes config check interval to factory" {
				config.CICheckInterval = 45 * time.Second
			}
			if tt.name == "passes config command timeout to factory" {
				config.GHCommandTimeout = 3 * time.Minute
			}

			o := &Orchestrator{
				config:           config,
				ciCheckerFactory: tt.ciCheckerFactory,
			}

			checker := o.getCIChecker(tt.workingDir)

			assert.NotNil(t, checker)
			if tt.wantMock {
				_, ok := checker.(*MockCIChecker)
				assert.True(t, ok, "expected MockCIChecker")
			}
		})
	}
}

func TestOrchestrator_getPRMetrics(t *testing.T) {
	tests := []struct {
		name       string
		gitDiff    string
		want       *PRMetrics
		wantErr    bool
		setupMocks func(ctx context.Context, workingDir string)
	}{
		{
			name: "successfully parses git diff output",
			want: &PRMetrics{
				LinesChanged:  10,
				FilesChanged:  1,
				FilesAdded:    []string{},
				FilesModified: []string{"file.go"},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want != nil {
				metrics, err := parseDiffStat(" file.go | 10 ++++++++++\n 1 file changed, 10 insertions(+)")
				require.NoError(t, err)
				assert.Equal(t, tt.want, metrics)
			}
		})
	}
}

func TestFormatCIErrors(t *testing.T) {
	tests := []struct {
		name   string
		result *CIResult
		want   string
	}{
		{
			name: "formats single failed job",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				Output:     "Build failed: syntax error",
				FailedJobs: []string{"build"},
			},
			want: "CI checks failed with the following errors:\n\nBuild failed: syntax error\n\nFailed jobs:\n- build\n",
		},
		{
			name: "formats multiple failed jobs",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				Output:     "Multiple errors occurred",
				FailedJobs: []string{"build", "test", "lint"},
			},
			want: "CI checks failed with the following errors:\n\nMultiple errors occurred\n\nFailed jobs:\n- build\n- test\n- lint\n",
		},
		{
			name: "handles empty output",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				Output:     "",
				FailedJobs: []string{"deploy"},
			},
			want: "CI checks failed with the following errors:\n\n\n\nFailed jobs:\n- deploy\n",
		},
		{
			name: "handles no failed jobs",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				Output:     "Unknown error",
				FailedJobs: []string{},
			},
			want: "CI checks failed with the following errors:\n\nUnknown error\n\nFailed jobs:\n",
		},
		{
			name: "formats output with multiline errors",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				Output:     "Error 1: Build failed\nError 2: Test failed\nError 3: Lint failed",
				FailedJobs: []string{"ci"},
			},
			want: "CI checks failed with the following errors:\n\nError 1: Build failed\nError 2: Test failed\nError 3: Lint failed\n\nFailed jobs:\n- ci\n",
		},
		{
			name: "formats with special characters in output",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				Output:     "Error: file \"test.go\" has issues\n\tLine 42: syntax error",
				FailedJobs: []string{"build"},
			},
			want: "CI checks failed with the following errors:\n\nError: file \"test.go\" has issues\n\tLine 42: syntax error\n\nFailed jobs:\n- build\n",
		},
		{
			name: "formats with nil failed jobs slice",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				Output:     "CI system error",
				FailedJobs: nil,
			},
			want: "CI checks failed with the following errors:\n\nCI system error\n\nFailed jobs:\n",
		},
		{
			name: "formats with long output",
			result: &CIResult{
				Passed: false,
				Status: "failure",
				Output: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. " +
					"Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. " +
					"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris.",
				FailedJobs: []string{"integration-test"},
			},
			want: "CI checks failed with the following errors:\n\nLorem ipsum dolor sit amet, consectetur adipiscing elit. " +
				"Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. " +
				"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris.\n\nFailed jobs:\n- integration-test\n",
		},
		{
			name: "formats with job names containing spaces",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				Output:     "Job failed",
				FailedJobs: []string{"build and test", "deploy to staging"},
			},
			want: "CI checks failed with the following errors:\n\nJob failed\n\nFailed jobs:\n- build and test\n- deploy to staging\n",
		},
		{
			name: "formats with job names containing special characters",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				Output:     "Job failed",
				FailedJobs: []string{"build/test/deploy", "test:unit"},
			},
			want: "CI checks failed with the following errors:\n\nJob failed\n\nFailed jobs:\n- build/test/deploy\n- test:unit\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCIErrors(tt.result)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOrchestrator_executePlanning_ParseErrors(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockStateManager, *MockClaudeExecutor, *MockPromptGenerator, *MockOutputParser)
		wantPhase  Phase
	}{
		{
			name: "fails when ExtractJSON fails and saves raw output",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				pg.On("GeneratePlanningPrompt", WorkflowTypeFeature, "test description", []string(nil)).Return("planning prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "some invalid output without json",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("", errors.New("no JSON found"))
				sm.On("SaveRawOutput", "test-workflow", PhasePlanning, "some invalid output without json").Return(nil)
				sm.On("WorkflowDir", "test-workflow").Return("/tmp/workflows/test-workflow")
			},
			wantPhase: PhaseFailed,
		},
		{
			name: "fails when ParsePlan fails and saves raw output",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				pg.On("GeneratePlanningPrompt", WorkflowTypeFeature, "test description", []string(nil)).Return("planning prompt", nil)
				exec.On("ExecuteStreaming", mock.Anything, mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"invalid\": \"plan\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"invalid\": \"plan\"}", nil)
				op.On("ParsePlan", mock.Anything).Return((*Plan)(nil), errors.New("invalid plan structure"))
				sm.On("SaveRawOutput", "test-workflow", PhasePlanning, "```json\n{\"invalid\": \"plan\"}\n```").Return(nil)
				sm.On("WorkflowDir", "test-workflow").Return("/tmp/workflows/test-workflow")
			},
			wantPhase: PhaseFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := new(MockStateManager)
			mockExec := new(MockClaudeExecutor)
			mockPG := new(MockPromptGenerator)
			mockOP := new(MockOutputParser)

			tt.setupMocks(mockSM, mockExec, mockPG, mockOP)

			o := &Orchestrator{
				stateManager:    mockSM,
				executor:        mockExec,
				promptGenerator: mockPG,
				parser:          mockOP,
				config:          DefaultConfig("/tmp/workflows"),
			}

			state := &WorkflowState{
				Name:         "test-workflow",
				Type:         WorkflowTypeFeature,
				Description:  "test description",
				CurrentPhase: PhasePlanning,
				Phases: map[Phase]*PhaseState{
					PhasePlanning: {Status: StatusInProgress},
				},
			}

			err := o.executePlanning(context.Background(), state)

			require.Error(t, err)
			assert.Equal(t, tt.wantPhase, state.CurrentPhase)
			mockSM.AssertExpectations(t)
			mockExec.AssertExpectations(t)
			mockPG.AssertExpectations(t)
			mockOP.AssertExpectations(t)
		})
	}
}
