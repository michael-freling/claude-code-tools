package hooks

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/michael-freling/claude-code-tools/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewGitPushRule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGit := command.NewMockGitRunner(ctrl)
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "` + tt.toolName + `", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.True(t, got.Allowed)

			// No expectations set, so gomock will verify no calls were made
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
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

			// No expectations set, so gomock will verify no calls were made
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
			mockGit.EXPECT().GetCurrentBranch(context.Background(), "").Return(tt.currentBranch, nil)
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
			mockGit.EXPECT().GetCurrentBranch(context.Background(), "").Return(tt.currentBranch, nil)
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.True(t, got.Allowed)
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.True(t, got.Allowed)

			// No expectations set, so gomock will verify no calls were made
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
			mockGit.EXPECT().GetCurrentBranch(context.Background(), "").Return("", errors.New("not in a git repository"))
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.True(t, got.Allowed, "should allow when GetCurrentBranch fails (fail open)")
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.True(t, got.Allowed)

			// No expectations set, so gomock will verify no calls were made
		})
	}
}

func TestGitPushRule_Evaluate_NoCommandArg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGit := command.NewMockGitRunner(ctrl)
	rule := NewGitPushRule(mockGit)

	jsonInput := `{"tool_name": "Bash", "tool_input": {}}`
	reader := strings.NewReader(jsonInput)
	toolInput, err := ParseToolInput(reader)
	require.NoError(t, err)

	got, err := rule.Evaluate(toolInput)
	require.NoError(t, err)
	assert.True(t, got.Allowed)

	// No expectations set, so gomock will verify no calls were made
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAllowed, got.Allowed)

			// No expectations set, so gomock will verify no calls were made
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

func TestGitPushRule_Evaluate_RefspecForcePush(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		wantAllowed bool
		wantMessage string
	}{
		{
			name:        "block git push origin +main:main",
			command:     "git push origin +main:main",
			wantAllowed: false,
			wantMessage: "Force push to main/master branch is not allowed",
		},
		{
			name:        "block git push origin +main",
			command:     "git push origin +main",
			wantAllowed: false,
			wantMessage: "Force push to main/master branch is not allowed",
		},
		{
			name:        "block git push origin +HEAD:main",
			command:     "git push origin +HEAD:main",
			wantAllowed: false,
			wantMessage: "Force push to main/master branch is not allowed",
		},
		{
			name:        "block git push origin +HEAD:master",
			command:     "git push origin +HEAD:master",
			wantAllowed: false,
			wantMessage: "Force push to main/master branch is not allowed",
		},
		{
			name:        "block git push origin +feature:main",
			command:     "git push origin +feature:main",
			wantAllowed: false,
			wantMessage: "Force push to main/master branch is not allowed",
		},
		{
			name:        "block git push origin feature:main (non-force but targets main)",
			command:     "git push origin feature:main",
			wantAllowed: false,
			wantMessage: "Direct push to main/master branch is not allowed",
		},
		{
			name:        "block git push origin feature:master",
			command:     "git push origin feature:master",
			wantAllowed: false,
			wantMessage: "Direct push to main/master branch is not allowed",
		},
		{
			name:        "allow git push origin +feature:feature",
			command:     "git push origin +feature:feature",
			wantAllowed: true,
		},
		{
			name:        "allow git push origin feature:feature",
			command:     "git push origin feature:feature",
			wantAllowed: true,
		},
		{
			name:        "allow git push origin +develop",
			command:     "git push origin +develop",
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAllowed, got.Allowed)
			if !tt.wantAllowed {
				assert.Equal(t, "git-push", got.RuleName)
				assert.Equal(t, tt.wantMessage, got.Message)
			}
		})
	}
}

func TestGitPushRule_Evaluate_PushAll(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "block git push --all",
			command: "git push --all",
		},
		{
			name:    "block git push --all origin",
			command: "git push --all origin",
		},
		{
			name:    "block git push origin --all",
			command: "git push origin --all",
		},
		{
			name:    "block git push --mirror",
			command: "git push --mirror",
		},
		{
			name:    "block git push --mirror origin",
			command: "git push --mirror origin",
		},
		{
			name:    "block git push origin --mirror",
			command: "git push origin --mirror",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.False(t, got.Allowed)
			assert.Equal(t, "git-push", got.RuleName)
			assert.Equal(t, "Push --all/--mirror includes protected branches and is not allowed", got.Message)
		})
	}
}

