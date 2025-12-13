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
