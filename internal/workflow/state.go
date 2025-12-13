package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

const (
	stateVersion  = "1.0"
	stateFileName = "state.json"
	lockFileName  = ".lock"
	planFileName  = "plan.json"
	planMdFile    = "plan.md"
	phasesDir     = "phases"
)

// TimeProvider provides the current time (allows mocking in tests)
type TimeProvider func() time.Time

// StateManager interface for state persistence operations
type StateManager interface {
	// Workflow directory operations
	EnsureWorkflowDir(name string) error
	WorkflowExists(name string) bool
	WorkflowDir(name string) string

	// State operations (with automatic locking)
	LoadState(name string) (*WorkflowState, error)
	SaveState(name string, state *WorkflowState) error
	InitState(name, description string, wfType WorkflowType) (*WorkflowState, error)

	// Plan operations
	SavePlan(name string, plan *Plan) error
	LoadPlan(name string) (*Plan, error)
	SavePlanMarkdown(name string, markdown string) error

	// Phase output operations
	SavePhaseOutput(name string, phase Phase, data interface{}) error
	LoadPhaseOutput(name string, phase Phase, target interface{}) error

	// Debug output operations
	SaveRawOutput(name string, phase Phase, output string) error

	// List and delete
	ListWorkflows() ([]WorkflowInfo, error)
	DeleteWorkflow(name string) error

	// Time provider for testing
	SetTimeProvider(tp TimeProvider)
}

// fileStateManager implements StateManager using file-based storage
type fileStateManager struct {
	baseDir      string
	locks        map[string]*flock.Flock
	mu           sync.Mutex
	timeProvider TimeProvider
}

// NewStateManager creates a new file-based state manager
func NewStateManager(baseDir string) StateManager {
	return &fileStateManager{
		baseDir:      baseDir,
		locks:        make(map[string]*flock.Flock),
		timeProvider: time.Now,
	}
}

// SetTimeProvider sets a custom time provider for testing
func (s *fileStateManager) SetTimeProvider(tp TimeProvider) {
	s.timeProvider = tp
}

// WorkflowDir returns the directory path for a workflow
func (s *fileStateManager) WorkflowDir(name string) string {
	return filepath.Join(s.baseDir, name)
}

// EnsureWorkflowDir creates the workflow directory structure if it doesn't exist
func (s *fileStateManager) EnsureWorkflowDir(name string) error {
	if err := ValidateWorkflowName(name); err != nil {
		return err
	}

	workflowDir := s.WorkflowDir(name)
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflow directory: %w", err)
	}

	phasesPath := filepath.Join(workflowDir, phasesDir)
	if err := os.MkdirAll(phasesPath, 0755); err != nil {
		return fmt.Errorf("failed to create phases directory: %w", err)
	}

	return nil
}

// WorkflowExists checks if a workflow directory exists
func (s *fileStateManager) WorkflowExists(name string) bool {
	if err := ValidateWorkflowName(name); err != nil {
		return false
	}

	statePath := filepath.Join(s.WorkflowDir(name), stateFileName)
	_, err := os.Stat(statePath)
	return err == nil
}

// lock acquires a file lock for the workflow
func (s *fileStateManager) lock(name string) (*flock.Flock, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lockPath := filepath.Join(s.WorkflowDir(name), lockFileName)
	fileLock := flock.New(lockPath)

	locked, err := fileLock.TryLock()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !locked {
		return nil, ErrStateLocked
	}

	s.locks[name] = fileLock
	return fileLock, nil
}

// unlock releases the file lock for the workflow
func (s *fileStateManager) unlock(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileLock, ok := s.locks[name]
	if !ok {
		return nil
	}

	if err := fileLock.Unlock(); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	delete(s.locks, name)
	return nil
}

// LoadState loads workflow state from disk
func (s *fileStateManager) LoadState(name string) (*WorkflowState, error) {
	if err := ValidateWorkflowName(name); err != nil {
		return nil, err
	}

	if !s.WorkflowExists(name) {
		return nil, fmt.Errorf("%w: %s", ErrWorkflowNotFound, name)
	}

	statePath := filepath.Join(s.WorkflowDir(name), stateFileName)
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state WorkflowState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStateCorrupted, err)
	}

	return &state, nil
}

