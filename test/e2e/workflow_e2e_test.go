//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/michael-freling/claude-code-tools/internal/workflow"
	"github.com/michael-freling/claude-code-tools/test/e2e/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkflow_SimpleFeature_E2E tests a simple feature workflow with real Claude.
// This test uses the REAL Claude CLI to verify end-to-end workflow functionality.
// It is skipped when Claude is not available (e.g., in CI).
func TestWorkflow_SimpleFeature_E2E(t *testing.T) {
	helpers.RequireClaude(t)
	helpers.RequireGit(t)

	// Create a real temp git repo
	repo := helpers.NewTempRepo(t)
	require.NoError(t, repo.CreateFile("main.go", "package main\n\nfunc main() {\n}\n"))
	require.NoError(t, repo.Commit("Initial commit"))

	workflowName := "test-simple-feature"
	// Keep description SIMPLE to minimize Claude execution time and cost
	description := "Add a hello function that returns the string 'hello'"

	// Create config with REAL Claude CLI (no mock)
	config := workflow.DefaultConfig(repo.Dir)
	// Use generous timeouts for real Claude (can be slow)
	config.Timeouts.Planning = 5 * time.Minute
	config.Timeouts.Implementation = 5 * time.Minute
	config.Timeouts.Refactoring = 5 * time.Minute
	config.SplitPR = false
	config.LogLevel = workflow.LogLevelVerbose

	// Mock CI checker since temp repos don't have real CI
	mockCI := &mockCIChecker{
		result: &workflow.CIResult{
			Passed: true,
			Status: "success",
		},
	}

	// Create orchestrator with REAL Claude executor
	orchestrator, err := workflow.NewTestOrchestrator(config, func(workingDir string, checkInterval time.Duration, commandTimeout time.Duration) workflow.CIChecker {
		return mockCI
	})
	require.NoError(t, err)

	// Auto-confirm to avoid interactive blocking
	confirmCalled := false
	orchestrator.SetConfirmFunc(func(plan *workflow.Plan) (bool, string, error) {
		confirmCalled = true
		// Verify we got a real plan from Claude
		assert.NotEmpty(t, plan.Summary, "plan summary should not be empty")
		assert.NotEmpty(t, plan.ContextType, "plan context type should not be empty")
		t.Logf("Plan received: %s", plan.Summary)
		return true, "", nil
	})

	// Run the workflow with REAL Claude
	ctx := context.Background()
	err = orchestrator.Start(ctx, workflowName, description, workflow.WorkflowTypeFeature)
	require.NoError(t, err, "workflow should complete successfully with real Claude")

	// Verify workflow completed
	assert.True(t, confirmCalled, "confirm function should have been called")

	state, err := orchestrator.Status(workflowName)
	require.NoError(t, err)
	assert.Equal(t, workflow.PhaseCompleted, state.CurrentPhase, "workflow should reach completed phase")

	// Verify planning phase completed with real Claude
	planningPhase := state.Phases[workflow.PhasePlanning]
	assert.Equal(t, workflow.StatusCompleted, planningPhase.Status, "planning phase should complete")
	assert.Greater(t, planningPhase.Attempts, 0, "planning phase should have at least one attempt")

	// Verify implementation phase completed
	implPhase := state.Phases[workflow.PhaseImplementation]
	assert.Equal(t, workflow.StatusCompleted, implPhase.Status, "implementation phase should complete")

	// Verify refactoring phase completed
	refactorPhase := state.Phases[workflow.PhaseRefactoring]
	assert.Equal(t, workflow.StatusCompleted, refactorPhase.Status, "refactoring phase should complete")

	// Verify worktree was created
	assert.NotEmpty(t, state.WorktreePath, "worktree path should be set")

	// Verify plan was saved
	stateManager := workflow.NewStateManager(repo.Dir)
	plan, err := stateManager.LoadPlan(workflowName)
	require.NoError(t, err)
	assert.NotEmpty(t, plan.Summary, "saved plan should have summary")
	t.Logf("Final plan: %+v", plan)
}

// TestWorkflow_PlanningOnly_E2E tests only the planning phase to save time and cost.
// This allows testing Claude integration without running the full workflow.
func TestWorkflow_PlanningOnly_E2E(t *testing.T) {
	t.Skip("Planning-only test - implement if needed for faster testing")
}

type mockCIChecker struct {
	result    *workflow.CIResult
	err       error
	onCall    func(int)
	callCount int
	checkFunc func() (*workflow.CIResult, error)
}

func (m *mockCIChecker) CheckCI(ctx context.Context, prNumber int) (*workflow.CIResult, error) {
	m.callCount++
	if m.onCall != nil {
		m.onCall(m.callCount)
	}
	if m.checkFunc != nil {
		return m.checkFunc()
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func (m *mockCIChecker) WaitForCI(ctx context.Context, prNumber int, timeout time.Duration) (*workflow.CIResult, error) {
	return m.CheckCI(ctx, prNumber)
}

func (m *mockCIChecker) WaitForCIWithOptions(ctx context.Context, prNumber int, timeout time.Duration, opts workflow.CheckCIOptions) (*workflow.CIResult, error) {
	return m.CheckCI(ctx, prNumber)
}

func (m *mockCIChecker) WaitForCIWithProgress(ctx context.Context, prNumber int, timeout time.Duration, opts workflow.CheckCIOptions, onProgress workflow.CIProgressCallback) (*workflow.CIResult, error) {
	return m.CheckCI(ctx, prNumber)
}
