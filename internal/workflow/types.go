package workflow

import (
	"errors"
	"time"
)

// Phase represents a workflow phase
type Phase string

const (
	PhasePlanning       Phase = "PLANNING"
	PhaseConfirmation   Phase = "CONFIRMATION"
	PhaseImplementation Phase = "IMPLEMENTATION"
	PhaseRefactoring    Phase = "REFACTORING"
	PhasePRSplit        Phase = "PR_SPLIT"
	PhaseCompleted      Phase = "COMPLETED"
	PhaseFailed         Phase = "FAILED"
)

// PhaseStatus represents the status of a phase
type PhaseStatus string

const (
	StatusPending    PhaseStatus = "pending"
	StatusInProgress PhaseStatus = "in_progress"
	StatusCompleted  PhaseStatus = "completed"
	StatusSkipped    PhaseStatus = "skipped"
	StatusFailed     PhaseStatus = "failed"
)

// WorkflowType represents the type of workflow
type WorkflowType string

const (
	WorkflowTypeFeature WorkflowType = "feature"
	WorkflowTypeFix     WorkflowType = "fix"
)

// WorkflowState represents the persisted state of a workflow
type WorkflowState struct {
	Version      string                `json:"version"`
	Name         string                `json:"name"`
	Type         WorkflowType          `json:"type"`
	Description  string                `json:"description"`
	CurrentPhase Phase                 `json:"currentPhase"`
	CreatedAt    time.Time             `json:"createdAt"`
	UpdatedAt    time.Time             `json:"updatedAt"`
	Phases       map[Phase]*PhaseState `json:"phases"`
	Error        *WorkflowError        `json:"error,omitempty"`
	WorktreePath string                `json:"worktreePath,omitempty"`
	PRNumber     int                   `json:"prNumber,omitempty"`
}

// PhaseState represents the state of a single phase
type PhaseState struct {
	Status      PhaseStatus `json:"status"`
	StartedAt   *time.Time  `json:"startedAt,omitempty"`
	CompletedAt *time.Time  `json:"completedAt,omitempty"`
	Attempts    int         `json:"attempts"`
	Feedback    []string    `json:"feedback,omitempty"`
	Required    *bool       `json:"required,omitempty"`
	Metrics     *PRMetrics  `json:"metrics,omitempty"`
}

// PRMetrics holds diff statistics for PR split decision
type PRMetrics struct {
	LinesChanged  int      `json:"linesChanged"`
	FilesChanged  int      `json:"filesChanged"`
	FilesAdded    []string `json:"filesAdded,omitempty"`
	FilesModified []string `json:"filesModified,omitempty"`
	FilesDeleted  []string `json:"filesDeleted,omitempty"`
}

// WorkflowError represents an error that occurred during workflow
type WorkflowError struct {
	Message     string                 `json:"message"`
	Phase       Phase                  `json:"phase"`
	Timestamp   time.Time              `json:"timestamp"`
	Recoverable bool                   `json:"recoverable"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

func (e *WorkflowError) Error() string {
	return e.Message
}

// Plan represents the structured plan output from Claude
type Plan struct {
	Summary             string       `json:"summary"`
	ContextType         string       `json:"contextType"`
	Architecture        Architecture `json:"architecture"`
	Phases              []PlanPhase  `json:"phases"`
	WorkStreams         []WorkStream `json:"workStreams"`
	Risks               []string     `json:"risks"`
	Complexity          string       `json:"complexity"`
	EstimatedTotalLines int          `json:"estimatedTotalLines"`
	EstimatedTotalFiles int          `json:"estimatedTotalFiles"`
}

// Architecture describes the architectural overview and components
type Architecture struct {
	Overview   string   `json:"overview"`
	Components []string `json:"components"`
}

// PlanPhase describes a phase in the implementation plan
type PlanPhase struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	EstimatedFiles int    `json:"estimatedFiles"`
	EstimatedLines int    `json:"estimatedLines"`
}

// WorkStream describes a stream of work with dependencies
type WorkStream struct {
	Name      string   `json:"name"`
	Tasks     []string `json:"tasks"`
	DependsOn []string `json:"dependsOn,omitempty"`
}

// ImplementationSummary represents output from implementation phase
type ImplementationSummary struct {
	FilesChanged []string `json:"filesChanged"`
	LinesAdded   int      `json:"linesAdded"`
	LinesRemoved int      `json:"linesRemoved"`
	TestsAdded   int      `json:"testsAdded"`
	PRNumber     int      `json:"prNumber"`
	PRURL        string   `json:"prUrl"`
	Summary      string   `json:"summary"`
	NextSteps    []string `json:"nextSteps,omitempty"`
}

// RefactoringSummary represents output from refactoring phase
type RefactoringSummary struct {
	FilesChanged     []string `json:"filesChanged"`
	ImprovementsMade []string `json:"improvementsMade"`
	Summary          string   `json:"summary"`
}

// PRSplitResult represents output from PR split phase
type PRSplitResult struct {
	ParentPR PRInfo   `json:"parentPR"`
	ChildPRs []PRInfo `json:"childPRs"`
	Summary  string   `json:"summary"`
}

// PRInfo contains information about a pull request
type PRInfo struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// WorkflowInfo represents summary information for listing
type WorkflowInfo struct {
	Name         string       `json:"name"`
	Type         WorkflowType `json:"type"`
	CurrentPhase Phase        `json:"currentPhase"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
	Status       string       `json:"status"`
}

// CheckCIOptions configures CI checking behavior
type CheckCIOptions struct {
	SkipE2E        bool
	E2ETestPattern string
}

// Error variables for common error conditions
var (
	ErrInvalidWorkflowName = errors.New("invalid workflow name")
	ErrInvalidWorkflowType = errors.New("invalid workflow type")
	ErrWorkflowExists      = errors.New("workflow already exists")
	ErrWorkflowNotFound    = errors.New("workflow not found")
	ErrStateCorrupted      = errors.New("state file corrupted")
	ErrStateLocked         = errors.New("workflow is locked by another process")
	ErrInvalidPhase        = errors.New("invalid phase transition")
	ErrClaudeTimeout       = errors.New("claude execution timeout")
	ErrClaudeNotFound      = errors.New("claude CLI not found in PATH")
	ErrClaude              = errors.New("claude execution failed")
	ErrParseJSON           = errors.New("failed to parse JSON output")
	ErrUserCancelled       = errors.New("workflow cancelled by user")
)
