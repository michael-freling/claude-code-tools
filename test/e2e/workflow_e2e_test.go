//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/michael-freling/claude-code-tools/internal/workflow"
	"github.com/michael-freling/claude-code-tools/test/e2e/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	sandboxRepoURL   = "https://github.com/michael-freling/claude-code-sandbox"
	sandboxRepoOwner = "michael-freling"
	sandboxRepoName  = "claude-code-sandbox"
)

// TestWorkflow_FeatureWorkflow_E2E tests a complete feature workflow with real CI checks
// using the sandbox repository. This test creates real commits, PRs, and waits for real CI.
func TestWorkflow_FeatureWorkflow_E2E(t *testing.T) {
	helpers.RequireClaude(t)
	helpers.RequireGit(t)
	helpers.RequireGHAuth(t)

	repo := helpers.CloneRepo(t, sandboxRepoURL)
	branchName := fmt.Sprintf("e2e-feature-%d", time.Now().Unix())

	var prNumber int

	t.Cleanup(func() {
		if prNumber > 0 {
			closeCmd := fmt.Sprintf("gh pr close %d --repo %s/%s --delete-branch", prNumber, sandboxRepoOwner, sandboxRepoName)
			t.Logf("Cleaning up PR: %s", closeCmd)
			output, err := repo.RunGit("sh", "-c", closeCmd)
			if err != nil {
				t.Logf("Warning: failed to close PR %d: %v: %s", prNumber, err, output)
			}
		}

		deleteCmd := fmt.Sprintf("git push origin --delete %s", branchName)
		t.Logf("Cleaning up branch: %s", deleteCmd)
		output, err := repo.RunGit("sh", "-c", deleteCmd)
		if err != nil {
			t.Logf("Warning: failed to delete branch %s: %v: %s", branchName, err, output)
		}
	})

	workflowName := "test-feature-sandbox"
	description := "Add a Subtract function to the calculator that takes two integers and returns their difference"

	config := workflow.DefaultConfig(repo.Dir)
	config.Timeouts.Planning = 5 * time.Minute
	config.Timeouts.Implementation = 5 * time.Minute
	config.Timeouts.Refactoring = 5 * time.Minute
	config.CICheckTimeout = 10 * time.Minute
	config.SplitPR = false
	config.LogLevel = workflow.LogLevelVerbose

	orchestrator, err := workflow.NewOrchestratorWithConfig(config)
	require.NoError(t, err)

	confirmCalled := false
	orchestrator.SetConfirmFunc(func(plan *workflow.Plan) (bool, string, error) {
		confirmCalled = true
		assert.NotEmpty(t, plan.Summary, "plan summary should not be empty")
		assert.NotEmpty(t, plan.ContextType, "plan context type should not be empty")
		t.Logf("Plan received: %s", plan.Summary)
		return true, "", nil
	})

	ctx := context.Background()
	err = orchestrator.Start(ctx, workflowName, description, workflow.WorkflowTypeFeature)

	state, statusErr := orchestrator.Status(workflowName)
	require.NoError(t, statusErr)

	if state.WorktreePath != "" {
		prListOutput, ghErr := repo.RunGit("sh", "-c", fmt.Sprintf("cd %s && gh pr list --head $(git rev-parse --abbrev-ref HEAD) --json number --jq '.[0].number'", state.WorktreePath))
		if ghErr == nil && prListOutput != "" {
			fmt.Sscanf(prListOutput, "%d", &prNumber)
			if prNumber > 0 {
				t.Logf("Found PR #%d", prNumber)
			}
		}
	}

	if err != nil {
		t.Logf("Workflow error: %v", err)
		if state.CurrentPhase != workflow.PhaseCompleted && state.CurrentPhase != workflow.PhaseFailed {
			require.NoError(t, err, "workflow should reach completion or failure state")
		}
		t.Logf("Workflow ended in phase: %s (this is acceptable for E2E test validation)", state.CurrentPhase)
	}

	assert.True(t, confirmCalled, "confirm function should have been called")

	state, err = orchestrator.Status(workflowName)
	require.NoError(t, err)

	planningPhase := state.Phases[workflow.PhasePlanning]
	assert.Equal(t, workflow.StatusCompleted, planningPhase.Status, "planning phase should complete")
	assert.Greater(t, planningPhase.Attempts, 0, "planning phase should have at least one attempt")

	implPhase := state.Phases[workflow.PhaseImplementation]
	assert.Equal(t, workflow.StatusCompleted, implPhase.Status, "implementation phase should complete")

	refactorPhase := state.Phases[workflow.PhaseRefactoring]
	assert.Equal(t, workflow.StatusCompleted, refactorPhase.Status, "refactoring phase should complete")

	assert.Greater(t, prNumber, 0, "PR should be created")

	assert.NotEmpty(t, state.WorktreePath, "worktree path should be set")

	stateManager := workflow.NewStateManager(repo.Dir)
	plan, err := stateManager.LoadPlan(workflowName)
	require.NoError(t, err)
	assert.NotEmpty(t, plan.Summary, "saved plan should have summary")
	t.Logf("Final plan: %+v", plan)

	t.Logf("Workflow final phase: %s", state.CurrentPhase)
	if prNumber > 0 {
		t.Logf("PR URL: https://github.com/%s/%s/pull/%d", sandboxRepoOwner, sandboxRepoName, prNumber)
	}
}

