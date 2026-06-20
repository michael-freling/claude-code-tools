package command

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewGitRunner(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRunner := NewMockRunner(ctrl)
	got := NewGitRunner(mockRunner)

	require.NotNil(t, got)
}

func TestGitRunner_GetCurrentBranch(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		setupMock   func(*MockRunner)
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name: "returns current branch successfully",
			dir:  "/test/repo",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "rev-parse", "--abbrev-ref", "HEAD").
					Return("main", "", nil)
			},
			want:    "main",
			wantErr: false,
		},
		{
			name: "returns trimmed branch name",
			dir:  "/test/repo",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "rev-parse", "--abbrev-ref", "HEAD").
					Return("  feature-branch  ", "", nil)
			},
			want:    "feature-branch",
			wantErr: false,
		},
		{
			name: "fails when git command fails",
			dir:  "/test/repo",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "rev-parse", "--abbrev-ref", "HEAD").
					Return("", "fatal: not a git repository", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to get current branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRunner := NewMockRunner(ctrl)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			got, err := gitRunner.GetCurrentBranch(ctx, tt.dir)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitRunner_Push(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		branch      string
		setupMock   func(*MockRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:   "pushes branch successfully",
			dir:    "/test/repo",
			branch: "feature-branch",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "push", "-u", "origin", "feature-branch").
					Return("", "", nil)
			},
			wantErr: false,
		},
		{
			name:        "fails when branch name is empty",
			dir:         "/test/repo",
			branch:      "",
			setupMock:   func(m *MockRunner) {},
			wantErr:     true,
			errContains: "branch name cannot be empty",
		},
		{
			name:   "fails when git push fails",
			dir:    "/test/repo",
			branch: "feature-branch",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "push", "-u", "origin", "feature-branch").
					Return("", "fatal: repository not found", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to push branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRunner := NewMockRunner(ctrl)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.Push(ctx, tt.dir, tt.branch)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitRunner_WorktreeAdd(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		path        string
		branch      string
		setupMock   func(*MockRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:   "creates worktree successfully",
			dir:    "/test/repo",
			path:   "/test/worktree",
			branch: "feature-branch",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "worktree", "add", "/test/worktree", "-b", "feature-branch").
					Return("", "", nil)
			},
			wantErr: false,
		},
		{
			name:        "fails when path is empty",
			dir:         "/test/repo",
			path:        "",
			branch:      "feature-branch",
			setupMock:   func(m *MockRunner) {},
			wantErr:     true,
			errContains: "worktree path cannot be empty",
		},
		{
			name:        "fails when branch is empty",
			dir:         "/test/repo",
			path:        "/test/worktree",
			branch:      "",
			setupMock:   func(m *MockRunner) {},
			wantErr:     true,
			errContains: "branch name cannot be empty",
		},
		{
			name:   "fails when branch already exists",
			dir:    "/test/repo",
			path:   "/test/worktree",
			branch: "existing-branch",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "worktree", "add", "/test/worktree", "-b", "existing-branch").
					Return("", "fatal: A branch named 'existing-branch' already exists", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "branch existing-branch already exists",
		},
		{
			name:   "fails when git worktree add fails with other error",
			dir:    "/test/repo",
			path:   "/test/worktree",
			branch: "feature-branch",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "worktree", "add", "/test/worktree", "-b", "feature-branch").
					Return("", "fatal: some other error", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to create worktree",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRunner := NewMockRunner(ctrl)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.WorktreeAdd(ctx, tt.dir, tt.path, tt.branch)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitRunner_WorktreeRemove(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		path        string
		setupMock   func(*MockRunner)
		wantErr     bool
		errContains string
	}{
		{
			name: "removes worktree successfully",
			dir:  "/test/repo",
			path: "/test/worktree",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "worktree", "remove", "/test/worktree").
					Return("", "", nil)
			},
			wantErr: false,
		},
		{
			name:        "fails when path is empty",
			dir:         "/test/repo",
			path:        "",
			setupMock:   func(m *MockRunner) {},
			wantErr:     true,
			errContains: "worktree path cannot be empty",
		},
		{
			name: "fails when git worktree remove fails",
			dir:  "/test/repo",
			path: "/test/worktree",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "worktree", "remove", "/test/worktree").
					Return("", "fatal: '/test/worktree' is not a working tree", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to remove worktree",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRunner := NewMockRunner(ctrl)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.WorktreeRemove(ctx, tt.dir, tt.path)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitRunner_GetCommits(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		base        string
		setupMock   func(*MockRunner)
		want        []Commit
		wantErr     bool
		errContains string
	}{
		{
			name: "returns commits successfully",
			dir:  "/test/repo",
			base: "main",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "log", "main..HEAD", "--format=%H|%s", "--reverse").
					Return("abc123|First commit\ndef456|Second commit\n", "", nil)
			},
			want: []Commit{
				{Hash: "abc123", Subject: "First commit"},
				{Hash: "def456", Subject: "Second commit"},
			},
			wantErr: false,
		},
		{
			name: "returns empty list when no commits",
			dir:  "/test/repo",
			base: "main",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "log", "main..HEAD", "--format=%H|%s", "--reverse").
					Return("", "", nil)
			},
			want:    []Commit{},
			wantErr: false,
		},
		{
			name:        "fails when base is empty",
			dir:         "/test/repo",
			base:        "",
			setupMock:   func(m *MockRunner) {},
			wantErr:     true,
			errContains: "base branch cannot be empty",
		},
		{
			name: "fails when git log fails",
			dir:  "/test/repo",
			base: "main",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "log", "main..HEAD", "--format=%H|%s", "--reverse").
					Return("", "fatal: bad revision", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to get commits from main",
		},
		{
			name: "fails when commit format is invalid",
			dir:  "/test/repo",
			base: "main",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "log", "main..HEAD", "--format=%H|%s", "--reverse").
					Return("invalid-format-without-pipe\n", "", nil)
			},
			wantErr:     true,
			errContains: "invalid commit format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRunner := NewMockRunner(ctrl)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			got, err := gitRunner.GetCommits(ctx, tt.dir, tt.base)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitRunner_CherryPick(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		commitHash  string
		setupMock   func(*MockRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:       "cherry-picks commit successfully",
			dir:        "/test/repo",
			commitHash: "abc123",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "cherry-pick", "abc123").
					Return("", "", nil)
			},
			wantErr: false,
		},
		{
			name:        "fails when commit hash is empty",
			dir:         "/test/repo",
			commitHash:  "",
			setupMock:   func(m *MockRunner) {},
			wantErr:     true,
			errContains: "commit hash cannot be empty",
		},
		{
			name:       "fails when git cherry-pick fails",
			dir:        "/test/repo",
			commitHash: "abc123",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "cherry-pick", "abc123").
					Return("", "error: could not apply abc123", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to cherry-pick commit abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRunner := NewMockRunner(ctrl)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.CherryPick(ctx, tt.dir, tt.commitHash)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitRunner_CreateBranch(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		branchName  string
		baseBranch  string
		setupMock   func(*MockRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:       "creates branch successfully",
			dir:        "/test/repo",
			branchName: "feature-branch",
			baseBranch: "main",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "checkout", "-b", "feature-branch", "main").
					Return("", "", nil)
			},
			wantErr: false,
		},
		{
			name:        "fails when branch name is empty",
			dir:         "/test/repo",
			branchName:  "",
			baseBranch:  "main",
			setupMock:   func(m *MockRunner) {},
			wantErr:     true,
			errContains: "branch name cannot be empty",
		},
		{
			name:        "fails when base branch is empty",
			dir:         "/test/repo",
			branchName:  "feature-branch",
			baseBranch:  "",
			setupMock:   func(m *MockRunner) {},
			wantErr:     true,
			errContains: "base branch cannot be empty",
		},
		{
			name:       "fails when git checkout fails",
			dir:        "/test/repo",
			branchName: "feature-branch",
			baseBranch: "main",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "checkout", "-b", "feature-branch", "main").
					Return("", "fatal: A branch named 'feature-branch' already exists", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to create branch feature-branch from main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRunner := NewMockRunner(ctrl)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.CreateBranch(ctx, tt.dir, tt.branchName, tt.baseBranch)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitRunner_CheckoutBranch(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		branchName  string
		setupMock   func(*MockRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:       "checks out branch successfully",
			dir:        "/test/repo",
			branchName: "feature-branch",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "checkout", "feature-branch").
					Return("", "", nil)
			},
			wantErr: false,
		},
		{
			name:        "fails when branch name is empty",
			dir:         "/test/repo",
			branchName:  "",
			setupMock:   func(m *MockRunner) {},
			wantErr:     true,
			errContains: "branch name cannot be empty",
		},
		{
			name:       "fails when git checkout fails",
			dir:        "/test/repo",
			branchName: "feature-branch",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "checkout", "feature-branch").
					Return("", "error: pathspec 'feature-branch' did not match any file(s) known to git", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to checkout branch feature-branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRunner := NewMockRunner(ctrl)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.CheckoutBranch(ctx, tt.dir, tt.branchName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitRunner_DeleteBranch(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		branchName  string
		force       bool
		setupMock   func(*MockRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:       "deletes branch successfully with normal delete",
			dir:        "/test/repo",
			branchName: "feature-branch",
			force:      false,
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "branch", "-d", "feature-branch").
					Return("", "", nil)
			},
			wantErr: false,
		},
		{
			name:       "deletes branch successfully with force delete",
			dir:        "/test/repo",
			branchName: "feature-branch",
			force:      true,
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "branch", "-D", "feature-branch").
					Return("", "", nil)
			},
			wantErr: false,
		},
		{
			name:        "fails when branch name is empty",
			dir:         "/test/repo",
			branchName:  "",
			force:       false,
			setupMock:   func(m *MockRunner) {},
			wantErr:     true,
			errContains: "branch name cannot be empty",
		},
		{
			name:       "fails when git branch delete fails",
			dir:        "/test/repo",
			branchName: "feature-branch",
			force:      false,
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "branch", "-d", "feature-branch").
					Return("", "error: The branch 'feature-branch' is not fully merged", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to delete branch feature-branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRunner := NewMockRunner(ctrl)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.DeleteBranch(ctx, tt.dir, tt.branchName, tt.force)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitRunner_DeleteRemoteBranch(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		branchName  string
		setupMock   func(*MockRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:       "deletes remote branch successfully",
			dir:        "/test/repo",
			branchName: "feature-branch",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "push", "origin", "--delete", "feature-branch").
					Return("", "", nil)
			},
			wantErr: false,
		},
		{
			name:        "fails when branch name is empty",
			dir:         "/test/repo",
			branchName:  "",
			setupMock:   func(m *MockRunner) {},
			wantErr:     true,
			errContains: "branch name cannot be empty",
		},
		{
			name:       "fails when git push delete fails",
			dir:        "/test/repo",
			branchName: "feature-branch",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "push", "origin", "--delete", "feature-branch").
					Return("", "error: unable to delete 'feature-branch': remote ref does not exist", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to delete remote branch feature-branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRunner := NewMockRunner(ctrl)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.DeleteRemoteBranch(ctx, tt.dir, tt.branchName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitRunner_CommitEmpty(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		message     string
		setupMock   func(*MockRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:    "creates empty commit successfully",
			dir:     "/test/repo",
			message: "Empty commit message",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "commit", "--allow-empty", "-m", "Empty commit message").
					Return("", "", nil)
			},
			wantErr: false,
		},
		{
			name:        "fails when message is empty",
			dir:         "/test/repo",
			message:     "",
			setupMock:   func(m *MockRunner) {},
			wantErr:     true,
			errContains: "commit message cannot be empty",
		},
		{
			name:    "fails when git commit fails",
			dir:     "/test/repo",
			message: "Empty commit message",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "commit", "--allow-empty", "-m", "Empty commit message").
					Return("", "fatal: unable to write new index file", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to create empty commit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRunner := NewMockRunner(ctrl)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.CommitEmpty(ctx, tt.dir, tt.message)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitRunner_CheckoutFiles(t *testing.T) {
	tests := []struct {
		name         string
		dir          string
		sourceBranch string
		files        []string
		setupMock    func(*MockRunner)
		wantErr      bool
		errContains  string
	}{
		{
			name:         "checks out files successfully",
			dir:          "/test/repo",
			sourceBranch: "main",
			files:        []string{"file1.go", "file2.go"},
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "checkout", "main", "--", "file1.go", "file2.go").
					Return("", "", nil)
			},
			wantErr: false,
		},
		{
			name:         "checks out single file successfully",
			dir:          "/test/repo",
			sourceBranch: "develop",
			files:        []string{"README.md"},
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "checkout", "develop", "--", "README.md").
					Return("", "", nil)
			},
			wantErr: false,
		},
		{
			name:         "fails when source branch is empty",
			dir:          "/test/repo",
			sourceBranch: "",
			files:        []string{"file1.go"},
			setupMock:    func(m *MockRunner) {},
			wantErr:      true,
			errContains:  "source branch cannot be empty",
		},
		{
			name:         "fails when files list is empty",
			dir:          "/test/repo",
			sourceBranch: "main",
			files:        []string{},
			setupMock:    func(m *MockRunner) {},
			wantErr:      true,
			errContains:  "files list cannot be empty",
		},
		{
			name:         "fails when git checkout fails",
			dir:          "/test/repo",
			sourceBranch: "main",
			files:        []string{"file1.go"},
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "checkout", "main", "--", "file1.go").
					Return("", "error: pathspec 'file1.go' did not match any file(s) known to git", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to checkout files from main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRunner := NewMockRunner(ctrl)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.CheckoutFiles(ctx, tt.dir, tt.sourceBranch, tt.files)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitRunner_CommitAll(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		message     string
		setupMock   func(*MockRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:    "commits all changes successfully",
			dir:     "/test/repo",
			message: "Add new feature",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "add", "-A").
					Return("", "", nil)
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "commit", "-m", "Add new feature").
					Return("", "", nil)
			},
			wantErr: false,
		},
		{
			name:        "fails when message is empty",
			dir:         "/test/repo",
			message:     "",
			setupMock:   func(m *MockRunner) {},
			wantErr:     true,
			errContains: "commit message cannot be empty",
		},
		{
			name:    "fails when git add fails",
			dir:     "/test/repo",
			message: "Add new feature",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "add", "-A").
					Return("", "fatal: not a git repository", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to stage changes",
		},
		{
			name:    "fails when git commit fails",
			dir:     "/test/repo",
			message: "Add new feature",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "add", "-A").
					Return("", "", nil)
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "commit", "-m", "Add new feature").
					Return("", "error: nothing to commit", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to create commit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRunner := NewMockRunner(ctrl)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			err := gitRunner.CommitAll(ctx, tt.dir, tt.message)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitRunner_GetDiffStat(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		base        string
		setupMock   func(*MockRunner)
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name: "returns diff stat successfully",
			dir:  "/test/repo",
			base: "main",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "diff", "--stat", "main").
					Return(" file1.go | 10 +++++-----\n file2.go | 5 +++++\n 2 files changed, 10 insertions(+), 5 deletions(-)\n", "", nil)
			},
			want:    " file1.go | 10 +++++-----\n file2.go | 5 +++++\n 2 files changed, 10 insertions(+), 5 deletions(-)\n",
			wantErr: false,
		},
		{
			name: "returns empty string when no differences",
			dir:  "/test/repo",
			base: "main",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "diff", "--stat", "main").
					Return("", "", nil)
			},
			want:    "",
			wantErr: false,
		},
		{
			name:        "fails when base is empty",
			dir:         "/test/repo",
			base:        "",
			setupMock:   func(m *MockRunner) {},
			wantErr:     true,
			errContains: "base branch cannot be empty",
		},
		{
			name: "fails when git diff fails",
			dir:  "/test/repo",
			base: "main",
			setupMock: func(m *MockRunner) {
				m.EXPECT().
					RunInDir(gomock.Any(), "/test/repo", "git", "diff", "--stat", "main").
					Return("", "fatal: bad revision 'main'", fmt.Errorf("exit status 128"))
			},
			wantErr:     true,
			errContains: "failed to get diff stat from main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRunner := NewMockRunner(ctrl)
			tt.setupMock(mockRunner)

			gitRunner := NewGitRunner(mockRunner)
			ctx := context.Background()

			got, err := gitRunner.GetDiffStat(ctx, tt.dir, tt.base)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