// SaveState saves workflow state to disk with atomic write
func (s *fileStateManager) SaveState(name string, state *WorkflowState) error {
	if err := ValidateWorkflowName(name); err != nil {
		return err
	}

	if err := s.EnsureWorkflowDir(name); err != nil {
		return err
	}

	fileLock, err := s.lock(name)
	if err != nil {
		return err
	}
	defer s.unlock(name)
	defer fileLock.Close()

	state.UpdatedAt = s.timeProvider()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	statePath := filepath.Join(s.WorkflowDir(name), stateFileName)
	if err := s.atomicWrite(statePath, data); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// InitState initializes a new workflow state
func (s *fileStateManager) InitState(name, description string, wfType WorkflowType) (*WorkflowState, error) {
	if err := ValidateWorkflowName(name); err != nil {
		return nil, err
	}

	if err := ValidateWorkflowType(wfType); err != nil {
		return nil, err
	}

	if err := ValidateDescription(description); err != nil {
		return nil, err
	}

	if s.WorkflowExists(name) {
		return nil, fmt.Errorf("%w: %s", ErrWorkflowExists, name)
	}

	now := s.timeProvider()
	state := &WorkflowState{
		Version:      stateVersion,
		Name:         name,
		Type:         wfType,
		Description:  description,
		CurrentPhase: PhasePlanning,
		CreatedAt:    now,
		UpdatedAt:    now,
		Phases:       make(map[Phase]*PhaseState),
	}

	// Initialize all phases
	phases := []Phase{
		PhasePlanning,
		PhaseConfirmation,
		PhaseImplementation,
		PhaseRefactoring,
		PhasePRSplit,
	}

	for _, phase := range phases {
		state.Phases[phase] = &PhaseState{
			Status:   StatusPending,
			Attempts: 0,
		}
	}

	// Set planning phase to in_progress
	state.Phases[PhasePlanning].Status = StatusInProgress

	if err := s.SaveState(name, state); err != nil {
		return nil, err
	}

	return state, nil
}

// SavePlan saves the plan to disk
func (s *fileStateManager) SavePlan(name string, plan *Plan) error {
	if err := ValidateWorkflowName(name); err != nil {
		return err
	}

	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	planPath := filepath.Join(s.WorkflowDir(name), planFileName)
	if err := s.atomicWrite(planPath, data); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}

	return nil
}

// LoadPlan loads the plan from disk
func (s *fileStateManager) LoadPlan(name string) (*Plan, error) {
	if err := ValidateWorkflowName(name); err != nil {
		return nil, err
	}

	planPath := filepath.Join(s.WorkflowDir(name), planFileName)
	data, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	return &plan, nil
}

// SavePlanMarkdown saves the plan in markdown format
func (s *fileStateManager) SavePlanMarkdown(name string, markdown string) error {
	if err := ValidateWorkflowName(name); err != nil {
		return err
	}

	planPath := filepath.Join(s.WorkflowDir(name), planMdFile)
	if err := s.atomicWrite(planPath, []byte(markdown)); err != nil {
		return fmt.Errorf("failed to write plan markdown: %w", err)
	}

	return nil
}

// SavePhaseOutput saves phase-specific output
func (s *fileStateManager) SavePhaseOutput(name string, phase Phase, data interface{}) error {
	if err := ValidateWorkflowName(name); err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal phase output: %w", err)
	}

	phaseFile := fmt.Sprintf("%s.json", string(phase))
	phasePath := filepath.Join(s.WorkflowDir(name), phasesDir, phaseFile)

	if err := s.atomicWrite(phasePath, jsonData); err != nil {
		return fmt.Errorf("failed to write phase output: %w", err)
	}

	return nil
}

// LoadPhaseOutput loads phase-specific output
func (s *fileStateManager) LoadPhaseOutput(name string, phase Phase, target interface{}) error {
	if err := ValidateWorkflowName(name); err != nil {
		return err
	}

	phaseFile := fmt.Sprintf("%s.json", string(phase))
	phasePath := filepath.Join(s.WorkflowDir(name), phasesDir, phaseFile)

	data, err := os.ReadFile(phasePath)
	if err != nil {
		return fmt.Errorf("failed to read phase output: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal phase output: %w", err)
	}

	return nil
}

// SaveRawOutput saves raw Claude output for debugging purposes
func (s *fileStateManager) SaveRawOutput(name string, phase Phase, output string) error {
	if err := ValidateWorkflowName(name); err != nil {
		return err
	}

	rawFile := fmt.Sprintf("%s_raw.txt", string(phase))
	rawPath := filepath.Join(s.WorkflowDir(name), phasesDir, rawFile)

	if err := s.atomicWrite(rawPath, []byte(output)); err != nil {
		return fmt.Errorf("failed to write raw output: %w", err)
	}

	return nil
}

// ListWorkflows returns information about all workflows
func (s *fileStateManager) ListWorkflows() ([]WorkflowInfo, error) {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}

	var workflows []WorkflowInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !s.WorkflowExists(name) {
			continue
		}

		state, err := s.LoadState(name)
		if err != nil {
			continue
		}

		status := "in_progress"
		if state.CurrentPhase == PhaseCompleted {
			status = "completed"
		} else if state.CurrentPhase == PhaseFailed || state.Error != nil {
			status = "failed"
		}

		workflows = append(workflows, WorkflowInfo{
			Name:         state.Name,
			Type:         state.Type,
			CurrentPhase: state.CurrentPhase,
			CreatedAt:    state.CreatedAt,
			UpdatedAt:    state.UpdatedAt,
			Status:       status,
		})
	}

	return workflows, nil
}

// DeleteWorkflow removes a workflow and all its state
func (s *fileStateManager) DeleteWorkflow(name string) error {
	if err := ValidateWorkflowName(name); err != nil {
		return err
	}

	if !s.WorkflowExists(name) {
		return fmt.Errorf("%w: %s", ErrWorkflowNotFound, name)
	}

	workflowDir := s.WorkflowDir(name)
	if err := os.RemoveAll(workflowDir); err != nil {
		return fmt.Errorf("failed to delete workflow directory: %w", err)
	}

	return nil
}

// atomicWrite writes data to a file atomically using a temp file and rename
func (s *fileStateManager) atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	tmpFile.Close()

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
