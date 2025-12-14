package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/michael-freling/claude-code-tools/internal/command"
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

func TestNewWorktreeManagerWithRunner(t *testing.T) {
	tests := []struct {
		name      string
		baseDir   string
		gitRunner command.GitRunner
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

func TestWorktreeManager_CreateWorktree_PreCommitSetup(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		setupMock    func(*MockGitRunner)
		setupFS      func(t *testing.T, worktreePath string)
		wantErr      bool
	}{
		{
			name:         "succeeds even if pre-commit setup would fail",
			workflowName: "test-workflow",
			setupMock: func(m *MockGitRunner) {
				m.On("WorktreeAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
					return filepath.Base(path) == "test-workflow"
				}), "workflow/test-workflow").Return(nil)
			},
			setupFS: func(t *testing.T, worktreePath string) {
				// Create pre-commit config file to trigger setup attempt
				// The setup will fail because pre-commit command isn't available in test
				// But worktree creation should still succeed
				configPath := filepath.Join(worktreePath, ".pre-commit-config.yaml")
				err := os.WriteFile(configPath, []byte("repos: []"), 0644)
				require.NoError(t, err)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGitRunner := new(MockGitRunner)
			tt.setupMock(mockGitRunner)

			baseDir, err := os.MkdirTemp("", "worktree-precommit-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(baseDir)

			worktreesDir := filepath.Join(baseDir, "worktrees")
			err = os.MkdirAll(worktreesDir, 0755)
			require.NoError(t, err)

			manager := NewWorktreeManagerWithRunner(filepath.Join(baseDir, "repo"), mockGitRunner)

			// Mock the WorktreeAdd to create the worktree directory
			mockGitRunner.On("WorktreeAdd", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					worktreePath := args.Get(2).(string)
					err := os.MkdirAll(worktreePath, 0755)
					require.NoError(t, err)
					gitDir := filepath.Join(worktreePath, ".git")
					err = os.Mkdir(gitDir, 0755)
					require.NoError(t, err)
					if tt.setupFS != nil {
						tt.setupFS(t, worktreePath)
					}
				}).Return(nil)

			got, err := manager.CreateWorktree(tt.workflowName)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Contains(t, got, tt.workflowName)
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

func TestWorktreeManager_setupPreCommitHooks(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) (worktreePath string, cleanup func())
		wantErr     bool
		errContains string
	}{
		{
			name: "skips when config doesn't exist",
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "precommit-test-*")
				require.NoError(t, err)
				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			wantErr: false,
		},
		{
			name: "skips when pre-commit command isn't available",
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "precommit-test-*")
				require.NoError(t, err)
				configPath := filepath.Join(tmpDir, ".pre-commit-config.yaml")
				err = os.WriteFile(configPath, []byte("repos: []"), 0644)
				require.NoError(t, err)
				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			wantErr: false,
		},
		{
			name: "succeeds when config exists and pre-commit is available",
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "precommit-test-*")
				require.NoError(t, err)

				configPath := filepath.Join(tmpDir, ".pre-commit-config.yaml")
				err = os.WriteFile(configPath, []byte("repos: []"), 0644)
				require.NoError(t, err)

				mockPreCommitPath := filepath.Join(tmpDir, "bin")
				err = os.MkdirAll(mockPreCommitPath, 0755)
				require.NoError(t, err)

				preCommitScript := filepath.Join(mockPreCommitPath, "pre-commit")
				script := `#!/bin/bash
exit 0
`
				err = os.WriteFile(preCommitScript, []byte(script), 0755)
				require.NoError(t, err)

				oldPath := os.Getenv("PATH")
				err = os.Setenv("PATH", mockPreCommitPath+":"+oldPath)
				require.NoError(t, err)

				return tmpDir, func() {
					os.Setenv("PATH", oldPath)
					os.RemoveAll(tmpDir)
				}
			},
			wantErr: false,
		},
		{
			name: "fails when pre-commit install fails",
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "precommit-test-*")
				require.NoError(t, err)

				configPath := filepath.Join(tmpDir, ".pre-commit-config.yaml")
				err = os.WriteFile(configPath, []byte("repos: []"), 0644)
				require.NoError(t, err)

				mockPreCommitPath := filepath.Join(tmpDir, "bin")
				err = os.MkdirAll(mockPreCommitPath, 0755)
				require.NoError(t, err)

				preCommitScript := filepath.Join(mockPreCommitPath, "pre-commit")
				script := `#!/bin/bash
exit 1
`
				err = os.WriteFile(preCommitScript, []byte(script), 0755)
				require.NoError(t, err)

				oldPath := os.Getenv("PATH")
				err = os.Setenv("PATH", mockPreCommitPath+":"+oldPath)
				require.NoError(t, err)

				return tmpDir, func() {
					os.Setenv("PATH", oldPath)
					os.RemoveAll(tmpDir)
				}
			},
			wantErr:     true,
			errContains: "failed to run pre-commit install",
		},
		{
			name: "fails when pre-commit install --hook-type pre-push fails",
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "precommit-test-*")
				require.NoError(t, err)

				configPath := filepath.Join(tmpDir, ".pre-commit-config.yaml")
				err = os.WriteFile(configPath, []byte("repos: []"), 0644)
				require.NoError(t, err)

				mockPreCommitPath := filepath.Join(tmpDir, "bin")
				err = os.MkdirAll(mockPreCommitPath, 0755)
				require.NoError(t, err)

				preCommitScript := filepath.Join(mockPreCommitPath, "pre-commit")
				script := `#!/bin/bash
if [[ "$1" == "install" ]] && [[ "$2" == "--hook-type" ]]; then
    exit 1
fi
exit 0
`
				err = os.WriteFile(preCommitScript, []byte(script), 0755)
				require.NoError(t, err)

				oldPath := os.Getenv("PATH")
				err = os.Setenv("PATH", mockPreCommitPath+":"+oldPath)
				require.NoError(t, err)

				return tmpDir, func() {
					os.Setenv("PATH", oldPath)
					os.RemoveAll(tmpDir)
				}
			},
			wantErr:     true,
			errContains: "failed to run pre-commit install --hook-type pre-push",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktreePath, cleanup := tt.setup(t)
			defer cleanup()

			manager := &worktreeManager{
				baseDir:   "/test/repo",
				gitRunner: &MockGitRunner{},
			}

			err := manager.setupPreCommitHooks(worktreePath)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
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
