package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestNewWorktreeManagerWithRunner(t *testing.T) {
	tests := []struct {
		name      string
		baseDir   string
		gitRunner GitRunner
	}{
		{
			name:      "creates manager with mock runner",
			baseDir:   "/test/repo",
			gitRunner: &MockGitRunner{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewWorktreeManagerWithRunner(tt.baseDir, tt.gitRunner)
			require.NotNil(t, got)

			manager, ok := got.(*worktreeManager)
			require.True(t, ok)
			assert.Equal(t, tt.baseDir, manager.baseDir)
			assert.Equal(t, tt.gitRunner, manager.gitRunner)
		})
	}
}

func TestWorktreeManager_CreateWorktree_WithMocks(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		setupMock    func(*MockGitRunner)
		setupFS      func(t *testing.T) (worktreesDir string, cleanup func())
		want         string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "creates worktree successfully when it doesn't exist",
			workflowName: "test-workflow",
			setupMock: func(m *MockGitRunner) {
				m.On("WorktreeAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
					return filepath.Base(path) == "test-workflow"
				}), "workflow/test-workflow").Return(nil)
			},
			setupFS: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "worktree-test-*")
				require.NoError(t, err)
				worktreesDir := filepath.Join(tmpDir, "worktrees")
				err = os.MkdirAll(worktreesDir, 0755)
				require.NoError(t, err)
				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			want: "test-workflow",
		},
		{
			name:         "returns existing worktree path when worktree exists",
			workflowName: "existing-workflow",
			setupMock:    func(m *MockGitRunner) {},
			setupFS: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "worktree-test-*")
				require.NoError(t, err)
				worktreesDir := filepath.Join(tmpDir, "worktrees")
				err = os.MkdirAll(worktreesDir, 0755)
				require.NoError(t, err)
				existingPath := filepath.Join(worktreesDir, "existing-workflow")
				err = os.MkdirAll(existingPath, 0755)
				require.NoError(t, err)
				gitDir := filepath.Join(existingPath, ".git")
				err = os.Mkdir(gitDir, 0755)
				require.NoError(t, err)
				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			want: "existing-workflow",
		},
		{
			name:         "fails when git worktree add fails",
			workflowName: "fail-workflow",
			setupMock: func(m *MockGitRunner) {
				m.On("WorktreeAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
					return filepath.Base(path) == "fail-workflow"
				}), "workflow/fail-workflow").Return(fmt.Errorf("branch workflow/fail-workflow already exists"))
			},
			setupFS: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "worktree-test-*")
				require.NoError(t, err)
				worktreesDir := filepath.Join(tmpDir, "worktrees")
				err = os.MkdirAll(worktreesDir, 0755)
				require.NoError(t, err)
				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			wantErr:     true,
			errContains: "already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGitRunner := new(MockGitRunner)
			tt.setupMock(mockGitRunner)

			baseDir, cleanup := tt.setupFS(t)
			defer cleanup()

			manager := NewWorktreeManagerWithRunner(filepath.Join(baseDir, "repo"), mockGitRunner)

			got, err := manager.CreateWorktree(tt.workflowName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				mockGitRunner.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			assert.Contains(t, got, tt.want)
			mockGitRunner.AssertExpectations(t)
		})
	}
}

func TestWorktreeManager_CreateWorktree_MkdirAll(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		setupMock    func(*MockGitRunner)
		setupFS      func(t *testing.T) (baseDir string, cleanup func())
		want         string
		wantErr      bool
	}{
		{
			name:         "creates worktrees directory when it doesn't exist",
			workflowName: "test-workflow",
			setupMock: func(m *MockGitRunner) {
				m.On("WorktreeAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
					return filepath.Base(path) == "test-workflow"
				}), "workflow/test-workflow").Return(nil)
			},
			setupFS: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "worktree-mkdir-test-*")
				require.NoError(t, err)
				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			want: "test-workflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGitRunner := new(MockGitRunner)
			tt.setupMock(mockGitRunner)

			baseDir, cleanup := tt.setupFS(t)
			defer cleanup()

			worktreesDir := filepath.Join(baseDir, "worktrees")
			assert.False(t, dirExists(worktreesDir), "worktrees directory should not exist before CreateWorktree")

			manager := NewWorktreeManagerWithRunner(filepath.Join(baseDir, "repo"), mockGitRunner)

			got, err := manager.CreateWorktree(tt.workflowName)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Contains(t, got, tt.want)
			assert.True(t, dirExists(worktreesDir), "worktrees directory should be created")
			mockGitRunner.AssertExpectations(t)
		})
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func TestWorktreeManager_DeleteWorktree_WithMocks(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setupMock   func(*MockGitRunner)
		setupFS     func(t *testing.T) (worktreePath string, cleanup func())
		wantErr     bool
		errContains string
	}{
		{
			name: "deletes worktree successfully",
			setupMock: func(m *MockGitRunner) {
				m.On("WorktreeRemove", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			setupFS: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "worktree-delete-")
				require.NoError(t, err)
				gitDir := filepath.Join(tmpDir, ".git")
				err = os.Mkdir(gitDir, 0755)
				require.NoError(t, err)
				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
		},
		{
			name:      "succeeds when worktree does not exist",
			setupMock: func(m *MockGitRunner) {},
			setupFS: func(t *testing.T) (string, func()) {
				return "/tmp/non-existent-worktree-12345", func() {}
			},
		},
		{
			name: "fails when git worktree remove fails",
			setupMock: func(m *MockGitRunner) {
				m.On("WorktreeRemove", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("failed to remove worktree"))
			},
			setupFS: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "worktree-delete-fail-")
				require.NoError(t, err)
				gitDir := filepath.Join(tmpDir, ".git")
				err = os.Mkdir(gitDir, 0755)
				require.NoError(t, err)
				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			wantErr:     true,
			errContains: "failed to remove worktree",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGitRunner := new(MockGitRunner)
			tt.setupMock(mockGitRunner)

			manager := NewWorktreeManagerWithRunner("/test/repo", mockGitRunner)

			worktreePath, cleanup := tt.setupFS(t)
			defer cleanup()

			err := manager.DeleteWorktree(worktreePath)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				mockGitRunner.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			mockGitRunner.AssertExpectations(t)
		})
	}
}
