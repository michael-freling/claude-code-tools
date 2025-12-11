package hooks

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGitPushRule(t *testing.T) {
	mockGit := &MockGitHelper{}
	rule := NewGitPushRule(mockGit)
	assert.NotNil(t, rule)
	assert.Equal(t, "git-push", rule.Name())
	assert.Equal(t, "Blocks git push commands to main/master branches", rule.Description())
}

func TestGitPushRule_Evaluate_NonBashTool(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		command  string
	}{
		{
			name:     "allow Write tool",
			toolName: "Write",
			command:  "git push origin main",
		},
		{
			name:     "allow Read tool",
			toolName: "Read",
			command:  "git push origin master",
		},
		{
			name:     "allow Edit tool",
			toolName: "Edit",
			command:  "git push",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitHelper{}
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "` + tt.toolName + `", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.True(t, got.Allowed)

			mockGit.AssertNotCalled(t, "GetCurrentBranch")
		})
	}
}

func TestGitPushRule_Evaluate_ExplicitPushToProtectedBranch(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "block git push origin main",
			command: "git push origin main",
		},
		{
			name:    "block git push origin master",
			command: "git push origin master",
		},
		{
			name:    "block git push -u origin main",
			command: "git push -u origin main",
		},
		{
			name:    "block git push --set-upstream origin main",
			command: "git push --set-upstream origin main",
		},
		{
			name:    "block git push -f origin main",
			command: "git push -f origin main",
		},
		{
			name:    "block git push --force origin main",
			command: "git push --force origin main",
		},
		{
			name:    "block git push -u origin master",
			command: "git push -u origin master",
		},
		{
			name:    "block git push --set-upstream origin master",
			command: "git push --set-upstream origin master",
		},
		{
			name:    "block git push -f origin master",
			command: "git push -f origin master",
		},
		{
			name:    "block git push --force origin master",
			command: "git push --force origin master",
		},
		{
			name:    "block git push with multiple flags and main",
			command: "git push -u --force origin main",
		},
		{
			name:    "block git push with tabs",
			command: "git\tpush\torigin\tmain",
		},
		{
			name:    "block git push with extra spaces",
			command: "git  push  origin  main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitHelper{}
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.False(t, got.Allowed)
			assert.Equal(t, "git-push", got.RuleName)
			assert.Equal(t, "Direct push to main/master branch is not allowed", got.Message)

			mockGit.AssertNotCalled(t, "GetCurrentBranch")
		})
	}
}

func TestGitPushRule_Evaluate_ImplicitPushOnProtectedBranch(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		currentBranch string
	}{
		{
			name:          "block git push when on main",
			command:       "git push",
			currentBranch: "main",
		},
		{
			name:          "block git push when on master",
			command:       "git push",
			currentBranch: "master",
		},
		{
			name:          "block git push origin when on main",
			command:       "git push origin",
			currentBranch: "main",
		},
		{
			name:          "block git push origin when on master",
			command:       "git push origin",
			currentBranch: "master",
		},
		{
			name:          "block git push -u origin when on main",
			command:       "git push -u origin",
			currentBranch: "main",
		},
		{
			name:          "block git push --set-upstream origin when on main",
			command:       "git push --set-upstream origin",
			currentBranch: "main",
		},
		{
			name:          "block git push -f when on main",
			command:       "git push -f",
			currentBranch: "main",
		},
		{
			name:          "block git push --force when on master",
			command:       "git push --force",
			currentBranch: "master",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitHelper{}
			mockGit.On("GetCurrentBranch").Return(tt.currentBranch, nil)
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.False(t, got.Allowed)
			assert.Equal(t, "git-push", got.RuleName)
			assert.Equal(t, "Direct push to main/master branch is not allowed", got.Message)

			mockGit.AssertExpectations(t)
		})
	}
}

func TestGitPushRule_Evaluate_ImplicitPushOnFeatureBranch(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		currentBranch string
	}{
		{
			name:          "allow git push on feature branch",
			command:       "git push",
			currentBranch: "feature-branch",
		},
		{
			name:          "allow git push origin on bugfix branch",
			command:       "git push origin",
			currentBranch: "bugfix/123",
		},
		{
			name:          "allow git push -u origin on dev branch",
			command:       "git push -u origin",
			currentBranch: "dev",
		},
		{
			name:          "allow git push on branch with main in name",
			command:       "git push",
			currentBranch: "main-feature",
		},
		{
			name:          "allow git push on branch with master in name",
			command:       "git push",
			currentBranch: "master-copy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitHelper{}
			mockGit.On("GetCurrentBranch").Return(tt.currentBranch, nil)
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.True(t, got.Allowed)

			mockGit.AssertExpectations(t)
		})
	}
}