// TestWorkflow_FixWorkflow_E2E tests a complete fix workflow with real CI checks
// using the sandbox repository. This test creates real commits, PRs, and waits for real CI.
func TestWorkflow_FixWorkflow_E2E(t *testing.T) {
	helpers.RequireClaude(t)
	helpers.RequireGit(t)
	helpers.RequireGHAuth(t)

	repo := helpers.CloneRepo(t, sandboxRepoURL)
	branchName := fmt.Sprintf("e2e-fix-%d", time.Now().Unix())

	var prNumber int

	t.Cleanup(func() {
		if prNumber > 0 {
			closeCmd := fmt.Sprintf("gh pr close %d --repo %s/%s --delete-branch", prNumber, sandboxRepoOwner, sandboxRepoName)
			t.Logf("Cleaning up PR: %s", closeCmd)
			output, err := repo.RunGit("sh", "-c", closeCmd)
			if err != nil {
				t.Logf("Warning: failed to close PR %d: %v: %s", prNumber, err, output)
			}
		}

		deleteCmd := fmt.Sprintf("git push origin --delete %s", branchName)
		t.Logf("Cleaning up branch: %s", deleteCmd)
		output, err := repo.RunGit("sh", "-c", deleteCmd)
		if err != nil {
			t.Logf("Warning: failed to delete branch %s: %v: %s", branchName, err, output)
		}
	})

	workflowName := "test-fix-sandbox"
	description := "Fix edge case in Add function when inputs are negative"

	config := workflow.DefaultConfig(repo.Dir)
	config.Timeouts.Planning = 5 * time.Minute
	config.Timeouts.Implementation = 5 * time.Minute
	config.Timeouts.Refactoring = 5 * time.Minute
	config.CICheckTimeout = 10 * time.Minute
	config.SplitPR = false
	config.LogLevel = workflow.LogLevelVerbose

	orchestrator, err := workflow.NewOrchestratorWithConfig(config)
	require.NoError(t, err)

	confirmCalled := false
	orchestrator.SetConfirmFunc(func(plan *workflow.Plan) (bool, string, error) {
		confirmCalled = true
		assert.NotEmpty(t, plan.Summary, "plan summary should not be empty")
		assert.NotEmpty(t, plan.ContextType, "plan context type should not be empty")
		t.Logf("Plan received: %s", plan.Summary)
		return true, "", nil
	})

	ctx := context.Background()
	err = orchestrator.Start(ctx, workflowName, description, workflow.WorkflowTypeFix)

	state, statusErr := orchestrator.Status(workflowName)
	require.NoError(t, statusErr)

	if state.WorktreePath != "" {
		prListOutput, ghErr := repo.RunGit("sh", "-c", fmt.Sprintf("cd %s && gh pr list --head $(git rev-parse --abbrev-ref HEAD) --json number --jq '.[0].number'", state.WorktreePath))
		if ghErr == nil && prListOutput != "" {
			fmt.Sscanf(prListOutput, "%d", &prNumber)
			if prNumber > 0 {
				t.Logf("Found PR #%d", prNumber)
			}
		}
	}

	if err != nil {
		t.Logf("Workflow error: %v", err)
		if state.CurrentPhase != workflow.PhaseCompleted && state.CurrentPhase != workflow.PhaseFailed {
			require.NoError(t, err, "workflow should reach completion or failure state")
		}
		t.Logf("Workflow ended in phase: %s (this is acceptable for E2E test validation)", state.CurrentPhase)
	}

	assert.True(t, confirmCalled, "confirm function should have been called")

	state, err = orchestrator.Status(workflowName)
	require.NoError(t, err)

	planningPhase := state.Phases[workflow.PhasePlanning]
	assert.Equal(t, workflow.StatusCompleted, planningPhase.Status, "planning phase should complete")
	assert.Greater(t, planningPhase.Attempts, 0, "planning phase should have at least one attempt")

	implPhase := state.Phases[workflow.PhaseImplementation]
	assert.Equal(t, workflow.StatusCompleted, implPhase.Status, "implementation phase should complete")

	refactorPhase := state.Phases[workflow.PhaseRefactoring]
	assert.Equal(t, workflow.StatusCompleted, refactorPhase.Status, "refactoring phase should complete")

	assert.Greater(t, prNumber, 0, "PR should be created")

	assert.NotEmpty(t, state.WorktreePath, "worktree path should be set")

	stateManager := workflow.NewStateManager(repo.Dir)
	plan, err := stateManager.LoadPlan(workflowName)
	require.NoError(t, err)
	assert.NotEmpty(t, plan.Summary, "saved plan should have summary")
	t.Logf("Final plan: %+v", plan)

	t.Logf("Workflow final phase: %s", state.CurrentPhase)
	if prNumber > 0 {
		t.Logf("PR URL: https://github.com/%s/%s/pull/%d", sandboxRepoOwner, sandboxRepoName, prNumber)
	}
}

