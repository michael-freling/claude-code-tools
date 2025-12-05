package workflow

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Config holds configuration for the orchestrator
type Config struct {
	BaseDir    string
	MaxLines   int
	MaxFiles   int
	Timeouts   PhaseTimeouts
	ClaudePath string
}

// PhaseTimeouts holds timeout durations for each phase
type PhaseTimeouts struct {
	Planning       time.Duration
	Implementation time.Duration
	Refactoring    time.Duration
	PRSplit        time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig(baseDir string) *Config {
	return &Config{
		BaseDir:    baseDir,
		MaxLines:   100,
		MaxFiles:   10,
		ClaudePath: "claude",
		Timeouts: PhaseTimeouts{
			Planning:       5 * time.Minute,
			Implementation: 30 * time.Minute,
			Refactoring:    15 * time.Minute,
			PRSplit:        10 * time.Minute,
		},
	}
}

// Orchestrator manages workflow execution
type Orchestrator struct {
	stateManager    StateManager
	executor        ClaudeExecutor
	promptGenerator PromptGenerator
	parser          OutputParser
	config          *Config
	confirmFunc     func(plan *Plan) (bool, string, error)
}

// NewOrchestrator creates orchestrator with default config
func NewOrchestrator(baseDir string) (*Orchestrator, error) {
	return NewOrchestratorWithConfig(DefaultConfig(baseDir))
}

// NewOrchestratorWithConfig creates orchestrator with custom config
func NewOrchestratorWithConfig(config *Config) (*Orchestrator, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.BaseDir == "" {
		return nil, fmt.Errorf("baseDir cannot be empty")
	}

	promptGen, err := NewPromptGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt generator: %w", err)
	}

	executor := NewClaudeExecutorWithPath(config.ClaudePath)
	stateManager := NewStateManager(config.BaseDir)
	parser := NewOutputParser()

	return &Orchestrator{
		stateManager:    stateManager,
		executor:        executor,
		promptGenerator: promptGen,
		parser:          parser,
		config:          config,
		confirmFunc:     defaultConfirmFunc,
	}, nil
}

// SetConfirmFunc allows setting a custom confirmation function for testing
func (o *Orchestrator) SetConfirmFunc(fn func(plan *Plan) (bool, string, error)) {
	o.confirmFunc = fn
}

// Start initializes and runs a new workflow
func (o *Orchestrator) Start(ctx context.Context, name, description string, wfType WorkflowType) error {
	state, err := o.stateManager.InitState(name, description, wfType)
	if err != nil {
		return fmt.Errorf("failed to initialize workflow: %w", err)
	}

	return o.runWorkflow(ctx, state)
}

// Resume continues an existing workflow from current phase
func (o *Orchestrator) Resume(ctx context.Context, name string) error {
	state, err := o.stateManager.LoadState(name)
	if err != nil {
		return fmt.Errorf("failed to load workflow state: %w", err)
	}

	if state.CurrentPhase == PhaseCompleted {
		return fmt.Errorf("workflow is already completed")
	}

	if state.Error != nil && !state.Error.Recoverable {
		return fmt.Errorf("workflow is in non-recoverable error state: %w", state.Error)
	}

	if state.Error != nil {
		state.Error = nil
		if err := o.stateManager.SaveState(name, state); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}
	}

	return o.runWorkflow(ctx, state)
}

// Status returns current workflow state
func (o *Orchestrator) Status(name string) (*WorkflowState, error) {
	return o.stateManager.LoadState(name)
}

// List returns all workflows with metadata
func (o *Orchestrator) List() ([]WorkflowInfo, error) {
	return o.stateManager.ListWorkflows()
}

// Delete removes a workflow and all its state
func (o *Orchestrator) Delete(name string) error {
	return o.stateManager.DeleteWorkflow(name)
}

// Clean removes all completed workflows
func (o *Orchestrator) Clean() ([]string, error) {
	workflows, err := o.stateManager.ListWorkflows()
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	var deleted []string
	for _, wf := range workflows {
		if wf.Status == "completed" {
			if err := o.stateManager.DeleteWorkflow(wf.Name); err != nil {
				continue
			}
			deleted = append(deleted, wf.Name)
		}
	}

	return deleted, nil
}