func TestGitPushRule_Evaluate_ExplicitPushToFeatureBranch(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "allow git push origin feature-branch",
			command: "git push origin feature-branch",
		},
		{
			name:    "allow git push -u origin bugfix/123",
			command: "git push -u origin bugfix/123",
		},
		{
			name:    "allow git push --set-upstream origin dev",
			command: "git push --set-upstream origin dev",
		},
		{
			name:    "allow git push -f origin hotfix",
			command: "git push -f origin hotfix",
		},
		{
			name:    "allow git push --force origin release/1.0",
			command: "git push --force origin release/1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitHelper{}
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.True(t, got.Allowed)

			mockGit.AssertNotCalled(t, "GetCurrentBranch")
		})
	}
}

func TestGitPushRule_Evaluate_GetCurrentBranchError(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "allow git push when GetCurrentBranch fails",
			command: "git push",
		},
		{
			name:    "allow git push origin when GetCurrentBranch fails",
			command: "git push origin",
		},
		{
			name:    "allow git push -u origin when GetCurrentBranch fails",
			command: "git push -u origin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitHelper{}
			mockGit.On("GetCurrentBranch").Return("", errors.New("not in a git repository"))
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.True(t, got.Allowed, "should allow when GetCurrentBranch fails (fail open)")

			mockGit.AssertExpectations(t)
		})
	}
}

func TestGitPushRule_Evaluate_NonGitPushCommands(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "allow git commit",
			command: "git commit -m 'test'",
		},
		{
			name:    "allow git pull",
			command: "git pull origin main",
		},
		{
			name:    "allow git status",
			command: "git status",
		},
		{
			name:    "allow echo git push",
			command: "echo 'git push origin main'",
		},
		{
			name:    "allow empty command",
			command: "",
		},
		{
			name:    "allow non-git command",
			command: "ls -la",
		},
		{
			name:    "allow git push in string",
			command: "echo 'do not git push to main'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitHelper{}
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.True(t, got.Allowed)

			mockGit.AssertNotCalled(t, "GetCurrentBranch")
		})
	}
}

func TestGitPushRule_Evaluate_NoCommandArg(t *testing.T) {
	mockGit := &MockGitHelper{}
	rule := NewGitPushRule(mockGit)

	jsonInput := `{"tool_name": "Bash", "tool_input": {}}`
	reader := strings.NewReader(jsonInput)
	toolInput, err := ParseToolInput(reader)
	require.NoError(t, err)

	got, err := rule.Evaluate(toolInput)
	require.NoError(t, err)
	assert.True(t, got.Allowed)

	mockGit.AssertNotCalled(t, "GetCurrentBranch")
}

func TestGitPushRule_Evaluate_QuotedBranchNames(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		wantAllowed bool
	}{
		{
			name:        "block git push with quoted main",
			command:     "git push origin 'main'",
			wantAllowed: false,
		},
		{
			name:        "block git push with double-quoted master",
			command:     `git push origin "master"`,
			wantAllowed: false,
		},
		{
			name:        "allow git push with quoted feature branch",
			command:     "git push origin 'feature-branch'",
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitHelper{}
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAllowed, got.Allowed)

			mockGit.AssertNotCalled(t, "GetCurrentBranch")
		})
	}
}

func TestParseGitPushArgs(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    []string
	}{
		{
			name:    "simple git push",
			command: "git push",
			want:    []string{"git", "push"},
		},
		{
			name:    "git push with remote and branch",
			command: "git push origin main",
			want:    []string{"git", "push", "origin", "main"},
		},
		{
			name:    "git push with flags",
			command: "git push -u origin main",
			want:    []string{"git", "push", "-u", "origin", "main"},
		},
		{
			name:    "git push with multiple flags",
			command: "git push -u --force origin main",
			want:    []string{"git", "push", "-u", "--force", "origin", "main"},
		},
		{
			name:    "git push with quoted branch",
			command: "git push origin 'main'",
			want:    []string{"git", "push", "origin", "main"},
		},
		{
			name:    "git push with double-quoted branch",
			command: `git push origin "master"`,
			want:    []string{"git", "push", "origin", "master"},
		},
		{
			name:    "git push with extra spaces",
			command: "git  push  origin  main",
			want:    []string{"git", "push", "origin", "main"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGitPushArgs(tt.command)
			assert.Equal(t, tt.want, got)
		})
	}
}