func TestGitPushRule_Evaluate_DeleteOperations(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		wantAllowed bool
		wantMessage string
	}{
		{
			name:        "block git push origin :main",
			command:     "git push origin :main",
			wantAllowed: false,
			wantMessage: "Deleting main/master branch is not allowed",
		},
		{
			name:        "block git push origin :master",
			command:     "git push origin :master",
			wantAllowed: false,
			wantMessage: "Deleting main/master branch is not allowed",
		},
		{
			name:        "block git push --delete origin main",
			command:     "git push --delete origin main",
			wantAllowed: false,
			wantMessage: "Deleting main/master branch is not allowed",
		},
		{
			name:        "block git push -d origin main",
			command:     "git push -d origin main",
			wantAllowed: false,
			wantMessage: "Deleting main/master branch is not allowed",
		},
		{
			name:        "block git push -d origin master",
			command:     "git push -d origin master",
			wantAllowed: false,
			wantMessage: "Deleting main/master branch is not allowed",
		},
		{
			name:        "allow git push --delete origin feature",
			command:     "git push --delete origin feature",
			wantAllowed: true,
		},
		{
			name:        "allow git push -d origin feature",
			command:     "git push -d origin feature",
			wantAllowed: true,
		},
		{
			name:        "allow git push origin :feature",
			command:     "git push origin :feature",
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAllowed, got.Allowed)
			if !tt.wantAllowed {
				assert.Equal(t, "git-push", got.RuleName)
				assert.Equal(t, tt.wantMessage, got.Message)
			}
		})
	}
}

func TestGitPushRule_Evaluate_CombinedFlags(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		wantAllowed bool
		wantMessage string
	}{
		{
			name:        "block git push --verbose --force origin main",
			command:     "git push --verbose --force origin main",
			wantAllowed: false,
			wantMessage: "Direct push to main/master branch is not allowed",
		},
		{
			name:        "block git push -v -f origin main",
			command:     "git push -v -f origin main",
			wantAllowed: false,
			wantMessage: "Direct push to main/master branch is not allowed",
		},
		{
			name:        "block git push -v origin +main",
			command:     "git push -v origin +main",
			wantAllowed: false,
			wantMessage: "Force push to main/master branch is not allowed",
		},
		{
			name:        "allow git push -v -f origin feature",
			command:     "git push -v -f origin feature",
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAllowed, got.Allowed)
			if !tt.wantAllowed {
				assert.Equal(t, "git-push", got.RuleName)
				assert.Equal(t, tt.wantMessage, got.Message)
			}
		})
	}
}

func TestGitPushRule_Evaluate_CommandChainingBypass(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		wantAllowed bool
		wantMessage string
	}{
		{
			name:        "block git fetch && git push --force origin main",
			command:     "git fetch && git push --force origin main",
			wantAllowed: false,
			wantMessage: "Direct push to main/master branch is not allowed",
		},
		{
			name:        "block true; git push -f origin main",
			command:     "true; git push -f origin main",
			wantAllowed: false,
			wantMessage: "Direct push to main/master branch is not allowed",
		},
		{
			name:        "block git status || git push origin +main",
			command:     "git status || git push origin +main",
			wantAllowed: false,
			wantMessage: "Force push to main/master branch is not allowed",
		},
		{
			name:        "allow echo foo && echo bar",
			command:     "echo foo && echo bar",
			wantAllowed: true,
		},
		{
			name:        "block multiple chained with git push at end",
			command:     "git status && git fetch && git push origin main",
			wantAllowed: false,
			wantMessage: "Direct push to main/master branch is not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAllowed, got.Allowed)
			if !tt.wantAllowed {
				assert.Equal(t, "git-push", got.RuleName)
				assert.Equal(t, tt.wantMessage, got.Message)
			}
		})
	}
}