// runWorkflow executes the workflow state machine
func (o *Orchestrator) runWorkflow(ctx context.Context, state *WorkflowState) error {
	fmt.Println(Bold(Cyan("Claude Workflow Orchestrator")))
	fmt.Println(strings.Repeat("=", 30))
	fmt.Printf("\n%s: %s\n", Bold("Workflow"), state.Name)
	fmt.Printf("%s: %s\n", Bold("Type"), state.Type)
	fmt.Printf("%s: %s\n", Bold("Description"), state.Description)

	for {
		if state.CurrentPhase == PhaseCompleted || state.CurrentPhase == PhaseFailed {
			if state.CurrentPhase == PhaseCompleted {
				elapsed := time.Since(state.CreatedAt)
				fmt.Printf("\n%s Workflow completed in %s\n", Green("✓"), FormatDuration(elapsed))
			}
			return nil
		}

		if err := o.executePhase(ctx, state); err != nil {
			return err
		}

		if state.CurrentPhase == PhaseCompleted || state.CurrentPhase == PhaseFailed {
			if state.CurrentPhase == PhaseCompleted {
				elapsed := time.Since(state.CreatedAt)
				fmt.Printf("\n%s Workflow completed in %s\n", Green("✓"), FormatDuration(elapsed))
			}
			return nil
		}
	}
}

// executePhase executes the current phase and transitions to the next
func (o *Orchestrator) executePhase(ctx context.Context, state *WorkflowState) error {
	switch state.CurrentPhase {
	case PhasePlanning:
		return o.executePlanning(ctx, state)
	case PhaseConfirmation:
		return o.executeConfirmation(ctx, state)
	case PhaseImplementation:
		return o.executeImplementation(ctx, state)
	case PhaseRefactoring:
		return o.executeRefactoring(ctx, state)
	case PhasePRSplit:
		return o.executePRSplit(ctx, state)
	default:
		return o.failWorkflow(state, fmt.Errorf("%w: %s", ErrInvalidPhase, state.CurrentPhase))
	}
}

