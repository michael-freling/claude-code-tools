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

func TestGitRunner_GetCommits(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		base        string
		setupMock   func(*MockCommandRunner)
		want        []Commit
		wantErr     bool
		errContains string
	}{
		{
			name: "returns multiple commits",
			dir:  "/test/repo",
			base: "main",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "log", "main..HEAD", "--format=%H|%s", "--reverse").
					Return("abc123|First commit\ndef456|Second commit\n", "", nil)
			},
			want: []Commit{
				{Hash: "abc123", Subject: "First commit"},
				{Hash: "def456", Subject: "Second commit"},
			},
		},
		{
			name: "returns no commits when branches are equal",
			dir:  "/test/repo",
			base: "main",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "log", "main..HEAD", "--format=%H|%s", "--reverse").
					Return("", "", nil)
			},
			want: []Commit{},
		},
		{
			name: "handles commit subject with pipe character",
			dir:  "/test/repo",
			base: "main",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "log", "main..HEAD", "--format=%H|%s", "--reverse").
					Return("abc123|Fix: handle | character in message\n", "", nil)
			},
			want: []Commit{
				{Hash: "abc123", Subject: "Fix: handle | character in message"},
			},
		},
		{
			name:        "fails when base branch is empty",
			dir:         "/test/repo",
			base:        "",
			setupMock:   func(m *MockCommandRunner) {},
			wantErr:     true,
			errContains: "base branch cannot be empty",
		},
		{
			name: "returns error when git command fails",
			dir:  "/test/repo",
			base: "main",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "log", "main..HEAD", "--format=%H|%s", "--reverse").
					Return("", "fatal: bad revision", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to get commits from main",
		},
		{
			name: "returns error for invalid commit format",
			dir:  "/test/repo",
			base: "main",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "log", "main..HEAD", "--format=%H|%s", "--reverse").
					Return("abc123\n", "", nil)
			},
			wantErr:     true,
			errContains: "invalid commit format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			got, err := gitRunner.GetCommits(ctx, tt.dir, tt.base)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, got)
				mockRunner.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			mockRunner.AssertExpectations(t)
		})
	}
}

