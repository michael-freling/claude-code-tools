package workflow

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewGhRunner(t *testing.T) {
	tests := []struct {
		name   string
		runner CommandRunner
	}{
		{
			name:   "creates gh runner with command runner",
			runner: &MockCommandRunner{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewGhRunner(tt.runner)
			require.NotNil(t, got)

			ghRunner, ok := got.(*ghRunner)
			require.True(t, ok)
			assert.Equal(t, tt.runner, ghRunner.runner)
		})
	}
}

func TestGhRunner_PRCreate(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		title       string
		body        string
		head        string
		base        string
		setupMock   func(*MockCommandRunner)
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name:  "creates PR successfully",
			dir:   "/test/repo",
			title: "Add new feature",
			body:  "This PR adds a new feature",
			head:  "feature-branch",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "create", "--title", "Add new feature", "--body", "This PR adds a new feature", "--head", "feature-branch").
					Return("https://github.com/owner/repo/pull/123", "", nil)
			},
			want: "https://github.com/owner/repo/pull/123",
		},
		{
			name:  "creates PR with multiline body",
			dir:   "/test/repo",
			title: "Fix bug",
			body:  "This PR fixes:\n- Issue 1\n- Issue 2",
			head:  "bugfix",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "create", "--title", "Fix bug", "--body", "This PR fixes:\n- Issue 1\n- Issue 2", "--head", "bugfix").
					Return("https://github.com/owner/repo/pull/456", "", nil)
			},
			want: "https://github.com/owner/repo/pull/456",
		},
		{
			name:  "creates PR with empty body",
			dir:   "/test/repo",
			title: "Update README",
			body:  "",
			head:  "docs",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "create", "--title", "Update README", "--body", "", "--head", "docs").
					Return("https://github.com/owner/repo/pull/789", "", nil)
			},
			want: "https://github.com/owner/repo/pull/789",
		},
		{
			name:  "returns error when PR creation fails",
			dir:   "/test/repo",
			title: "Test PR",
			body:  "Test body",
			head:  "test-branch",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "create", "--title", "Test PR", "--body", "Test body", "--head", "test-branch").
					Return("", "error: pull request create failed", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to create PR",
		},
		{
			name:  "includes stderr in error message",
			dir:   "/test/repo",
			title: "Failed PR",
			body:  "This will fail",
			head:  "failing-branch",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "create", "--title", "Failed PR", "--body", "This will fail", "--head", "failing-branch").
					Return("", "GraphQL: Base branch does not exist", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "stderr: GraphQL: Base branch does not exist",
		},
		{
			name:  "handles special characters in title",
			dir:   "/test/repo",
			title: "Fix: Handle \"quotes\" and 'apostrophes'",
			body:  "Description",
			head:  "fix-quotes",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "create", "--title", "Fix: Handle \"quotes\" and 'apostrophes'", "--body", "Description", "--head", "fix-quotes").
					Return("https://github.com/owner/repo/pull/999", "", nil)
			},
			want: "https://github.com/owner/repo/pull/999",
		},
		{
			name:  "creates PR with base parameter",
			dir:   "/test/repo",
			title: "Add feature",
			body:  "Feature description",
			head:  "feature-branch",
			base:  "develop",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "create", "--title", "Add feature", "--body", "Feature description", "--head", "feature-branch", "--base", "develop").
					Return("https://github.com/owner/repo/pull/100", "", nil)
			},
			want: "https://github.com/owner/repo/pull/100",
		},
		{
			name:  "creates PR without base parameter when empty",
			dir:   "/test/repo",
			title: "Update code",
			body:  "Code update",
			head:  "update-branch",
			base:  "",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "create", "--title", "Update code", "--body", "Code update", "--head", "update-branch").
					Return("https://github.com/owner/repo/pull/200", "", nil)
			},
			want: "https://github.com/owner/repo/pull/200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			ghRunner := NewGhRunner(mockRunner)
			ctx := context.Background()

			got, err := ghRunner.PRCreate(ctx, tt.dir, tt.title, tt.body, tt.head, tt.base)

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