// executePlanning runs the planning phase
func (o *Orchestrator) executePlanning(ctx context.Context, state *WorkflowState) error {
	fmt.Printf("\n%s\n", Bold(FormatPhase(PhasePlanning, 5)))
	fmt.Println(strings.Repeat("-", len(FormatPhase(PhasePlanning, 5))))

	phaseState := state.Phases[PhasePlanning]
	phaseState.Attempts++
	now := time.Now()
	phaseState.StartedAt = &now

	if err := o.stateManager.SaveState(state.Name, state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	prompt, err := o.promptGenerator.GeneratePlanningPrompt(state.Type, state.Description, phaseState.Feedback)
	if err != nil {
		return o.failWorkflow(state, fmt.Errorf("failed to generate planning prompt: %w", err))
	}

	spinner := NewSpinner("Invoking Claude Code to analyze codebase...")
	spinner.Start()

	result, err := o.executor.Execute(ctx, ExecuteConfig{
		Prompt:  prompt,
		Timeout: o.config.Timeouts.Planning,
	})

	if err != nil {
		spinner.Fail("Planning failed")
		return o.failWorkflow(state, fmt.Errorf("failed to execute planning: %w", err))
	}

	jsonStr, err := o.parser.ExtractJSON(result.Output)
	if err != nil {
		spinner.Fail("Failed to parse planning output")
		return o.failWorkflow(state, fmt.Errorf("failed to extract JSON from planning output: %w", err))
	}

	plan, err := o.parser.ParsePlan(jsonStr)
	if err != nil {
		spinner.Fail("Failed to parse plan")
		return o.failWorkflow(state, fmt.Errorf("failed to parse plan: %w", err))
	}

	if err := o.stateManager.SavePlan(state.Name, plan); err != nil {
		spinner.Fail("Failed to save plan")
		return o.failWorkflow(state, fmt.Errorf("failed to save plan: %w", err))
	}

	if err := o.stateManager.SavePhaseOutput(state.Name, PhasePlanning, plan); err != nil {
		spinner.Fail("Failed to save planning output")
		return o.failWorkflow(state, fmt.Errorf("failed to save planning output: %w", err))
	}

	spinner.Success("Plan created")

	return o.transitionPhase(state, PhaseConfirmation)
}

// executeConfirmation runs the confirmation phase
func (o *Orchestrator) executeConfirmation(ctx context.Context, state *WorkflowState) error {
	fmt.Printf("\n%s\n", Bold(FormatPhase(PhaseConfirmation, 5)))
	fmt.Println(strings.Repeat("-", len(FormatPhase(PhaseConfirmation, 5))))

	phaseState := state.Phases[PhaseConfirmation]
	phaseState.Attempts++
	now := time.Now()
	phaseState.StartedAt = &now

	if err := o.stateManager.SaveState(state.Name, state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	plan, err := o.stateManager.LoadPlan(state.Name)
	if err != nil {
		return o.failWorkflow(state, fmt.Errorf("failed to load plan: %w", err))
	}

	approved, feedback, err := o.confirmFunc(plan)
	if err != nil {
		return o.failWorkflow(state, fmt.Errorf("confirmation failed: %w", err))
	}

	if !approved {
		planningPhase := state.Phases[PhasePlanning]
		planningPhase.Feedback = append(planningPhase.Feedback, feedback)
		planningPhase.Status = StatusPending
		return o.transitionPhase(state, PhasePlanning)
	}

	return o.transitionPhase(state, PhaseImplementation)
}

// executeImplementation runs the implementation phase
func (o *Orchestrator) executeImplementation(ctx context.Context, state *WorkflowState) error {
	fmt.Printf("\n%s\n", Bold(FormatPhase(PhaseImplementation, 5)))
	fmt.Println(strings.Repeat("-", len(FormatPhase(PhaseImplementation, 5))))

	phaseState := state.Phases[PhaseImplementation]
	phaseState.Attempts++
	now := time.Now()
	phaseState.StartedAt = &now

	if err := o.stateManager.SaveState(state.Name, state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	plan, err := o.stateManager.LoadPlan(state.Name)
	if err != nil {
		return o.failWorkflow(state, fmt.Errorf("failed to load plan: %w", err))
	}

	prompt, err := o.promptGenerator.GenerateImplementationPrompt(plan)
	if err != nil {
		return o.failWorkflow(state, fmt.Errorf("failed to generate implementation prompt: %w", err))
	}

	spinner := NewSpinner("Implementing changes...")
	spinner.Start()

	result, err := o.executor.Execute(ctx, ExecuteConfig{
		Prompt:  prompt,
		Timeout: o.config.Timeouts.Implementation,
	})

	if err != nil {
		spinner.Fail("Implementation failed")
		return o.failWorkflow(state, fmt.Errorf("failed to execute implementation: %w", err))
	}

	jsonStr, err := o.parser.ExtractJSON(result.Output)
	if err != nil {
		spinner.Fail("Failed to parse implementation output")
		return o.failWorkflow(state, fmt.Errorf("failed to extract JSON from implementation output: %w", err))
	}

	summary, err := o.parser.ParseImplementationSummary(jsonStr)
	if err != nil {
		spinner.Fail("Failed to parse implementation summary")
		return o.failWorkflow(state, fmt.Errorf("failed to parse implementation summary: %w", err))
	}

	if err := o.stateManager.SavePhaseOutput(state.Name, PhaseImplementation, summary); err != nil {
		spinner.Fail("Failed to save implementation output")
		return o.failWorkflow(state, fmt.Errorf("failed to save implementation output: %w", err))
	}

	spinner.Success("Implementation complete")

	return o.transitionPhase(state, PhaseRefactoring)
}

// executeRefactoring runs the refactoring phase
func (o *Orchestrator) executeRefactoring(ctx context.Context, state *WorkflowState) error {
	fmt.Printf("\n%s\n", Bold(FormatPhase(PhaseRefactoring, 5)))
	fmt.Println(strings.Repeat("-", len(FormatPhase(PhaseRefactoring, 5))))

	phaseState := state.Phases[PhaseRefactoring]
	phaseState.Attempts++
	now := time.Now()
	phaseState.StartedAt = &now

	if err := o.stateManager.SaveState(state.Name, state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	plan, err := o.stateManager.LoadPlan(state.Name)
	if err != nil {
		return o.failWorkflow(state, fmt.Errorf("failed to load plan: %w", err))
	}

	prompt, err := o.promptGenerator.GenerateRefactoringPrompt(plan)
	if err != nil {
		return o.failWorkflow(state, fmt.Errorf("failed to generate refactoring prompt: %w", err))
	}

	spinner := NewSpinner("Refactoring code...")
	spinner.Start()

	result, err := o.executor.Execute(ctx, ExecuteConfig{
		Prompt:  prompt,
		Timeout: o.config.Timeouts.Refactoring,
	})

	if err != nil {
		spinner.Fail("Refactoring failed")
		return o.failWorkflow(state, fmt.Errorf("failed to execute refactoring: %w", err))
	}

	jsonStr, err := o.parser.ExtractJSON(result.Output)
	if err != nil {
		spinner.Fail("Failed to parse refactoring output")
		return o.failWorkflow(state, fmt.Errorf("failed to extract JSON from refactoring output: %w", err))
	}

	summary, err := o.parser.ParseRefactoringSummary(jsonStr)
	if err != nil {
		spinner.Fail("Failed to parse refactoring summary")
		return o.failWorkflow(state, fmt.Errorf("failed to parse refactoring summary: %w", err))
	}

	if err := o.stateManager.SavePhaseOutput(state.Name, PhaseRefactoring, summary); err != nil {
		spinner.Fail("Failed to save refactoring output")
		return o.failWorkflow(state, fmt.Errorf("failed to save refactoring output: %w", err))
	}

	spinner.Success("Refactoring complete")

	metrics, err := o.getPRMetrics(ctx)
	if err != nil {
		return o.failWorkflow(state, fmt.Errorf("failed to get PR metrics: %w", err))
	}

	prSplitPhase := state.Phases[PhasePRSplit]
	prSplitPhase.Metrics = metrics

	needsPRSplit := metrics.LinesChanged > o.config.MaxLines || metrics.FilesChanged > o.config.MaxFiles
	required := needsPRSplit
	prSplitPhase.Required = &required

	if err := o.stateManager.SaveState(state.Name, state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	if needsPRSplit {
		return o.transitionPhase(state, PhasePRSplit)
	}

	return o.transitionPhase(state, PhaseCompleted)
}

// executePRSplit runs the PR split phase
func (o *Orchestrator) executePRSplit(ctx context.Context, state *WorkflowState) error {
	fmt.Printf("\n%s\n", Bold(FormatPhase(PhasePRSplit, 5)))
	fmt.Println(strings.Repeat("-", len(FormatPhase(PhasePRSplit, 5))))

	phaseState := state.Phases[PhasePRSplit]
	phaseState.Attempts++
	now := time.Now()
	phaseState.StartedAt = &now

	if err := o.stateManager.SaveState(state.Name, state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	if phaseState.Metrics == nil {
		return o.failWorkflow(state, fmt.Errorf("PR metrics not available"))
	}

	prompt, err := o.promptGenerator.GeneratePRSplitPrompt(phaseState.Metrics)
	if err != nil {
		return o.failWorkflow(state, fmt.Errorf("failed to generate PR split prompt: %w", err))
	}

	spinner := NewSpinner("Splitting PR into manageable pieces...")
	spinner.Start()

	result, err := o.executor.Execute(ctx, ExecuteConfig{
		Prompt:  prompt,
		Timeout: o.config.Timeouts.PRSplit,
	})

	if err != nil {
		spinner.Fail("PR split failed")
		return o.failWorkflow(state, fmt.Errorf("failed to execute PR split: %w", err))
	}

	jsonStr, err := o.parser.ExtractJSON(result.Output)
	if err != nil {
		spinner.Fail("Failed to parse PR split output")
		return o.failWorkflow(state, fmt.Errorf("failed to extract JSON from PR split output: %w", err))
	}

	prResult, err := o.parser.ParsePRSplitResult(jsonStr)
	if err != nil {
		spinner.Fail("Failed to parse PR split result")
		return o.failWorkflow(state, fmt.Errorf("failed to parse PR split result: %w", err))
	}

	if err := o.stateManager.SavePhaseOutput(state.Name, PhasePRSplit, prResult); err != nil {
		spinner.Fail("Failed to save PR split output")
		return o.failWorkflow(state, fmt.Errorf("failed to save PR split output: %w", err))
	}

	spinner.Success("PR split complete")

	return o.transitionPhase(state, PhaseCompleted)
}

// transitionPhase transitions the workflow to the next phase
func (o *Orchestrator) transitionPhase(state *WorkflowState, nextPhase Phase) error {
	currentPhaseState := state.Phases[state.CurrentPhase]
	now := time.Now()
	currentPhaseState.CompletedAt = &now
	currentPhaseState.Status = StatusCompleted

	state.CurrentPhase = nextPhase

	if nextPhase == PhaseCompleted || nextPhase == PhaseFailed {
		if err := o.stateManager.SaveState(state.Name, state); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}
		return nil
	}

	nextPhaseState := state.Phases[nextPhase]
	nextPhaseState.Status = StatusInProgress

	if err := o.stateManager.SaveState(state.Name, state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// failWorkflow transitions the workflow to failed state
func (o *Orchestrator) failWorkflow(state *WorkflowState, err error) error {
	state.Error = &WorkflowError{
		Message:     err.Error(),
		Phase:       state.CurrentPhase,
		Timestamp:   time.Now(),
		Recoverable: isRecoverableError(err),
	}

	currentPhaseState := state.Phases[state.CurrentPhase]
	currentPhaseState.Status = StatusFailed

	state.CurrentPhase = PhaseFailed

	if saveErr := o.stateManager.SaveState(state.Name, state); saveErr != nil {
		return fmt.Errorf("failed to save failed state: %w (original error: %v)", saveErr, err)
	}

	return err
}

// getPRMetrics collects PR metrics from git diff
func (o *Orchestrator) getPRMetrics(ctx context.Context) (*PRMetrics, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--stat", "origin/main")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run git diff: %w", err)
	}

	return parseDiffStat(string(output))
}

// parseDiffStat parses git diff --stat output
func parseDiffStat(output string) (*PRMetrics, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return &PRMetrics{}, nil
	}

	metrics := &PRMetrics{
		FilesAdded:    []string{},
		FilesModified: []string{},
		FilesDeleted:  []string{},
	}

	for i, line := range lines {
		if i == len(lines)-1 {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				filesChanged, _ := strconv.Atoi(parts[0])
				metrics.FilesChanged = filesChanged
			}
			if len(parts) >= 4 {
				linesChanged, _ := strconv.Atoi(parts[3])
				metrics.LinesChanged = linesChanged
			}
			continue
		}

		parts := strings.Fields(line)
		if len(parts) > 0 {
			fileName := parts[0]
			if strings.Contains(line, "(new)") {
				metrics.FilesAdded = append(metrics.FilesAdded, fileName)
			} else if strings.Contains(line, "(gone)") {
				metrics.FilesDeleted = append(metrics.FilesDeleted, fileName)
			} else {
				metrics.FilesModified = append(metrics.FilesModified, fileName)
			}
		}
	}

	return metrics, nil
}

// isRecoverableError determines if an error is recoverable
func isRecoverableError(err error) bool {
	if err == nil {
		return false
	}

	switch {
	case strings.Contains(err.Error(), "timeout"):
		return true
	case strings.Contains(err.Error(), "claude execution failed"):
		return true
	case strings.Contains(err.Error(), "failed to parse"):
		return false
	case strings.Contains(err.Error(), "invalid"):
		return false
	default:
		return true
	}
}

// defaultConfirmFunc is the default confirmation function that reads from stdin
func defaultConfirmFunc(plan *Plan) (bool, string, error) {
	fmt.Println()
	fmt.Println(FormatPlanSummary(plan))
	fmt.Println()
	fmt.Println(Cyan("Full plan saved to: .claude/workflow/<name>/plan.md"))
	fmt.Println()
	fmt.Print(Bold("Approve this plan? [y/n/feedback]: "))

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false, "", fmt.Errorf("failed to read input")
	}

	response := strings.TrimSpace(strings.ToLower(scanner.Text()))

	if response == "yes" || response == "y" {
		return true, "", nil
	}

	if response == "no" || response == "n" {
		return false, "", ErrUserCancelled
	}

	fmt.Print(Yellow("Please provide your feedback: "))
	if !scanner.Scan() {
		return false, "", fmt.Errorf("failed to read feedback")
	}

	feedback := strings.TrimSpace(scanner.Text())
	if feedback == "" {
		return false, "", fmt.Errorf("feedback cannot be empty")
	}

	fmt.Println(Green("✓") + " Feedback received. Replanning with your suggestions...")

	return false, feedback, nil
}