func TestGitRunner_CherryPick(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		commitHash  string
		setupMock   func(*MockCommandRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:       "cherry-picks commit successfully",
			dir:        "/test/repo",
			commitHash: "abc123",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "cherry-pick", "abc123").
					Return("", "", nil)
			},
		},
		{
			name:       "cherry-picks commit with long hash",
			dir:        "/test/repo",
			commitHash: "abc123def456ghi789",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "cherry-pick", "abc123def456ghi789").
					Return("", "", nil)
			},
		},
		{
			name:        "fails when commit hash is empty",
			dir:         "/test/repo",
			commitHash:  "",
			setupMock:   func(m *MockCommandRunner) {},
			wantErr:     true,
			errContains: "commit hash cannot be empty",
		},
		{
			name:       "returns error when cherry-pick fails",
			dir:        "/test/repo",
			commitHash: "abc123",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "cherry-pick", "abc123").
					Return("", "error: could not apply abc123", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to cherry-pick commit abc123",
		},
		{
			name:       "includes stderr in error message",
			dir:        "/test/repo",
			commitHash: "def456",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "cherry-pick", "def456").
					Return("", "error: merge conflict", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "stderr: error: merge conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.CherryPick(ctx, tt.dir, tt.commitHash)

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

func TestGitRunner_CreateBranch(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		branchName  string
		baseBranch  string
		setupMock   func(*MockCommandRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:       "creates branch successfully",
			dir:        "/test/repo",
			branchName: "feature/test",
			baseBranch: "main",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "checkout", "-b", "feature/test", "main").
					Return("", "", nil)
			},
		},
		{
			name:       "creates branch from different base",
			dir:        "/test/repo",
			branchName: "hotfix/bug",
			baseBranch: "develop",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "checkout", "-b", "hotfix/bug", "develop").
					Return("", "", nil)
			},
		},
		{
			name:        "fails when branch name is empty",
			dir:         "/test/repo",
			branchName:  "",
			baseBranch:  "main",
			setupMock:   func(m *MockCommandRunner) {},
			wantErr:     true,
			errContains: "branch name cannot be empty",
		},
		{
			name:        "fails when base branch is empty",
			dir:         "/test/repo",
			branchName:  "feature/test",
			baseBranch:  "",
			setupMock:   func(m *MockCommandRunner) {},
			wantErr:     true,
			errContains: "base branch cannot be empty",
		},
		{
			name:       "returns error when branch already exists",
			dir:        "/test/repo",
			branchName: "existing",
			baseBranch: "main",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "checkout", "-b", "existing", "main").
					Return("", "fatal: a branch named 'existing' already exists", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to create branch existing from main",
		},
		{
			name:       "includes stderr in error message",
			dir:        "/test/repo",
			branchName: "feature/test",
			baseBranch: "nonexistent",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "checkout", "-b", "feature/test", "nonexistent").
					Return("", "fatal: invalid reference: nonexistent", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "stderr: fatal: invalid reference: nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.CreateBranch(ctx, tt.dir, tt.branchName, tt.baseBranch)

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

func TestGitRunner_CheckoutBranch(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		branchName  string
		setupMock   func(*MockCommandRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:       "checks out branch successfully",
			dir:        "/test/repo",
			branchName: "main",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "checkout", "main").
					Return("", "", nil)
			},
		},
		{
			name:       "checks out feature branch",
			dir:        "/test/repo",
			branchName: "feature/test",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "checkout", "feature/test").
					Return("", "", nil)
			},
		},
		{
			name:        "fails when branch name is empty",
			dir:         "/test/repo",
			branchName:  "",
			setupMock:   func(m *MockCommandRunner) {},
			wantErr:     true,
			errContains: "branch name cannot be empty",
		},
		{
			name:       "returns error when branch does not exist",
			dir:        "/test/repo",
			branchName: "nonexistent",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "checkout", "nonexistent").
					Return("", "error: pathspec 'nonexistent' did not match any file(s)", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to checkout branch nonexistent",
		},
		{
			name:       "includes stderr in error message",
			dir:        "/test/repo",
			branchName: "test",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "checkout", "test").
					Return("", "error: uncommitted changes", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "stderr: error: uncommitted changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.CheckoutBranch(ctx, tt.dir, tt.branchName)

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

func TestGitRunner_DeleteBranch(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		branchName  string
		force       bool
		setupMock   func(*MockCommandRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:       "deletes branch successfully",
			dir:        "/test/repo",
			branchName: "feature/test",
			force:      false,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "branch", "-d", "feature/test").
					Return("", "", nil)
			},
		},
		{
			name:       "force deletes branch successfully",
			dir:        "/test/repo",
			branchName: "feature/test",
			force:      true,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "branch", "-D", "feature/test").
					Return("", "", nil)
			},
		},
		{
			name:        "fails when branch name is empty",
			dir:         "/test/repo",
			branchName:  "",
			force:       false,
			setupMock:   func(m *MockCommandRunner) {},
			wantErr:     true,
			errContains: "branch name cannot be empty",
		},
		{
			name:       "returns error when branch does not exist",
			dir:        "/test/repo",
			branchName: "nonexistent",
			force:      false,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "branch", "-d", "nonexistent").
					Return("", "error: branch 'nonexistent' not found", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to delete branch nonexistent",
		},
		{
			name:       "includes stderr in error message",
			dir:        "/test/repo",
			branchName: "unmerged",
			force:      false,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "branch", "-d", "unmerged").
					Return("", "error: branch 'unmerged' is not fully merged", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "stderr: error: branch 'unmerged' is not fully merged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.DeleteBranch(ctx, tt.dir, tt.branchName, tt.force)

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

func TestGitRunner_DeleteRemoteBranch(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		branchName  string
		setupMock   func(*MockCommandRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:       "deletes remote branch successfully",
			dir:        "/test/repo",
			branchName: "feature/test",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "push", "origin", "--delete", "feature/test").
					Return("", "", nil)
			},
		},
		{
			name:        "fails when branch name is empty",
			dir:         "/test/repo",
			branchName:  "",
			setupMock:   func(m *MockCommandRunner) {},
			wantErr:     true,
			errContains: "branch name cannot be empty",
		},
		{
			name:       "returns error when remote branch does not exist",
			dir:        "/test/repo",
			branchName: "nonexistent",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "push", "origin", "--delete", "nonexistent").
					Return("", "error: unable to delete 'nonexistent': remote ref does not exist", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to delete remote branch nonexistent",
		},
		{
			name:       "includes stderr in error message",
			dir:        "/test/repo",
			branchName: "protected",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "push", "origin", "--delete", "protected").
					Return("", "error: refusing to delete protected branch", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "stderr: error: refusing to delete protected branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.DeleteRemoteBranch(ctx, tt.dir, tt.branchName)

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

func TestGitRunner_CommitEmpty(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		message     string
		setupMock   func(*MockCommandRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:    "creates empty commit successfully",
			dir:     "/test/repo",
			message: "Empty commit for parent PR",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "commit", "--allow-empty", "-m", "Empty commit for parent PR").
					Return("", "", nil)
			},
		},
		{
			name:    "creates empty commit with multiline message",
			dir:     "/test/repo",
			message: "Parent PR\n\nThis is a parent PR for child PRs",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "commit", "--allow-empty", "-m", "Parent PR\n\nThis is a parent PR for child PRs").
					Return("", "", nil)
			},
		},
		{
			name:        "fails when message is empty",
			dir:         "/test/repo",
			message:     "",
			setupMock:   func(m *MockCommandRunner) {},
			wantErr:     true,
			errContains: "commit message cannot be empty",
		},
		{
			name:    "returns error when commit fails",
			dir:     "/test/repo",
			message: "Empty commit",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "commit", "--allow-empty", "-m", "Empty commit").
					Return("", "fatal: unable to commit", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to create empty commit",
		},
		{
			name:    "includes stderr in error message",
			dir:     "/test/repo",
			message: "Empty commit",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "commit", "--allow-empty", "-m", "Empty commit").
					Return("", "error: gpg failed to sign", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "stderr: error: gpg failed to sign",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.CommitEmpty(ctx, tt.dir, tt.message)

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

func TestGitRunner_CheckoutFiles(t *testing.T) {
	tests := []struct {
		name         string
		dir          string
		sourceBranch string
		files        []string
		setupMock    func(*MockCommandRunner)
		wantErr      bool
		errContains  string
	}{
		{
			name:         "checks out single file successfully",
			dir:          "/test/repo",
			sourceBranch: "feature-branch",
			files:        []string{"file1.go"},
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "checkout", "feature-branch", "--", "file1.go").
					Return("", "", nil)
			},
		},
		{
			name:         "checks out multiple files successfully",
			dir:          "/test/repo",
			sourceBranch: "feature-branch",
			files:        []string{"file1.go", "file2.go", "file3.go"},
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "checkout", "feature-branch", "--", "file1.go", "file2.go", "file3.go").
					Return("", "", nil)
			},
		},
		{
			name:         "checks out files with paths",
			dir:          "/test/repo",
			sourceBranch: "main",
			files:        []string{"path/to/file1.go", "another/path/file2.go"},
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "checkout", "main", "--", "path/to/file1.go", "another/path/file2.go").
					Return("", "", nil)
			},
		},
		{
			name:         "fails when source branch is empty",
			dir:          "/test/repo",
			sourceBranch: "",
			files:        []string{"file1.go"},
			setupMock:    func(m *MockCommandRunner) {},
			wantErr:      true,
			errContains:  "source branch cannot be empty",
		},
		{
			name:         "fails when files list is empty",
			dir:          "/test/repo",
			sourceBranch: "feature-branch",
			files:        []string{},
			setupMock:    func(m *MockCommandRunner) {},
			wantErr:      true,
			errContains:  "files list cannot be empty",
		},
		{
			name:         "returns error when checkout fails",
			dir:          "/test/repo",
			sourceBranch: "feature-branch",
			files:        []string{"file1.go"},
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "checkout", "feature-branch", "--", "file1.go").
					Return("", "error: pathspec 'file1.go' did not match any file(s)", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to checkout files from feature-branch",
		},
		{
			name:         "includes stderr in error message",
			dir:          "/test/repo",
			sourceBranch: "feature-branch",
			files:        []string{"file1.go"},
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "checkout", "feature-branch", "--", "file1.go").
					Return("", "fatal: invalid reference: feature-branch", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "stderr: fatal: invalid reference: feature-branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.CheckoutFiles(ctx, tt.dir, tt.sourceBranch, tt.files)

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

func TestGitRunner_CommitAll(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		message     string
		setupMock   func(*MockCommandRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:    "commits all changes successfully",
			dir:     "/test/repo",
			message: "Add new feature",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "add", "-A").
					Return("", "", nil)
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "commit", "-m", "Add new feature").
					Return("", "", nil)
			},
		},
		{
			name:    "commits with multiline message",
			dir:     "/test/repo",
			message: "Add feature\n\nDetailed description",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "add", "-A").
					Return("", "", nil)
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "commit", "-m", "Add feature\n\nDetailed description").
					Return("", "", nil)
			},
		},
		{
			name:        "fails when message is empty",
			dir:         "/test/repo",
			message:     "",
			setupMock:   func(m *MockCommandRunner) {},
			wantErr:     true,
			errContains: "commit message cannot be empty",
		},
		{
			name:    "returns error when add fails",
			dir:     "/test/repo",
			message: "Add feature",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "add", "-A").
					Return("", "fatal: not a git repository", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to stage changes",
		},
		{
			name:    "includes stderr in add error message",
			dir:     "/test/repo",
			message: "Add feature",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "add", "-A").
					Return("", "fatal: permission denied", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "stderr: fatal: permission denied",
		},
		{
			name:    "returns error when commit fails",
			dir:     "/test/repo",
			message: "Add feature",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "add", "-A").
					Return("", "", nil)
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "commit", "-m", "Add feature").
					Return("", "fatal: unable to commit", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to create commit",
		},
		{
			name:    "includes stderr in commit error message",
			dir:     "/test/repo",
			message: "Add feature",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "add", "-A").
					Return("", "", nil)
				m.On("RunInDir", mock.Anything, "/test/repo", "git", "commit", "-m", "Add feature").
					Return("", "error: gpg failed to sign", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "stderr: error: gpg failed to sign",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.CommitAll(ctx, tt.dir, tt.message)

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
