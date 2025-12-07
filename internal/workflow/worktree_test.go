package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorktreeManager(t *testing.T) {
	baseDir := "/tmp/test-workflow"
	wm := NewWorktreeManager(baseDir)

	assert.NotNil(t, wm)
}

func TestWorktreeManager_CreateWorktree(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "fails with empty workflow name",
			workflowName: "",
			wantErr:      true,
			errContains:  "workflow name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wm := NewWorktreeManager("/tmp/test")

			path, err := wm.CreateWorktree(tt.workflowName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Empty(t, path)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, path)
		})
	}
}

func TestWorktreeManager_WorktreeExists(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		setup func(t *testing.T) string
		want  bool
	}{
		{
			name: "returns false for empty path",
			path: "",
			want: false,
		},
		{
			name: "returns false for non-existent path",
			path: "/tmp/non-existent-worktree-path-12345",
			want: false,
		},
		{
			name: "returns false for file instead of directory",
			setup: func(t *testing.T) string {
				f, err := os.CreateTemp("", "worktree-test-file-*")
				require.NoError(t, err)
				f.Close()
				return f.Name()
			},
			want: false,
		},
		{
			name: "returns false for directory without .git",
			setup: func(t *testing.T) string {
				dir, err := os.MkdirTemp("", "worktree-test-dir-*")
				require.NoError(t, err)
				return dir
			},
			want: false,
		},
		{
			name: "returns true for directory with .git",
			setup: func(t *testing.T) string {
				dir, err := os.MkdirTemp("", "worktree-test-git-*")
				require.NoError(t, err)
				gitDir := filepath.Join(dir, ".git")
				err = os.Mkdir(gitDir, 0755)
				require.NoError(t, err)
				return dir
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wm := NewWorktreeManager("/tmp/test")

			path := tt.path
			if tt.setup != nil {
				path = tt.setup(t)
				defer os.RemoveAll(path)
			}

			got := wm.WorktreeExists(path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWorktreeManager_DeleteWorktree(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantErr     bool
		errContains string
	}{
		{
			name:        "fails with empty path",
			path:        "",
			wantErr:     true,
			errContains: "worktree path cannot be empty",
		},
		{
			name:    "succeeds when worktree does not exist",
			path:    "/tmp/non-existent-worktree-12345",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wm := NewWorktreeManager("/tmp/test")

			err := wm.DeleteWorktree(tt.path)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

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