// TestWorkflow_ResumeWorkflow_E2E tests resuming a workflow after interruption.
// This test starts a workflow, interrupts it during planning, and then resumes it.
func TestWorkflow_ResumeWorkflow_E2E(t *testing.T) {
	helpers.RequireClaude(t)
	helpers.RequireGit(t)
	helpers.RequireGHAuth(t)

	repo := helpers.CloneRepo(t, sandboxRepoURL)
	branchName := fmt.Sprintf("e2e-resume-%d", time.Now().Unix())

	var prNumber int

	t.Cleanup(func() {
		if prNumber > 0 {
			closeCmd := fmt.Sprintf("gh pr close %d --repo %s/%s --delete-branch", prNumber, sandboxRepoOwner, sandboxRepoName)
			t.Logf("Cleaning up PR: %s", closeCmd)
			output, err := repo.RunGit("sh", "-c", closeCmd)
			if err != nil {
				t.Logf("Warning: failed to close PR %d: %v: %s", prNumber, err, output)
			}
		}

		deleteCmd := fmt.Sprintf("git push origin --delete %s", branchName)
		t.Logf("Cleaning up branch: %s", deleteCmd)
		output, err := repo.RunGit("sh", "-c", deleteCmd)
		if err != nil {
			t.Logf("Warning: failed to delete branch %s: %v: %s", branchName, err, output)
		}
	})

	workflowName := "test-resume-sandbox"
	description := "Add a Multiply function to the calculator that takes two integers and returns their product"

	config := workflow.DefaultConfig(repo.Dir)
	config.Timeouts.Planning = 5 * time.Minute
	config.Timeouts.Implementation = 5 * time.Minute
	config.Timeouts.Refactoring = 5 * time.Minute
	config.CICheckTimeout = 10 * time.Minute
	config.SplitPR = false
	config.LogLevel = workflow.LogLevelVerbose

	orchestrator, err := workflow.NewOrchestratorWithConfig(config)
	require.NoError(t, err)

	confirmCalled := false
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	orchestrator.SetConfirmFunc(func(plan *workflow.Plan) (bool, string, error) {
		confirmCalled = true
		assert.NotEmpty(t, plan.Summary, "plan summary should not be empty")
		assert.NotEmpty(t, plan.ContextType, "plan context type should not be empty")
		t.Logf("Plan received during initial workflow: %s", plan.Summary)
		cancel()
		return false, "Interrupting workflow for resume test", nil
	})

	err = orchestrator.Start(ctx, workflowName, description, workflow.WorkflowTypeFeature)
	assert.Error(t, err, "workflow should be interrupted")
	assert.True(t, confirmCalled, "confirm function should have been called before interruption")

	state, err := orchestrator.Status(workflowName)
	require.NoError(t, err)
	assert.Equal(t, workflow.PhasePlanning, state.CurrentPhase, "workflow should be in planning phase after interruption")
	t.Logf("Workflow interrupted in phase: %s", state.CurrentPhase)

	t.Logf("Resuming workflow...")

	orchestrator2, err := workflow.NewOrchestratorWithConfig(config)
	require.NoError(t, err)

	resumeConfirmCalled := false
	orchestrator2.SetConfirmFunc(func(plan *workflow.Plan) (bool, string, error) {
		resumeConfirmCalled = true
		assert.NotEmpty(t, plan.Summary, "plan summary should not be empty")
		assert.NotEmpty(t, plan.ContextType, "plan context type should not be empty")
		t.Logf("Plan received during resume: %s", plan.Summary)
		return true, "", nil
	})

	resumeCtx := context.Background()
	err = orchestrator2.Resume(resumeCtx, workflowName)

	state, statusErr := orchestrator2.Status(workflowName)
	require.NoError(t, statusErr)

	if state.WorktreePath != "" {
		prListOutput, ghErr := repo.RunGit("sh", "-c", fmt.Sprintf("cd %s && gh pr list --head $(git rev-parse --abbrev-ref HEAD) --json number --jq '.[0].number'", state.WorktreePath))
		if ghErr == nil && prListOutput != "" {
			fmt.Sscanf(prListOutput, "%d", &prNumber)
			if prNumber > 0 {
				t.Logf("Found PR #%d", prNumber)
			}
		}
	}

	if err != nil {
		t.Logf("Workflow error: %v", err)
		if state.CurrentPhase != workflow.PhaseCompleted && state.CurrentPhase != workflow.PhaseFailed {
			require.NoError(t, err, "workflow should reach completion or failure state")
		}
		t.Logf("Workflow ended in phase: %s (this is acceptable for E2E test validation)", state.CurrentPhase)
	}

	assert.True(t, resumeConfirmCalled, "confirm function should have been called during resume")

	state, err = orchestrator2.Status(workflowName)
	require.NoError(t, err)

	planningPhase := state.Phases[workflow.PhasePlanning]
	assert.Equal(t, workflow.StatusCompleted, planningPhase.Status, "planning phase should complete")
	assert.Greater(t, planningPhase.Attempts, 0, "planning phase should have at least one attempt")

	implPhase := state.Phases[workflow.PhaseImplementation]
	assert.Equal(t, workflow.StatusCompleted, implPhase.Status, "implementation phase should complete")

	refactorPhase := state.Phases[workflow.PhaseRefactoring]
	assert.Equal(t, workflow.StatusCompleted, refactorPhase.Status, "refactoring phase should complete")

	assert.Greater(t, prNumber, 0, "PR should be created")

	assert.NotEmpty(t, state.WorktreePath, "worktree path should be set")

	stateManager := workflow.NewStateManager(repo.Dir)
	plan, err := stateManager.LoadPlan(workflowName)
	require.NoError(t, err)
	assert.NotEmpty(t, plan.Summary, "saved plan should have summary")
	t.Logf("Final plan: %+v", plan)

	t.Logf("Workflow final phase: %s", state.CurrentPhase)
	if prNumber > 0 {
		t.Logf("PR URL: https://github.com/%s/%s/pull/%d", sandboxRepoOwner, sandboxRepoName, prNumber)
	}
}
