package workflow

import (
	"context"
	"errors"
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
				exec.On("Execute", mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"test plan\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"test plan\"}", nil)
				op.On("ParsePlan", mock.Anything).Return(&Plan{Summary: "test plan"}, nil)
				sm.On("SavePlan", "test-workflow", mock.Anything).Return(nil)
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
				exec.On("Execute", mock.Anything, mock.Anything).Return((*ExecuteResult)(nil), errors.New("execution failed"))
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
		name          string
		setupMocks    func(*MockStateManager, *MockClaudeExecutor, *MockPromptGenerator, *MockOutputParser)
		wantErr       bool
		wantNextPhase Phase
	}{
		{
			name: "successfully implements plan",
			setupMocks: func(sm *MockStateManager, exec *MockClaudeExecutor, pg *MockPromptGenerator, op *MockOutputParser) {
				sm.On("SaveState", "test-workflow", mock.Anything).Return(nil)
				sm.On("LoadPlan", "test-workflow").Return(&Plan{Summary: "test plan"}, nil)
				pg.On("GenerateImplementationPrompt", mock.Anything).Return("implementation prompt", nil)
				exec.On("Execute", mock.Anything, mock.Anything).Return(&ExecuteResult{
					Output:   "```json\n{\"summary\": \"implemented\"}\n```",
					ExitCode: 0,
				}, nil)
				op.On("ExtractJSON", mock.Anything).Return("{\"summary\": \"implemented\"}", nil)
				op.On("ParseImplementationSummary", mock.Anything).Return(&ImplementationSummary{Summary: "implemented"}, nil)
				sm.On("SavePhaseOutput", "test-workflow", PhaseImplementation, mock.Anything).Return(nil)
			},
			wantErr:       false,
			wantNextPhase: PhaseRefactoring,
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
				sm.On("InitState", "test-workflow", "test description", WorkflowTypeFeature).Return((*WorkflowState)(nil), errors.New("init failed"))
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
			name: "timeout is recoverable",
			err:  errors.New("operation timeout"),
			want: true,
		},
		{
			name: "execution failure is recoverable",
			err:  errors.New("claude execution failed"),
			want: true,
		},
		{
			name: "parse error is not recoverable",
			err:  errors.New("failed to parse JSON"),
			want: false,
		},
		{
			name: "invalid input is not recoverable",
			err:  errors.New("invalid workflow name"),
			want: false,
		},
		{
			name: "nil error is not recoverable",
			err:  nil,
			want: false,
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
	assert.Equal(t, 5*time.Minute, config.Timeouts.Planning)
	assert.Equal(t, 30*time.Minute, config.Timeouts.Implementation)
	assert.Equal(t, 15*time.Minute, config.Timeouts.Refactoring)
	assert.Equal(t, 10*time.Minute, config.Timeouts.PRSplit)
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
			name: "parses diff stat with modifications",
			diffOutput: ` file1.go | 10 ++++++++++
 file2.go | 5 +++++
 2 files changed, 15 insertions(+)`,
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
			name:       "handles empty diff",
			diffOutput: "",
			want: &PRMetrics{
				FilesAdded:    []string{},
				FilesModified: []string{},
				FilesDeleted:  []string{},
			},
			wantErr: false,
		},
		{
			name: "parses diff stat with new files",
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
				exec.On("Execute", mock.Anything, mock.Anything).Return(&ExecuteResult{
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
			assert.Equal(t, StatusFailed, state.Phases[PhasePlanning].Status)
			mockSM.AssertExpectations(t)
		})
	}
}