func TestGhRunner_PRView(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		jsonFields  string
		jqQuery     string
		setupMock   func(*MockCommandRunner)
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name:       "views PR number successfully",
			dir:        "/test/repo",
			jsonFields: "number",
			jqQuery:    ".number",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "view", "--json", "number", "-q", ".number").
					Return("123", "", nil)
			},
			want: "123",
		},
		{
			name:       "views PR title",
			dir:        "/test/repo",
			jsonFields: "title",
			jqQuery:    ".title",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "view", "--json", "title", "-q", ".title").
					Return("Add new feature", "", nil)
			},
			want: "Add new feature",
		},
		{
			name:       "views multiple fields",
			dir:        "/test/repo",
			jsonFields: "number,title,state",
			jqQuery:    ".number",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "view", "--json", "number,title,state", "-q", ".number").
					Return("456", "", nil)
			},
			want: "456",
		},
		{
			name:       "returns error when no PR found",
			dir:        "/test/repo",
			jsonFields: "number",
			jqQuery:    ".number",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "view", "--json", "number", "-q", ".number").
					Return("", "no pull requests found for branch", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to view PR",
		},
		{
			name:       "includes stderr in error message",
			dir:        "/test/repo",
			jsonFields: "number",
			jqQuery:    ".number",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "view", "--json", "number", "-q", ".number").
					Return("", "GraphQL error: Could not resolve to a PullRequest", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "stderr: GraphQL error: Could not resolve to a PullRequest",
		},
		{
			name:       "handles complex jq query",
			dir:        "/test/repo",
			jsonFields: "reviews",
			jqQuery:    ".reviews | length",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "view", "--json", "reviews", "-q", ".reviews | length").
					Return("3", "", nil)
			},
			want: "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			ghRunner := NewGhRunner(mockRunner)
			ctx := context.Background()

			got, err := ghRunner.PRView(ctx, tt.dir, tt.jsonFields, tt.jqQuery)

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

func TestGhRunner_PRChecks(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		prNumber    int
		jsonFields  string
		setupMock   func(*MockCommandRunner)
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name:       "checks current branch PR (prNumber=0)",
			dir:        "/test/repo",
			prNumber:   0,
			jsonFields: "name,state",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "checks", "--json", "name,state").
					Return(`[{"name":"build","state":"SUCCESS"},{"name":"test","state":"SUCCESS"}]`, "", nil)
			},
			want: `[{"name":"build","state":"SUCCESS"},{"name":"test","state":"SUCCESS"}]`,
		},
		{
			name:       "checks specific PR number",
			dir:        "/test/repo",
			prNumber:   123,
			jsonFields: "name,state",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "checks", "123", "--json", "name,state").
					Return(`[{"name":"build","state":"FAILURE"}]`, "", nil)
			},
			want: `[{"name":"build","state":"FAILURE"}]`,
		},
		{
			name:       "checks PR with pending status",
			dir:        "/test/repo",
			prNumber:   456,
			jsonFields: "name,state",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "checks", "456", "--json", "name,state").
					Return(`[{"name":"build","state":"PENDING"}]`, "", nil)
			},
			want: `[{"name":"build","state":"PENDING"}]`,
		},
		{
			name:       "checks PR with multiple fields",
			dir:        "/test/repo",
			prNumber:   789,
			jsonFields: "name,state,startedAt,completedAt",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "checks", "789", "--json", "name,state,startedAt,completedAt").
					Return(`[{"name":"build","state":"SUCCESS","startedAt":"2023-01-01","completedAt":"2023-01-01"}]`, "", nil)
			},
			want: `[{"name":"build","state":"SUCCESS","startedAt":"2023-01-01","completedAt":"2023-01-01"}]`,
		},
		{
			name:       "returns error when checks fail",
			dir:        "/test/repo",
			prNumber:   999,
			jsonFields: "name,state",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "checks", "999", "--json", "name,state").
					Return("", "no pull request found", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to check PR status",
		},
		{
			name:       "includes stderr in error message",
			dir:        "/test/repo",
			prNumber:   0,
			jsonFields: "name,state",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "checks", "--json", "name,state").
					Return("", "GraphQL: Could not resolve to a PullRequest", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "stderr: GraphQL: Could not resolve to a PullRequest",
		},
		{
			name:       "handles empty checks response",
			dir:        "/test/repo",
			prNumber:   0,
			jsonFields: "name,state",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "checks", "--json", "name,state").
					Return("[]", "", nil)
			},
			want: "[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			ghRunner := NewGhRunner(mockRunner)
			ctx := context.Background()

			got, err := ghRunner.PRChecks(ctx, tt.dir, tt.prNumber, tt.jsonFields)

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

