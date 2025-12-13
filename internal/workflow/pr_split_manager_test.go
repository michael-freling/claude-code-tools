package workflow

import (
	"context"
	"fmt"
	"testing"

	"github.com/michael-freling/claude-code-tools/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewPRSplitManager(t *testing.T) {
	tests := []struct {
		name string
		git  command.GitRunner
		gh   command.GhRunner
	}{
		{
			name: "creates PR split manager with git and gh runners",
			git:  &MockGitRunner{},
			gh:   &MockGhRunner{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewPRSplitManager(tt.git, tt.gh)
			require.NotNil(t, got)

			manager, ok := got.(*prSplitManager)
			require.True(t, ok)
			assert.Equal(t, tt.git, manager.git)
			assert.Equal(t, tt.gh, manager.gh)
		})
	}
}

func TestPRSplitManager_ExecuteSplit_Commits(t *testing.T) {
	tests := []struct {
		name         string
		dir          string
		plan         *PRSplitPlan
		sourceBranch string
		mainBranch   string
		setupMocks   func(*MockGitRunner, *MockGhRunner)
		want         *PRSplitResult
		wantErr      bool
		errContains  string
	}{
		{
			name:         "successfully splits PR with commits strategy",
			dir:          "/test/repo",
			sourceBranch: "feature-branch",
			mainBranch:   "main",
			plan: &PRSplitPlan{
				Strategy:    SplitByCommits,
				ParentTitle: "Parent: Feature Implementation",
				ParentDesc:  "Parent PR for split feature",
				ChildPRs: []ChildPRPlan{
					{
						Title:       "Part 1: Setup",
						Description: "Initial setup",
						Commits:     []string{"abc123", "def456"},
					},
					{
						Title:       "Part 2: Implementation",
						Description: "Main implementation",
						Commits:     []string{"ghi789"},
					},
				},
				Summary: "Split into 2 child PRs",
			},
			setupMocks: func(mockGit *MockGitRunner, mockGh *MockGhRunner) {
				mockGit.On("CreateBranch", mock.Anything, "/test/repo", "split/feature-branch/parent", "main").Return(nil)
				mockGit.On("CheckoutBranch", mock.Anything, "/test/repo", "split/feature-branch/parent").Return(nil)
				mockGit.On("CommitEmpty", mock.Anything, "/test/repo", "Parent PR for split: Parent: Feature Implementation").Return(nil)
				mockGit.On("Push", mock.Anything, "/test/repo", "split/feature-branch/parent").Return(nil)

				mockGit.On("CreateBranch", mock.Anything, "/test/repo", "split/feature-branch/child-1", "split/feature-branch/parent").Return(nil)
				mockGit.On("CheckoutBranch", mock.Anything, "/test/repo", "split/feature-branch/child-1").Return(nil)
				mockGit.On("CherryPick", mock.Anything, "/test/repo", "abc123").Return(nil)
				mockGit.On("CherryPick", mock.Anything, "/test/repo", "def456").Return(nil)
				mockGit.On("Push", mock.Anything, "/test/repo", "split/feature-branch/child-1").Return(nil)

				mockGit.On("CreateBranch", mock.Anything, "/test/repo", "split/feature-branch/child-2", "split/feature-branch/child-1").Return(nil)
				mockGit.On("CheckoutBranch", mock.Anything, "/test/repo", "split/feature-branch/child-2").Return(nil)
				mockGit.On("CherryPick", mock.Anything, "/test/repo", "ghi789").Return(nil)
				mockGit.On("Push", mock.Anything, "/test/repo", "split/feature-branch/child-2").Return(nil)

				mockGh.On("PRCreate", mock.Anything, "/test/repo", "Parent: Feature Implementation", "Parent PR for split feature", "split/feature-branch/parent", "main").
					Return("https://github.com/owner/repo/pull/100", nil)
				mockGh.On("PRCreate", mock.Anything, "/test/repo", "Part 1: Setup", "Initial setup", "split/feature-branch/child-1", "split/feature-branch/parent").
					Return("https://github.com/owner/repo/pull/101", nil)
				mockGh.On("PRCreate", mock.Anything, "/test/repo", "Part 2: Implementation", "Main implementation", "split/feature-branch/child-2", "split/feature-branch/child-1").
					Return("https://github.com/owner/repo/pull/102", nil)
				mockGh.On("PREdit", mock.Anything, "/test/repo", 100, "Parent PR for split feature\n\n## Child PRs\n\n- #101 - Part 1: Setup\n- #102 - Part 2: Implementation").
					Return(nil)
			},
			want: &PRSplitResult{
				ParentPR: PRInfo{
					Number:      100,
					URL:         "https://github.com/owner/repo/pull/100",
					Title:       "Parent: Feature Implementation",
					Description: "Parent PR for split feature",
				},
				ChildPRs: []PRInfo{
					{
						Number:      101,
						URL:         "https://github.com/owner/repo/pull/101",
						Title:       "Part 1: Setup",
						Description: "Initial setup",
					},
					{
						Number:      102,
						URL:         "https://github.com/owner/repo/pull/102",
						Title:       "Part 2: Implementation",
						Description: "Main implementation",
					},
				},
				Summary: "Split into 2 child PRs",
				BranchNames: []string{
					"split/feature-branch/parent",
					"split/feature-branch/child-1",
					"split/feature-branch/child-2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := new(MockGitRunner)
			mockGh := new(MockGhRunner)
			tt.setupMocks(mockGit, mockGh)

			manager := NewPRSplitManager(mockGit, mockGh)
			ctx := context.Background()

			got, err := manager.ExecuteSplit(ctx, tt.dir, tt.plan, tt.sourceBranch, tt.mainBranch)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				mockGit.AssertExpectations(t)
				mockGh.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			mockGit.AssertExpectations(t)
			mockGh.AssertExpectations(t)
		})
	}
}

func TestPRSplitManager_ExecuteSplit_Files(t *testing.T) {
	tests := []struct {
		name         string
		dir          string
		plan         *PRSplitPlan
		sourceBranch string
		mainBranch   string
		setupMocks   func(*MockGitRunner, *MockGhRunner)
		want         *PRSplitResult
		wantErr      bool
		errContains  string
	}{
		{
			name:         "successfully splits PR with files strategy",
			dir:          "/test/repo",
			sourceBranch: "feature-branch",
			mainBranch:   "main",
			plan: &PRSplitPlan{
				Strategy:    SplitByFiles,
				ParentTitle: "Parent: Feature Implementation",
				ParentDesc:  "Parent PR for split feature",
				ChildPRs: []ChildPRPlan{
					{
						Title:       "Part 1: Backend",
						Description: "Backend changes",
						Files:       []string{"backend/file1.go", "backend/file2.go"},
					},
					{
						Title:       "Part 2: Frontend",
						Description: "Frontend changes",
						Files:       []string{"frontend/file1.ts", "frontend/file2.ts"},
					},
				},
				Summary: "Split into backend and frontend",
			},
			setupMocks: func(mockGit *MockGitRunner, mockGh *MockGhRunner) {
				mockGit.On("CreateBranch", mock.Anything, "/test/repo", "split/feature-branch/parent", "main").Return(nil)
				mockGit.On("CheckoutBranch", mock.Anything, "/test/repo", "split/feature-branch/parent").Return(nil)
				mockGit.On("CommitEmpty", mock.Anything, "/test/repo", "Parent PR for split: Parent: Feature Implementation").Return(nil)
				mockGit.On("Push", mock.Anything, "/test/repo", "split/feature-branch/parent").Return(nil)

				mockGit.On("CreateBranch", mock.Anything, "/test/repo", "split/feature-branch/child-1", "split/feature-branch/parent").Return(nil)
				mockGit.On("CheckoutBranch", mock.Anything, "/test/repo", "split/feature-branch/child-1").Return(nil)
				mockGit.On("CheckoutFiles", mock.Anything, "/test/repo", "feature-branch", []string{"backend/file1.go", "backend/file2.go"}).Return(nil)
				mockGit.On("CommitAll", mock.Anything, "/test/repo", "Part 1: Backend").Return(nil)
				mockGit.On("Push", mock.Anything, "/test/repo", "split/feature-branch/child-1").Return(nil)

				mockGit.On("CreateBranch", mock.Anything, "/test/repo", "split/feature-branch/child-2", "split/feature-branch/child-1").Return(nil)
				mockGit.On("CheckoutBranch", mock.Anything, "/test/repo", "split/feature-branch/child-2").Return(nil)
				mockGit.On("CheckoutFiles", mock.Anything, "/test/repo", "feature-branch", []string{"frontend/file1.ts", "frontend/file2.ts"}).Return(nil)
				mockGit.On("CommitAll", mock.Anything, "/test/repo", "Part 2: Frontend").Return(nil)
				mockGit.On("Push", mock.Anything, "/test/repo", "split/feature-branch/child-2").Return(nil)

				mockGh.On("PRCreate", mock.Anything, "/test/repo", "Parent: Feature Implementation", "Parent PR for split feature", "split/feature-branch/parent", "main").
					Return("https://github.com/owner/repo/pull/200", nil)
				mockGh.On("PRCreate", mock.Anything, "/test/repo", "Part 1: Backend", "Backend changes", "split/feature-branch/child-1", "split/feature-branch/parent").
					Return("https://github.com/owner/repo/pull/201", nil)
				mockGh.On("PRCreate", mock.Anything, "/test/repo", "Part 2: Frontend", "Frontend changes", "split/feature-branch/child-2", "split/feature-branch/child-1").
					Return("https://github.com/owner/repo/pull/202", nil)
				mockGh.On("PREdit", mock.Anything, "/test/repo", 200, "Parent PR for split feature\n\n## Child PRs\n\n- #201 - Part 1: Backend\n- #202 - Part 2: Frontend").
					Return(nil)
			},
			want: &PRSplitResult{
				ParentPR: PRInfo{
					Number:      200,
					URL:         "https://github.com/owner/repo/pull/200",
					Title:       "Parent: Feature Implementation",
					Description: "Parent PR for split feature",
				},
				ChildPRs: []PRInfo{
					{
						Number:      201,
						URL:         "https://github.com/owner/repo/pull/201",
						Title:       "Part 1: Backend",
						Description: "Backend changes",
					},
					{
						Number:      202,
						URL:         "https://github.com/owner/repo/pull/202",
						Title:       "Part 2: Frontend",
						Description: "Frontend changes",
					},
				},
				Summary: "Split into backend and frontend",
				BranchNames: []string{
					"split/feature-branch/parent",
					"split/feature-branch/child-1",
					"split/feature-branch/child-2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := new(MockGitRunner)
			mockGh := new(MockGhRunner)
			tt.setupMocks(mockGit, mockGh)

			manager := NewPRSplitManager(mockGit, mockGh)
			ctx := context.Background()

			got, err := manager.ExecuteSplit(ctx, tt.dir, tt.plan, tt.sourceBranch, tt.mainBranch)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				mockGit.AssertExpectations(t)
				mockGh.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			mockGit.AssertExpectations(t)
			mockGh.AssertExpectations(t)
		})
	}
}

func TestPRSplitManager_ExecuteSplit_Errors(t *testing.T) {
	tests := []struct {
		name         string
		dir          string
		plan         *PRSplitPlan
		sourceBranch string
		mainBranch   string
		setupMocks   func(*MockGitRunner, *MockGhRunner)
		wantErr      bool
		errContains  string
	}{
		{
			name:         "fails when plan is nil",
			dir:          "/test/repo",
			plan:         nil,
			sourceBranch: "feature-branch",
			mainBranch:   "main",
			setupMocks:   func(mockGit *MockGitRunner, mockGh *MockGhRunner) {},
			wantErr:      true,
			errContains:  "plan cannot be nil",
		},
		{
			name: "fails when plan has no child PRs",
			dir:  "/test/repo",
			plan: &PRSplitPlan{
				Strategy:    SplitByCommits,
				ParentTitle: "Parent PR",
				ParentDesc:  "Description",
				ChildPRs:    []ChildPRPlan{},
			},
			sourceBranch: "feature-branch",
			mainBranch:   "main",
			setupMocks:   func(mockGit *MockGitRunner, mockGh *MockGhRunner) {},
			wantErr:      true,
			errContains:  "plan must have at least one child PR",
		},
		{
			name: "fails when sourceBranch is empty",
			dir:  "/test/repo",
			plan: &PRSplitPlan{
				Strategy:    SplitByCommits,
				ParentTitle: "Parent PR",
				ParentDesc:  "Description",
				ChildPRs:    []ChildPRPlan{{Title: "Child 1", Commits: []string{"abc123"}}},
			},
			sourceBranch: "",
			mainBranch:   "main",
			setupMocks:   func(mockGit *MockGitRunner, mockGh *MockGhRunner) {},
			wantErr:      true,
			errContains:  "sourceBranch cannot be empty",
		},
		{
			name: "fails when mainBranch is empty",
			dir:  "/test/repo",
			plan: &PRSplitPlan{
				Strategy:    SplitByCommits,
				ParentTitle: "Parent PR",
				ParentDesc:  "Description",
				ChildPRs:    []ChildPRPlan{{Title: "Child 1", Commits: []string{"abc123"}}},
			},
			sourceBranch: "feature-branch",
			mainBranch:   "",
			setupMocks:   func(mockGit *MockGitRunner, mockGh *MockGhRunner) {},
			wantErr:      true,
			errContains:  "mainBranch cannot be empty",
		},
		{
			name:         "fails when parent branch creation fails",
			dir:          "/test/repo",
			sourceBranch: "feature-branch",
			mainBranch:   "main",
			plan: &PRSplitPlan{
				Strategy:    SplitByCommits,
				ParentTitle: "Parent PR",
				ParentDesc:  "Description",
				ChildPRs:    []ChildPRPlan{{Title: "Child 1", Commits: []string{"abc123"}}},
			},
			setupMocks: func(mockGit *MockGitRunner, mockGh *MockGhRunner) {
				mockGit.On("CreateBranch", mock.Anything, "/test/repo", "split/feature-branch/parent", "main").
					Return(fmt.Errorf("branch already exists"))
			},
			wantErr:     true,
			errContains: "failed to create parent branch",
		},
		{
			name:         "fails when parent checkout fails",
			dir:          "/test/repo",
			sourceBranch: "feature-branch",
			mainBranch:   "main",
			plan: &PRSplitPlan{
				Strategy:    SplitByCommits,
				ParentTitle: "Parent PR",
				ParentDesc:  "Description",
				ChildPRs:    []ChildPRPlan{{Title: "Child 1", Commits: []string{"abc123"}}},
			},
			setupMocks: func(mockGit *MockGitRunner, mockGh *MockGhRunner) {
				mockGit.On("CreateBranch", mock.Anything, "/test/repo", "split/feature-branch/parent", "main").Return(nil)
				mockGit.On("CheckoutBranch", mock.Anything, "/test/repo", "split/feature-branch/parent").
					Return(fmt.Errorf("checkout failed"))
			},
			wantErr:     true,
			errContains: "failed to checkout parent branch",
		},
		{
			name:         "fails when parent commit fails",
			dir:          "/test/repo",
			sourceBranch: "feature-branch",
			mainBranch:   "main",
			plan: &PRSplitPlan{
				Strategy:    SplitByCommits,
				ParentTitle: "Parent PR",
				ParentDesc:  "Description",
				ChildPRs:    []ChildPRPlan{{Title: "Child 1", Commits: []string{"abc123"}}},
			},
			setupMocks: func(mockGit *MockGitRunner, mockGh *MockGhRunner) {
				mockGit.On("CreateBranch", mock.Anything, "/test/repo", "split/feature-branch/parent", "main").Return(nil)
				mockGit.On("CheckoutBranch", mock.Anything, "/test/repo", "split/feature-branch/parent").Return(nil)
				mockGit.On("CommitEmpty", mock.Anything, "/test/repo", "Parent PR for split: Parent PR").
					Return(fmt.Errorf("commit failed"))
			},
			wantErr:     true,
			errContains: "failed to create empty commit on parent branch",
		},
		{
			name:         "fails when parent push fails",
			dir:          "/test/repo",
			sourceBranch: "feature-branch",
			mainBranch:   "main",
			plan: &PRSplitPlan{
				Strategy:    SplitByCommits,
				ParentTitle: "Parent PR",
				ParentDesc:  "Description",
				ChildPRs:    []ChildPRPlan{{Title: "Child 1", Commits: []string{"abc123"}}},
			},
			setupMocks: func(mockGit *MockGitRunner, mockGh *MockGhRunner) {
				mockGit.On("CreateBranch", mock.Anything, "/test/repo", "split/feature-branch/parent", "main").Return(nil)
				mockGit.On("CheckoutBranch", mock.Anything, "/test/repo", "split/feature-branch/parent").Return(nil)
				mockGit.On("CommitEmpty", mock.Anything, "/test/repo", "Parent PR for split: Parent PR").Return(nil)
				mockGit.On("Push", mock.Anything, "/test/repo", "split/feature-branch/parent").
					Return(fmt.Errorf("push failed"))
			},
			wantErr:     true,
			errContains: "failed to push parent branch",
		},
		{
			name:         "fails when child branch creation fails",
			dir:          "/test/repo",
			sourceBranch: "feature-branch",
			mainBranch:   "main",
			plan: &PRSplitPlan{
				Strategy:    SplitByCommits,
				ParentTitle: "Parent PR",
				ParentDesc:  "Description",
				ChildPRs:    []ChildPRPlan{{Title: "Child 1", Commits: []string{"abc123"}}},
			},
			setupMocks: func(mockGit *MockGitRunner, mockGh *MockGhRunner) {
				mockGit.On("CreateBranch", mock.Anything, "/test/repo", "split/feature-branch/parent", "main").Return(nil)
				mockGit.On("CheckoutBranch", mock.Anything, "/test/repo", "split/feature-branch/parent").Return(nil)
				mockGit.On("CommitEmpty", mock.Anything, "/test/repo", "Parent PR for split: Parent PR").Return(nil)
				mockGit.On("Push", mock.Anything, "/test/repo", "split/feature-branch/parent").Return(nil)
				mockGit.On("CreateBranch", mock.Anything, "/test/repo", "split/feature-branch/child-1", "split/feature-branch/parent").
					Return(fmt.Errorf("branch creation failed"))
			},
			wantErr:     true,
			errContains: "failed to create child branch 1",
		},
		{
			name:         "fails when cherry-pick fails",
			dir:          "/test/repo",
			sourceBranch: "feature-branch",
			mainBranch:   "main",
			plan: &PRSplitPlan{
				Strategy:    SplitByCommits,
				ParentTitle: "Parent PR",
				ParentDesc:  "Description",
				ChildPRs:    []ChildPRPlan{{Title: "Child 1", Commits: []string{"abc123"}}},
			},
			setupMocks: func(mockGit *MockGitRunner, mockGh *MockGhRunner) {
				mockGit.On("CreateBranch", mock.Anything, "/test/repo", "split/feature-branch/parent", "main").Return(nil)
				mockGit.On("CheckoutBranch", mock.Anything, "/test/repo", "split/feature-branch/parent").Return(nil)
				mockGit.On("CommitEmpty", mock.Anything, "/test/repo", "Parent PR for split: Parent PR").Return(nil)
				mockGit.On("Push", mock.Anything, "/test/repo", "split/feature-branch/parent").Return(nil)
				mockGit.On("CreateBranch", mock.Anything, "/test/repo", "split/feature-branch/child-1", "split/feature-branch/parent").Return(nil)
				mockGit.On("CheckoutBranch", mock.Anything, "/test/repo", "split/feature-branch/child-1").Return(nil)
				mockGit.On("CherryPick", mock.Anything, "/test/repo", "abc123").
					Return(fmt.Errorf("cherry-pick conflict"))
			},
			wantErr:     true,
			errContains: "failed to apply changes to child branch 1",
		},
		{
			name:         "fails when parent PR creation fails",
			dir:          "/test/repo",
			sourceBranch: "feature-branch",
			mainBranch:   "main",
			plan: &PRSplitPlan{
				Strategy:    SplitByCommits,
				ParentTitle: "Parent PR",
				ParentDesc:  "Description",
				ChildPRs:    []ChildPRPlan{{Title: "Child 1", Commits: []string{"abc123"}}},
			},
			setupMocks: func(mockGit *MockGitRunner, mockGh *MockGhRunner) {
				mockGit.On("CreateBranch", mock.Anything, "/test/repo", "split/feature-branch/parent", "main").Return(nil)
				mockGit.On("CheckoutBranch", mock.Anything, "/test/repo", "split/feature-branch/parent").Return(nil)
				mockGit.On("CommitEmpty", mock.Anything, "/test/repo", "Parent PR for split: Parent PR").Return(nil)
				mockGit.On("Push", mock.Anything, "/test/repo", "split/feature-branch/parent").Return(nil)
				mockGit.On("CreateBranch", mock.Anything, "/test/repo", "split/feature-branch/child-1", "split/feature-branch/parent").Return(nil)
				mockGit.On("CheckoutBranch", mock.Anything, "/test/repo", "split/feature-branch/child-1").Return(nil)
				mockGit.On("CherryPick", mock.Anything, "/test/repo", "abc123").Return(nil)
				mockGit.On("Push", mock.Anything, "/test/repo", "split/feature-branch/child-1").Return(nil)
				mockGh.On("PRCreate", mock.Anything, "/test/repo", "Parent PR", "Description", "split/feature-branch/parent", "main").
					Return("", fmt.Errorf("PR creation failed"))
			},
			wantErr:     true,
			errContains: "failed to create parent PR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := new(MockGitRunner)
			mockGh := new(MockGhRunner)
			tt.setupMocks(mockGit, mockGh)

			manager := NewPRSplitManager(mockGit, mockGh)
			ctx := context.Background()

			_, err := manager.ExecuteSplit(ctx, tt.dir, tt.plan, tt.sourceBranch, tt.mainBranch)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
			mockGit.AssertExpectations(t)
			mockGh.AssertExpectations(t)
		})
	}
}

func TestPRSplitManager_Rollback(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		result      *PRSplitResult
		setupMocks  func(*MockGitRunner, *MockGhRunner)
		wantErr     bool
		errContains string
	}{
		{
			name: "successfully rolls back all PRs and branches",
			dir:  "/test/repo",
			result: &PRSplitResult{
				ParentPR: PRInfo{Number: 100},
				ChildPRs: []PRInfo{
					{Number: 101},
					{Number: 102},
				},
				BranchNames: []string{
					"split/feature-branch/parent",
					"split/feature-branch/child-1",
					"split/feature-branch/child-2",
				},
			},
			setupMocks: func(mockGit *MockGitRunner, mockGh *MockGhRunner) {
				mockGh.On("PRClose", mock.Anything, "/test/repo", 102).Return(nil)
				mockGh.On("PRClose", mock.Anything, "/test/repo", 101).Return(nil)
				mockGh.On("PRClose", mock.Anything, "/test/repo", 100).Return(nil)

				mockGit.On("DeleteRemoteBranch", mock.Anything, "/test/repo", "split/feature-branch/child-2").Return(nil)
				mockGit.On("DeleteRemoteBranch", mock.Anything, "/test/repo", "split/feature-branch/child-1").Return(nil)
				mockGit.On("DeleteRemoteBranch", mock.Anything, "/test/repo", "split/feature-branch/parent").Return(nil)

				mockGit.On("DeleteBranch", mock.Anything, "/test/repo", "split/feature-branch/child-2", true).Return(nil)
				mockGit.On("DeleteBranch", mock.Anything, "/test/repo", "split/feature-branch/child-1", true).Return(nil)
				mockGit.On("DeleteBranch", mock.Anything, "/test/repo", "split/feature-branch/parent", true).Return(nil)
			},
		},
		{
			name: "continues rollback despite PR close errors",
			dir:  "/test/repo",
			result: &PRSplitResult{
				ParentPR: PRInfo{Number: 100},
				ChildPRs: []PRInfo{
					{Number: 101},
				},
				BranchNames: []string{
					"split/feature-branch/parent",
					"split/feature-branch/child-1",
				},
			},
			setupMocks: func(mockGit *MockGitRunner, mockGh *MockGhRunner) {
				mockGh.On("PRClose", mock.Anything, "/test/repo", 101).Return(fmt.Errorf("PR already closed"))
				mockGh.On("PRClose", mock.Anything, "/test/repo", 100).Return(nil)

				mockGit.On("DeleteRemoteBranch", mock.Anything, "/test/repo", "split/feature-branch/child-1").Return(nil)
				mockGit.On("DeleteRemoteBranch", mock.Anything, "/test/repo", "split/feature-branch/parent").Return(nil)

				mockGit.On("DeleteBranch", mock.Anything, "/test/repo", "split/feature-branch/child-1", true).Return(nil)
				mockGit.On("DeleteBranch", mock.Anything, "/test/repo", "split/feature-branch/parent", true).Return(nil)
			},
			wantErr:     true,
			errContains: "rollback encountered errors",
		},
		{
			name: "continues rollback despite branch deletion errors",
			dir:  "/test/repo",
			result: &PRSplitResult{
				ParentPR: PRInfo{Number: 100},
				ChildPRs: []PRInfo{},
				BranchNames: []string{
					"split/feature-branch/parent",
				},
			},
			setupMocks: func(mockGit *MockGitRunner, mockGh *MockGhRunner) {
				mockGh.On("PRClose", mock.Anything, "/test/repo", 100).Return(nil)

				mockGit.On("DeleteRemoteBranch", mock.Anything, "/test/repo", "split/feature-branch/parent").
					Return(fmt.Errorf("branch not found"))
				mockGit.On("DeleteBranch", mock.Anything, "/test/repo", "split/feature-branch/parent", true).Return(nil)
			},
			wantErr:     true,
			errContains: "rollback encountered errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := new(MockGitRunner)
			mockGh := new(MockGhRunner)
			tt.setupMocks(mockGit, mockGh)

			manager := NewPRSplitManager(mockGit, mockGh)
			ctx := context.Background()

			err := manager.Rollback(ctx, tt.dir, tt.result)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				mockGit.AssertExpectations(t)
				mockGh.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			mockGit.AssertExpectations(t)
			mockGh.AssertExpectations(t)
		})
	}
}

