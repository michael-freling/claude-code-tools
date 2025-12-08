package workflow

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewGitRunner(t *testing.T) {
	tests := []struct {
		name   string
		runner CommandRunner
	}{
		{
			name:   "creates git runner with command runner",
			runner: &MockCommandRunner{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewGitRunner(tt.runner)
			require.NotNil(t, got)

			gitRunner, ok := got.(*gitRunner)
			require.True(t, ok)
			assert.Equal(t, tt.runner, gitRunner.runner)
		})
	}
}

func TestGitRunner_GetCurrentBranch(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		setupMock   func(*MockCommandRunner)
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name: "returns current branch successfully",
			dir:  "/test/repo",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "rev-parse", "--abbrev-ref", "HEAD").
					Return("feature-branch", "", nil)
			},
			want: "feature-branch",
		},
		{
			name: "returns main branch",
			dir:  "/test/repo",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "rev-parse", "--abbrev-ref", "HEAD").
					Return("main", "", nil)
			},
			want: "main",
		},
		{
			name: "trims whitespace from branch name",
			dir:  "/test/repo",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "rev-parse", "--abbrev-ref", "HEAD").
					Return("  feature-branch  \n", "", nil)
			},
			want: "feature-branch",
		},
		{
			name: "returns error when git command fails",
			dir:  "/test/repo",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "rev-parse", "--abbrev-ref", "HEAD").
					Return("", "fatal: not a git repository", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to get current branch",
		},
		{
			name: "handles empty directory path",
			dir:  "",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "", "git", "rev-parse", "--abbrev-ref", "HEAD").
					Return("main", "", nil)
			},
			want: "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			got, err := gitRunner.GetCurrentBranch(ctx, tt.dir)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Empty(t, got)
				mockRunner.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			mockRunner.AssertExpectations(t)
		})
	}
}

func TestGitRunner_Push(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		branch      string
		setupMock   func(*MockCommandRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:   "pushes branch successfully",
			dir:    "/test/repo",
			branch: "feature-branch",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "push", "-u", "origin", "feature-branch").
					Return("", "", nil)
			},
		},
		{
			name:   "pushes main branch successfully",
			dir:    "/test/repo",
			branch: "main",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "push", "-u", "origin", "main").
					Return("", "", nil)
			},
		},
		{
			name:        "fails when branch name is empty",
			dir:         "/test/repo",
			branch:      "",
			setupMock:   func(m *MockCommandRunner) {},
			wantErr:     true,
			errContains: "branch name cannot be empty",
		},
		{
			name:   "returns error when push fails",
			dir:    "/test/repo",
			branch: "feature-branch",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "push", "-u", "origin", "feature-branch").
					Return("", "error: failed to push some refs", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to push branch feature-branch",
		},
		{
			name:   "includes stderr in error message",
			dir:    "/test/repo",
			branch: "test-branch",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "push", "-u", "origin", "test-branch").
					Return("", "remote: Permission denied", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "stderr: remote: Permission denied",
		},
		{
			name:   "handles branch with special characters",
			dir:    "/test/repo",
			branch: "feature/user-auth",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "push", "-u", "origin", "feature/user-auth").
					Return("", "", nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.Push(ctx, tt.dir, tt.branch)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				mockRunner.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			mockRunner.AssertExpectations(t)
		})
	}
}

func TestGitRunner_WorktreeAdd(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		path        string
		branch      string
		setupMock   func(*MockCommandRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:   "creates worktree successfully",
			dir:    "/test/repo",
			path:   "/test/worktrees/feature",
			branch: "feature-branch",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "worktree", "add", "/test/worktrees/feature", "-b", "feature-branch").
					Return("", "", nil)
			},
		},
		{
			name:   "creates worktree with workflow branch",
			dir:    "/test/repo",
			path:   "/test/worktrees/workflow-1",
			branch: "workflow/workflow-1",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "worktree", "add", "/test/worktrees/workflow-1", "-b", "workflow/workflow-1").
					Return("", "", nil)
			},
		},
		{
			name:        "fails when path is empty",
			dir:         "/test/repo",
			path:        "",
			branch:      "feature-branch",
			setupMock:   func(m *MockCommandRunner) {},
			wantErr:     true,
			errContains: "worktree path cannot be empty",
		},
		{
			name:        "fails when branch is empty",
			dir:         "/test/repo",
			path:        "/test/worktrees/feature",
			branch:      "",
			setupMock:   func(m *MockCommandRunner) {},
			wantErr:     true,
			errContains: "branch name cannot be empty",
		},
		{
			name:   "returns error when branch already exists",
			dir:    "/test/repo",
			path:   "/test/worktrees/feature",
			branch: "existing-branch",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "worktree", "add", "/test/worktrees/feature", "-b", "existing-branch").
					Return("", "fatal: A branch named 'existing-branch' already exists", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "branch existing-branch already exists",
		},
		{
			name:   "returns error when worktree creation fails",
			dir:    "/test/repo",
			path:   "/invalid/path",
			branch: "feature-branch",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "worktree", "add", "/invalid/path", "-b", "feature-branch").
					Return("", "fatal: could not create directory", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to create worktree at /invalid/path",
		},
		{
			name:   "includes stderr in error message",
			dir:    "/test/repo",
			path:   "/test/worktrees/feature",
			branch: "test-branch",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "worktree", "add", "/test/worktrees/feature", "-b", "test-branch").
					Return("", "fatal: permission denied", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "stderr: fatal: permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.WorktreeAdd(ctx, tt.dir, tt.path, tt.branch)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				mockRunner.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			mockRunner.AssertExpectations(t)
		})
	}
}

func TestGitRunner_WorktreeRemove(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		path        string
		setupMock   func(*MockCommandRunner)
		wantErr     bool
		errContains string
	}{
		{
			name: "removes worktree successfully",
			dir:  "/test/repo",
			path: "/test/worktrees/feature",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "worktree", "remove", "/test/worktrees/feature").
					Return("", "", nil)
			},
		},
		{
			name: "removes worktree with workflow path",
			dir:  "/test/repo",
			path: "/test/worktrees/workflow-1",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "worktree", "remove", "/test/worktrees/workflow-1").
					Return("", "", nil)
			},
		},
		{
			name:        "fails when path is empty",
			dir:         "/test/repo",
			path:        "",
			setupMock:   func(m *MockCommandRunner) {},
			wantErr:     true,
			errContains: "worktree path cannot be empty",
		},
		{
			name: "returns error when worktree removal fails",
			dir:  "/test/repo",
			path: "/test/worktrees/feature",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "worktree", "remove", "/test/worktrees/feature").
					Return("", "fatal: worktree not found", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to remove worktree at /test/worktrees/feature",
		},
		{
			name: "includes stderr in error message",
			dir:  "/test/repo",
			path: "/test/worktrees/feature",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "worktree", "remove", "/test/worktrees/feature").
					Return("", "fatal: permission denied", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "stderr: fatal: permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.WorktreeRemove(ctx, tt.dir, tt.path)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				mockRunner.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			mockRunner.AssertExpectations(t)
		})
	}
}
