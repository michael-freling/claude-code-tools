//go:build integration

package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorktreeManager_CreateWorktree_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	// Create a temporary git repository
	tmpDir, err := os.MkdirTemp("", "worktree-integration-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)

	// Create an initial commit
	dummyFile := filepath.Join(tmpDir, "README.md")
	err = os.WriteFile(dummyFile, []byte("# Test"), 0644)
	require.NoError(t, err)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)

	// Create worktree manager
	wm := NewWorktreeManager(tmpDir)

	// Test creating a worktree
	worktreePath, err := wm.CreateWorktree("test-workflow-1")
	require.NoError(t, err)
	defer os.RemoveAll(worktreePath)

	// Verify worktree was created
	assert.True(t, wm.WorktreeExists(worktreePath))

	// Verify worktree is at expected location
	expectedPath := filepath.Join(tmpDir, "..", "worktrees", "test-workflow-1")
	absExpected, _ := filepath.Abs(expectedPath)
	assert.Equal(t, absExpected, worktreePath)

	// Test that creating the same worktree again returns existing path
	worktreePath2, err := wm.CreateWorktree("test-workflow-1")
	require.NoError(t, err)
	assert.Equal(t, worktreePath, worktreePath2)

	// Test deleting the worktree
	err = wm.DeleteWorktree(worktreePath)
	require.NoError(t, err)
	assert.False(t, wm.WorktreeExists(worktreePath))
}

func TestOrchestrator_Delete_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tests := []struct {
		name                string
		createWorktree      bool
		worktreePathInState string
		wantWorktreeDeleted bool
	}{
		{
			name:                "deletes workflow with worktree",
			createWorktree:      true,
			worktreePathInState: "",
			wantWorktreeDeleted: true,
		},
		{
			name:                "deletes workflow without worktree",
			createWorktree:      false,
			worktreePathInState: "",
			wantWorktreeDeleted: false,
		},
		{
			name:                "deletes workflow with invalid worktree path",
			createWorktree:      false,
			worktreePathInState: "/nonexistent/path",
			wantWorktreeDeleted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitRepoDir, err := os.MkdirTemp("", "orchestrator-delete-integration-*")
			require.NoError(t, err)
			defer os.RemoveAll(gitRepoDir)

			cmd := exec.Command("git", "init")
			cmd.Dir = gitRepoDir
			err = cmd.Run()
			require.NoError(t, err)

			cmd = exec.Command("git", "config", "user.email", "test@test.com")
			cmd.Dir = gitRepoDir
			err = cmd.Run()
			require.NoError(t, err)

			cmd = exec.Command("git", "config", "user.name", "Test User")
			cmd.Dir = gitRepoDir
			err = cmd.Run()
			require.NoError(t, err)

			dummyFile := filepath.Join(gitRepoDir, "README.md")
			err = os.WriteFile(dummyFile, []byte("# Test"), 0644)
			require.NoError(t, err)

			cmd = exec.Command("git", "add", ".")
			cmd.Dir = gitRepoDir
			err = cmd.Run()
			require.NoError(t, err)

			cmd = exec.Command("git", "commit", "-m", "Initial commit")
			cmd.Dir = gitRepoDir
			err = cmd.Run()
			require.NoError(t, err)

			workflowDir := filepath.Join(gitRepoDir, ".claude", "workflow")
			err = os.MkdirAll(workflowDir, 0755)
			require.NoError(t, err)

			config := DefaultConfig(workflowDir)

			orchestrator, err := NewOrchestratorWithConfig(config)
			require.NoError(t, err)

			workflowName := "test-delete-workflow"

			var worktreePath string
			if tt.createWorktree {
				wm := NewWorktreeManager(workflowDir)
				worktreePath, err = wm.CreateWorktree(workflowName)
				require.NoError(t, err)

				assert.True(t, wm.WorktreeExists(worktreePath))
			}

			stateManager := NewStateManager(workflowDir)
			state, err := stateManager.InitState(workflowName, "test workflow", WorkflowTypeFeature)
			require.NoError(t, err)

			if worktreePath != "" {
				state.WorktreePath = worktreePath
				err = stateManager.SaveState(workflowName, state)
				require.NoError(t, err)
			} else if tt.worktreePathInState != "" {
				state.WorktreePath = tt.worktreePathInState
				err = stateManager.SaveState(workflowName, state)
				require.NoError(t, err)
			}

			assert.True(t, stateManager.WorkflowExists(workflowName))

			err = orchestrator.Delete(workflowName)
			require.NoError(t, err)

			assert.False(t, stateManager.WorkflowExists(workflowName))

			if tt.wantWorktreeDeleted && worktreePath != "" {
				wm := NewWorktreeManager(workflowDir)
				assert.False(t, wm.WorktreeExists(worktreePath), "worktree should be deleted")

				_, statErr := os.Stat(worktreePath)
				assert.True(t, os.IsNotExist(statErr), "worktree directory should not exist")
			}
		})
	}
}