func TestGhRunner_RunRerun(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		runID       int64
		setupMock   func(*MockCommandRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:  "reruns workflow successfully",
			dir:   "/test/repo",
			runID: 123456789,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "run", "rerun", "123456789", "--failed").
					Return("", "", nil)
			},
		},
		{
			name:  "reruns with different run ID",
			dir:   "/test/repo",
			runID: 987654321,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "run", "rerun", "987654321", "--failed").
					Return("", "", nil)
			},
		},
		{
			name:  "returns error when rerun fails",
			dir:   "/test/repo",
			runID: 111222333,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "run", "rerun", "111222333", "--failed").
					Return("", "workflow run not found", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to rerun workflow",
		},
		{
			name:  "includes stderr in error message",
			dir:   "/test/repo",
			runID: 444555666,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "run", "rerun", "444555666", "--failed").
					Return("", "GraphQL: Could not resolve to a WorkflowRun", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "stderr: GraphQL: Could not resolve to a WorkflowRun",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			ghRunner := NewGhRunner(mockRunner)
			ctx := context.Background()

			err := ghRunner.RunRerun(ctx, tt.dir, tt.runID)

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

func TestGhRunner_GetLatestRunID(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		prNumber    int
		setupMock   func(*MockCommandRunner)
		want        int64
		wantErr     bool
		errContains string
	}{
		{
			name:     "gets latest run ID successfully",
			dir:      "/test/repo",
			prNumber: 123,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "checks", "123", "--json", "databaseId").
					Return(`[{"databaseId":987654321},{"databaseId":123456789}]`, "", nil)
			},
			want: 987654321,
		},
		{
			name:     "gets run ID for different PR",
			dir:      "/test/repo",
			prNumber: 456,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "checks", "456", "--json", "databaseId").
					Return(`[{"databaseId":111222333}]`, "", nil)
			},
			want: 111222333,
		},
		{
			name:     "returns error when command fails",
			dir:      "/test/repo",
			prNumber: 789,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "checks", "789", "--json", "databaseId").
					Return("", "no pull request found", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to get workflow run ID",
		},
		{
			name:     "returns error when JSON parsing fails",
			dir:      "/test/repo",
			prNumber: 999,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "checks", "999", "--json", "databaseId").
					Return("invalid json", "", nil)
			},
			wantErr:     true,
			errContains: "failed to parse workflow run ID from output",
		},
		{
			name:     "returns error when no runs found",
			dir:      "/test/repo",
			prNumber: 111,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "checks", "111", "--json", "databaseId").
					Return("[]", "", nil)
			},
			wantErr:     true,
			errContains: "no workflow runs found for PR 111",
		},
		{
			name:     "includes stderr in error message",
			dir:      "/test/repo",
			prNumber: 222,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "checks", "222", "--json", "databaseId").
					Return("", "GraphQL: Could not resolve to a PullRequest", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "stderr: GraphQL: Could not resolve to a PullRequest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			ghRunner := NewGhRunner(mockRunner)
			ctx := context.Background()

			got, err := ghRunner.GetLatestRunID(ctx, tt.dir, tt.prNumber)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Equal(t, int64(0), got)
				mockRunner.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			mockRunner.AssertExpectations(t)
		})
	}
}

func TestGhRunner_PREdit(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		prNumber    int
		body        string
		setupMock   func(*MockCommandRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:     "edits PR successfully",
			dir:      "/test/repo",
			prNumber: 123,
			body:     "Updated PR description",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "edit", "123", "--body", "Updated PR description").
					Return("", "", nil)
			},
		},
		{
			name:     "edits PR with multiline body",
			dir:      "/test/repo",
			prNumber: 456,
			body:     "Updated description:\n- Point 1\n- Point 2",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "edit", "456", "--body", "Updated description:\n- Point 1\n- Point 2").
					Return("", "", nil)
			},
		},
		{
			name:     "edits PR with empty body",
			dir:      "/test/repo",
			prNumber: 789,
			body:     "",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "edit", "789", "--body", "").
					Return("", "", nil)
			},
		},
		{
			name:     "returns error when edit fails",
			dir:      "/test/repo",
			prNumber: 999,
			body:     "New body",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "edit", "999", "--body", "New body").
					Return("", "pull request not found", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to edit PR 999",
		},
		{
			name:     "includes stderr in error message",
			dir:      "/test/repo",
			prNumber: 111,
			body:     "Updated body",
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "edit", "111", "--body", "Updated body").
					Return("", "GraphQL: Could not resolve to a PullRequest", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "stderr: GraphQL: Could not resolve to a PullRequest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			ghRunner := NewGhRunner(mockRunner)
			ctx := context.Background()

			err := ghRunner.PREdit(ctx, tt.dir, tt.prNumber, tt.body)

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

func TestGhRunner_PRClose(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		prNumber    int
		setupMock   func(*MockCommandRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:     "closes PR successfully",
			dir:      "/test/repo",
			prNumber: 123,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "close", "123").
					Return("", "", nil)
			},
		},
		{
			name:     "closes different PR",
			dir:      "/test/repo",
			prNumber: 456,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "close", "456").
					Return("", "", nil)
			},
		},
		{
			name:     "returns error when close fails",
			dir:      "/test/repo",
			prNumber: 999,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "close", "999").
					Return("", "pull request not found", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "failed to close PR 999",
		},
		{
			name:     "includes stderr in error message",
			dir:      "/test/repo",
			prNumber: 111,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "close", "111").
					Return("", "GraphQL: Could not resolve to a PullRequest", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "stderr: GraphQL: Could not resolve to a PullRequest",
		},
		{
			name:     "handles already closed PR",
			dir:      "/test/repo",
			prNumber: 222,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunInDir", mock.Anything, "/test/repo", "gh", "pr", "close", "222").
					Return("", "pull request already closed", fmt.Errorf("exit status 1"))
			},
			wantErr:     true,
			errContains: "stderr: pull request already closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)
			tt.setupMock(mockRunner)

			ghRunner := NewGhRunner(mockRunner)
			ctx := context.Background()

			err := ghRunner.PRClose(ctx, tt.dir, tt.prNumber)

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