func TestExtractPRNumber(t *testing.T) {
	tests := []struct {
		name        string
		prURL       string
		want        int
		wantErr     bool
		errContains string
	}{
		{
			name:  "extracts PR number from standard URL",
			prURL: "https://github.com/owner/repo/pull/123",
			want:  123,
		},
		{
			name:  "extracts PR number with trailing newline",
			prURL: "https://github.com/owner/repo/pull/456\n",
			want:  456,
		},
		{
			name:  "extracts PR number with spaces",
			prURL: "  https://github.com/owner/repo/pull/789  ",
			want:  789,
		},
		{
			name:  "extracts large PR number",
			prURL: "https://github.com/owner/repo/pull/12345",
			want:  12345,
		},
		{
			name:  "extracts PR number with query parameters",
			prURL: "https://github.com/owner/repo/pull/999?tab=checks",
			want:  999,
		},
		{
			name:  "extracts PR number with fragment",
			prURL: "https://github.com/owner/repo/pull/888#discussion_r123",
			want:  888,
		},
		{
			name:  "extracts PR number with trailing slash",
			prURL: "https://github.com/owner/repo/pull/777/",
			want:  777,
		},
		{
			name:        "fails with empty URL",
			prURL:       "",
			wantErr:     true,
			errContains: "PR URL is empty",
		},
		{
			name:        "fails with whitespace only",
			prURL:       "   \n  ",
			wantErr:     true,
			errContains: "PR URL is empty",
		},
		{
			name:        "fails with invalid URL format",
			prURL:       "https://github.com/owner/repo",
			wantErr:     true,
			errContains: "invalid PR URL format",
		},
		{
			name:        "fails with issues URL",
			prURL:       "https://github.com/owner/repo/issues/123",
			wantErr:     true,
			errContains: "invalid PR URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractPRNumber(tt.prURL)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Equal(t, 0, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateParentBranchName(t *testing.T) {
	tests := []struct {
		name         string
		sourceBranch string
		want         string
	}{
		{
			name:         "generates parent branch name for feature branch",
			sourceBranch: "feature-branch",
			want:         "split/feature-branch/parent",
		},
		{
			name:         "generates parent branch name for workflow branch",
			sourceBranch: "workflow/my-workflow",
			want:         "split/workflow/my-workflow/parent",
		},
		{
			name:         "generates parent branch name for simple branch",
			sourceBranch: "main",
			want:         "split/main/parent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateParentBranchName(tt.sourceBranch)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateChildBranchName(t *testing.T) {
	tests := []struct {
		name         string
		sourceBranch string
		index        int
		want         string
	}{
		{
			name:         "generates first child branch name",
			sourceBranch: "feature-branch",
			index:        0,
			want:         "split/feature-branch/child-1",
		},
		{
			name:         "generates second child branch name",
			sourceBranch: "feature-branch",
			index:        1,
			want:         "split/feature-branch/child-2",
		},
		{
			name:         "generates child branch name with workflow branch",
			sourceBranch: "workflow/my-workflow",
			index:        0,
			want:         "split/workflow/my-workflow/child-1",
		},
		{
			name:         "generates tenth child branch name",
			sourceBranch: "feature",
			index:        9,
			want:         "split/feature/child-10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateChildBranchName(tt.sourceBranch, tt.index)
			assert.Equal(t, tt.want, got)
		})
	}
}
