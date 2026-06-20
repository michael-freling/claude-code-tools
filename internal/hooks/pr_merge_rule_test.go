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

func TestNewPRMergeRule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGh := command.NewMockGhRunner(ctrl)
	rule := NewPRMergeRule(mockGh)
	assert.NotNil(t, rule)
	assert.Equal(t, "gh-pr-merge", rule.Name())
	assert.Equal(t, "Blocks PR merge commands to main/master branches", rule.Description())
}

func TestPRMergeRule_Evaluate_NonBashTool(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		command  string
	}{
		{
			name:     "allow Write tool",
			toolName: "Write",
			command:  "gh pr merge 123",
		},
		{
			name:     "allow Read tool",
			toolName: "Read",
			command:  "gh api -X PUT repos/owner/repo/pulls/123/merge",
		},
		{
			name:     "allow Edit tool",
			toolName: "Edit",
			command:  "gh pr merge 456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGh := command.NewMockGhRunner(ctrl)
			rule := NewPRMergeRule(mockGh)

			jsonInput := `{"tool_name": "` + tt.toolName + `", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.True(t, got.Allowed)
		})
	}
}

func TestPRMergeRule_Evaluate_NonMergeCommands(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "allow git command",
			command: "git status",
		},
		{
			name:    "allow gh pr create",
			command: "gh pr create --title 'test'",
		},
		{
			name:    "allow gh pr list",
			command: "gh pr list",
		},
		{
			name:    "allow gh pr view",
			command: "gh pr view 123",
		},
		{
			name:    "allow gh issue list",
			command: "gh issue list",
		},
		{
			name:    "allow gh api GET to merge endpoint",
			command: "gh api repos/owner/repo/pulls/123/merge",
		},
		{
			name:    "allow gh api POST to other endpoint",
			command: "gh api -X POST repos/owner/repo/issues",
		},
		{
			name:    "allow empty command",
			command: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGh := command.NewMockGhRunner(ctrl)
			rule := NewPRMergeRule(mockGh)

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

func TestPRMergeRule_Evaluate_AllowMergeToFeatureBranch(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		prNumber   string
		baseBranch string
	}{
		{
			name:       "allow gh pr merge to feature branch",
			command:    "gh pr merge 123",
			prNumber:   "123",
			baseBranch: "feature/test",
		},
		{
			name:       "allow gh pr merge to develop branch",
			command:    "gh pr merge 456",
			prNumber:   "456",
			baseBranch: "develop",
		},
		{
			name:       "allow gh pr merge with --squash to feature branch",
			command:    "gh pr merge 789 --squash",
			prNumber:   "789",
			baseBranch: "feature/new-feature",
		},
		{
			name:       "allow gh pr merge with --merge to feature branch",
			command:    "gh pr merge 101 --merge",
			prNumber:   "101",
			baseBranch: "staging",
		},
		{
			name:       "allow gh pr merge with --rebase to feature branch",
			command:    "gh pr merge 102 --rebase",
			prNumber:   "102",
			baseBranch: "release/v1.0",
		},
		{
			name:       "allow gh api PUT to merge feature branch",
			command:    "gh api -X PUT repos/owner/repo/pulls/123/merge",
			prNumber:   "123",
			baseBranch: "feature/test",
		},
		{
			name:       "allow gh api --method PUT to merge feature branch",
			command:    "gh api --method PUT repos/owner/repo/pulls/456/merge -f merge_method=squash",
			prNumber:   "456",
			baseBranch: "develop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGh := command.NewMockGhRunner(ctrl)
			mockGh.EXPECT().GetPRBaseBranch(context.Background(), "", tt.prNumber).Return(tt.baseBranch, nil)
			rule := NewPRMergeRule(mockGh)

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

func TestPRMergeRule_Evaluate_BlockMergeToMain(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		prNumber string
	}{
		{
			name:     "block gh pr merge to main",
			command:  "gh pr merge 123",
			prNumber: "123",
		},
		{
			name:     "block gh pr merge with --squash to main",
			command:  "gh pr merge 456 --squash",
			prNumber: "456",
		},
		{
			name:     "block gh pr merge with --merge to main",
			command:  "gh pr merge 789 --merge",
			prNumber: "789",
		},
		{
			name:     "block gh pr merge with --rebase to main",
			command:  "gh pr merge 101 --rebase",
			prNumber: "101",
		},
		{
			name:     "block gh pr merge with flags before PR number",
			command:  "gh pr merge --squash 102",
			prNumber: "102",
		},
		{
			name:     "block gh api PUT to merge main",
			command:  "gh api -X PUT repos/owner/repo/pulls/123/merge",
			prNumber: "123",
		},
		{
			name:     "block gh api --method PUT to merge main",
			command:  "gh api --method PUT repos/owner/repo/pulls/456/merge -f merge_method=squash",
			prNumber: "456",
		},
		{
			name:     "block gh api with flag after endpoint",
			command:  "gh api repos/owner/repo/pulls/789/merge -X PUT",
			prNumber: "789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGh := command.NewMockGhRunner(ctrl)
			mockGh.EXPECT().GetPRBaseBranch(context.Background(), "", tt.prNumber).Return("main", nil)
			rule := NewPRMergeRule(mockGh)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.False(t, got.Allowed)
			assert.Equal(t, "gh-pr-merge", got.RuleName)
			assert.Equal(t, "Merging PR to main/master branch is not allowed", got.Message)
		})
	}
}

func TestPRMergeRule_Evaluate_BlockMergeToMaster(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		prNumber string
	}{
		{
			name:     "block gh pr merge to master",
			command:  "gh pr merge 123",
			prNumber: "123",
		},
		{
			name:     "block gh pr merge with --squash to master",
			command:  "gh pr merge 456 --squash",
			prNumber: "456",
		},
		{
			name:     "block gh api PUT to merge master",
			command:  "gh api -X PUT repos/owner/repo/pulls/789/merge",
			prNumber: "789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGh := command.NewMockGhRunner(ctrl)
			mockGh.EXPECT().GetPRBaseBranch(context.Background(), "", tt.prNumber).Return("master", nil)
			rule := NewPRMergeRule(mockGh)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.False(t, got.Allowed)
			assert.Equal(t, "gh-pr-merge", got.RuleName)
			assert.Equal(t, "Merging PR to main/master branch is not allowed", got.Message)
		})
	}
}

func TestPRMergeRule_Evaluate_PRNumberFromURL(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		prNumber string
	}{
		{
			name:     "extract PR number from full GitHub URL",
			command:  "gh pr merge https://github.com/owner/repo/pull/123",
			prNumber: "123",
		},
		{
			name:     "extract PR number from GitHub URL with flags",
			command:  "gh pr merge https://github.com/owner/repo/pull/456 --squash",
			prNumber: "456",
		},
		{
			name:     "extract PR number from GitHub URL with --merge",
			command:  "gh pr merge https://github.com/owner/repo/pull/789 --merge",
			prNumber: "789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGh := command.NewMockGhRunner(ctrl)
			mockGh.EXPECT().GetPRBaseBranch(context.Background(), "", tt.prNumber).Return("main", nil)
			rule := NewPRMergeRule(mockGh)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.False(t, got.Allowed)
		})
	}
}

func TestPRMergeRule_Evaluate_FailOpenOnError(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		prNumber string
		ghError  error
	}{
		{
			name:     "allow on gh error",
			command:  "gh pr merge 123",
			prNumber: "123",
			ghError:  errors.New("gh command failed"),
		},
		{
			name:     "allow on network error",
			command:  "gh pr merge 456",
			prNumber: "456",
			ghError:  errors.New("network error"),
		},
		{
			name:     "allow on gh api error",
			command:  "gh api -X PUT repos/owner/repo/pulls/789/merge",
			prNumber: "789",
			ghError:  errors.New("api error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGh := command.NewMockGhRunner(ctrl)
			mockGh.EXPECT().GetPRBaseBranch(context.Background(), "", tt.prNumber).Return("", tt.ghError)
			rule := NewPRMergeRule(mockGh)

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

func TestPRMergeRule_Evaluate_FailOpenOnInvalidPRNumber(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "allow when PR number is not numeric",
			command: "gh pr merge abc",
		},
		{
			name:    "allow when no PR number provided",
			command: "gh pr merge",
		},
		{
			name:    "allow when only flags provided",
			command: "gh pr merge --squash",
		},
		{
			name:    "allow when invalid URL format",
			command: "gh pr merge https://github.com/owner/repo/issues/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGh := command.NewMockGhRunner(ctrl)
			rule := NewPRMergeRule(mockGh)

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

func TestPRMergeRule_Evaluate_NoCommandArg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGh := command.NewMockGhRunner(ctrl)
	rule := NewPRMergeRule(mockGh)

	jsonInput := `{"tool_name": "Bash", "tool_input": {}}`
	reader := strings.NewReader(jsonInput)
	toolInput, err := ParseToolInput(reader)
	require.NoError(t, err)

	got, err := rule.Evaluate(toolInput)
	require.NoError(t, err)
	assert.True(t, got.Allowed)
	// No expectations set, so gomock will verify no calls were made
}

func TestPRMergeRule_Evaluate_ApiMergeVariants(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		prNumber    string
		baseBranch  string
		wantBlocked bool
	}{
		{
			name:        "block api merge with -X PUT to main",
			command:     "gh api -X PUT repos/owner/repo/pulls/123/merge",
			prNumber:    "123",
			baseBranch:  "main",
			wantBlocked: true,
		},
		{
			name:        "block api merge with --method PUT to main",
			command:     "gh api --method PUT repos/owner/repo/pulls/456/merge",
			prNumber:    "456",
			baseBranch:  "main",
			wantBlocked: true,
		},
		{
			name:        "block api merge with lowercase put to main",
			command:     "gh api -X put repos/owner/repo/pulls/789/merge",
			prNumber:    "789",
			baseBranch:  "main",
			wantBlocked: true,
		},
		{
			name:        "allow api merge with -X PUT to feature branch",
			command:     "gh api -X PUT repos/owner/repo/pulls/111/merge",
			prNumber:    "111",
			baseBranch:  "feature/test",
			wantBlocked: false,
		},
		{
			name:        "allow api merge with different repo owner",
			command:     "gh api -X PUT repos/different/other/pulls/222/merge",
			prNumber:    "222",
			baseBranch:  "develop",
			wantBlocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGh := command.NewMockGhRunner(ctrl)
			mockGh.EXPECT().GetPRBaseBranch(context.Background(), "", tt.prNumber).Return(tt.baseBranch, nil)
			rule := NewPRMergeRule(mockGh)

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)

			if tt.wantBlocked {
				assert.False(t, got.Allowed)
				assert.Equal(t, "gh-pr-merge", got.RuleName)
				assert.Equal(t, "Merging PR to main/master branch is not allowed", got.Message)
			} else {
				assert.True(t, got.Allowed)
			}
		})
	}
}
