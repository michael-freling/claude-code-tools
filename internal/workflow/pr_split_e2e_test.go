//go:build e2e
// +build e2e

package workflow

import (
	"context"
	"os"
	"testing"

	"github.com/michael-freling/claude-code-tools/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPRSplitManager_E2E tests the PR split functionality with real git and gh commands.
//
// Run with:
//
//	RUN_E2E_TESTS=true TEST_REPO_DIR=/path/to/repo go test -tags=e2e -v ./internal/workflow/... -run TestPRSplitManager_E2E
//
// Prerequisites:
//  1. A branch with commits to split (set via TEST_SOURCE_BRANCH, defaults to current branch)
//  2. GitHub authentication configured (gh auth login)
//  3. TEST_REPO_DIR pointing to the repository root
func TestPRSplitManager_E2E(t *testing.T) {
	// Skip if not in e2e mode
	if os.Getenv("RUN_E2E_TESTS") != "true" {
		t.Skip("Skipping E2E test. Set RUN_E2E_TESTS=true to run.")
	}

	ctx := context.Background()
	dir := os.Getenv("TEST_REPO_DIR")
	if dir == "" {
		t.Fatal("TEST_REPO_DIR environment variable must be set")
	}

	baseBranch := os.Getenv("TEST_BASE_BRANCH")
	if baseBranch == "" {
		baseBranch = "main"
	}

	sourceBranch := os.Getenv("TEST_SOURCE_BRANCH")
	if sourceBranch == "" {
		t.Fatal("TEST_SOURCE_BRANCH environment variable must be set")
	}

	// Create real runners
	cmdRunner := command.NewRunner()
	gitRunner := command.NewGitRunner(cmdRunner)
	ghRunner := command.NewGhRunner(cmdRunner)

	// Create the manager
	manager := NewPRSplitManager(gitRunner, ghRunner)

	// Get commits from the source branch
	commits, err := gitRunner.GetCommits(ctx, dir, baseBranch)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(commits), 2, "Need at least 2 commits to split")

	t.Logf("Found %d commits to split:", len(commits))
	for _, c := range commits {
		t.Logf("  - %s: %s", c.Hash[:7], c.Subject)
	}

	// Create a split plan - put each commit in its own child PR
	childPRs := make([]ChildPRPlan, len(commits))
	for i, commit := range commits {
		childPRs[i] = ChildPRPlan{
			Title:       "E2E Test: Child " + string(rune('1'+i)) + " - " + commit.Subject,
			Description: "This child PR contains commit: " + commit.Hash[:7],
			Commits:     []string{commit.Hash},
		}
	}

	plan := &PRSplitPlan{
		Strategy:    SplitByCommits,
		ParentTitle: "E2E Test: PR Split Parent",
		ParentDesc:  "This is a test parent PR created by E2E test",
		ChildPRs:    childPRs,
		Summary:     "Split into child PRs for E2E testing",
	}

	// Execute the split
	t.Log("Executing PR split...")
	result, err := manager.ExecuteSplit(ctx, dir, plan, sourceBranch, baseBranch)
	require.NoError(t, err)

	t.Logf("PR Split successful!")
	t.Logf("Parent PR: #%d - %s", result.ParentPR.Number, result.ParentPR.URL)
	for i, child := range result.ChildPRs {
		t.Logf("Child PR %d: #%d - %s", i+1, child.Number, child.URL)
	}
	t.Logf("Branches created: %v", result.BranchNames)

	// Verify the results
	assert.Greater(t, result.ParentPR.Number, 0)
	assert.NotEmpty(t, result.ParentPR.URL)
	assert.Len(t, result.ChildPRs, len(commits))
	assert.Len(t, result.BranchNames, 1+len(commits)) // parent + children

	// Print cleanup instructions
	t.Logf("\nTo clean up, run:")
	t.Logf("  gh pr close %d --delete-branch", result.ParentPR.Number)
	for _, child := range result.ChildPRs {
		t.Logf("  gh pr close %d --delete-branch", child.Number)
	}
}