func TestGitPushRule_Evaluate_PipeBypass(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		wantAllowed bool
		wantMessage string
	}{
		{
			name:        "block git push --force origin main | cat",
			command:     "git push --force origin main | cat",
			wantAllowed: false,
			wantMessage: "Direct push to main/master branch is not allowed",
		},
		{
			name:        "block git push origin main 2>&1 | tee log.txt",
			command:     "git push origin main 2>&1 | tee log.txt",
			wantAllowed: false,
			wantMessage: "Direct push to main/master branch is not allowed",
		},
		{
			name:        "block git push -f origin master | grep output",
			command:     "git push -f origin master | grep output",
			wantAllowed: false,
			wantMessage: "Direct push to main/master branch is not allowed",
		},
		{
			name:        "allow ls | grep foo",
			command:     "ls | grep foo",
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAllowed, got.Allowed)
			if !tt.wantAllowed {
				assert.Equal(t, "git-push", got.RuleName)
				assert.Equal(t, tt.wantMessage, got.Message)
			}
		})
	}
}

func TestGitPushRule_Evaluate_SubshellBypass(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		wantAllowed bool
		wantMessage string
	}{
		{
			name:        "block (git push --force origin main)",
			command:     "(git push --force origin main)",
			wantAllowed: false,
			wantMessage: "Direct push to main/master branch is not allowed",
		},
		{
			name:        "block ( git push -f origin main )",
			command:     "( git push -f origin main )",
			wantAllowed: false,
			wantMessage: "Direct push to main/master branch is not allowed",
		},
		{
			name:        "block ((git push origin master))",
			command:     "((git push origin master))",
			wantAllowed: false,
			wantMessage: "Direct push to main/master branch is not allowed",
		},
		{
			name:        "allow (echo test)",
			command:     "(echo test)",
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAllowed, got.Allowed)
			if !tt.wantAllowed {
				assert.Equal(t, "git-push", got.RuleName)
				assert.Equal(t, tt.wantMessage, got.Message)
			}
		})
	}
}

func TestGitPushRule_Evaluate_BackgroundingBypass(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		wantAllowed bool
		wantMessage string
	}{
		{
			name:        "block git push --force origin main &",
			command:     "git push --force origin main &",
			wantAllowed: false,
			wantMessage: "Direct push to main/master branch is not allowed",
		},
		{
			name:        "block git push -f origin master &",
			command:     "git push -f origin master &",
			wantAllowed: false,
			wantMessage: "Direct push to main/master branch is not allowed",
		},
		{
			name:        "allow echo test &",
			command:     "echo test &",
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAllowed, got.Allowed)
			if !tt.wantAllowed {
				assert.Equal(t, "git-push", got.RuleName)
				assert.Equal(t, tt.wantMessage, got.Message)
			}
		})
	}
}

func TestGitPushRule_Evaluate_FullRefPathBypass(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		wantAllowed bool
		wantMessage string
	}{
		{
			name:        "block git push origin +HEAD:refs/heads/main",
			command:     "git push origin +HEAD:refs/heads/main",
			wantAllowed: false,
			wantMessage: "Force push to main/master branch is not allowed",
		},
		{
			name:        "block git push origin :refs/heads/main (delete)",
			command:     "git push origin :refs/heads/main",
			wantAllowed: false,
			wantMessage: "Deleting main/master branch is not allowed",
		},
		{
			name:        "block git push origin feature:refs/heads/main",
			command:     "git push origin feature:refs/heads/main",
			wantAllowed: false,
			wantMessage: "Direct push to main/master branch is not allowed",
		},
		{
			name:        "block git push origin +HEAD:refs/heads/master",
			command:     "git push origin +HEAD:refs/heads/master",
			wantAllowed: false,
			wantMessage: "Force push to main/master branch is not allowed",
		},
		{
			name:        "block git push origin :refs/heads/master (delete)",
			command:     "git push origin :refs/heads/master",
			wantAllowed: false,
			wantMessage: "Deleting main/master branch is not allowed",
		},
		{
			name:        "allow git push origin +HEAD:refs/heads/feature",
			command:     "git push origin +HEAD:refs/heads/feature",
			wantAllowed: true,
		},
		{
			name:        "allow git push origin :refs/heads/feature (delete)",
			command:     "git push origin :refs/heads/feature",
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := command.NewMockGitRunner(ctrl)
			rule := NewGitPushRule(mockGit)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAllowed, got.Allowed)
			if !tt.wantAllowed {
				assert.Equal(t, "git-push", got.RuleName)
				assert.Equal(t, tt.wantMessage, got.Message)
			}
		})
	}
}
