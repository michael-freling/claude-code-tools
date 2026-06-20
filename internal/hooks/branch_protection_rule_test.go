package hooks

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBranchProtectionRule(t *testing.T) {
	rule := NewBranchProtectionRule()
	assert.NotNil(t, rule)
	assert.Equal(t, "gh-branch-protection", rule.Name())
	assert.Equal(t, "Blocks gh api commands that modify branch protection settings", rule.Description())
}

func TestBranchProtectionRule_Evaluate_NonBashTool(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		command  string
	}{
		{
			name:     "allow Write tool",
			toolName: "Write",
			command:  "gh api -X DELETE /repos/owner/repo/branches/main/protection",
		},
		{
			name:     "allow Read tool",
			toolName: "Read",
			command:  "gh api -X PUT /repos/owner/repo/branches/main/protection",
		},
		{
			name:     "allow Edit tool",
			toolName: "Edit",
			command:  "gh api -X PATCH /repos/owner/repo/branches/main/protection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewBranchProtectionRule()

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

func TestBranchProtectionRule_Evaluate_NonGhApiCommands(t *testing.T) {
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
			name:    "allow gh issue list",
			command: "gh issue list",
		},
		{
			name:    "allow other command",
			command: "echo 'gh api -X DELETE /repos/owner/repo/branches/main/protection'",
		},
		{
			name:    "allow empty command",
			command: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewBranchProtectionRule()

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

func TestBranchProtectionRule_Evaluate_NonProtectionEndpoints(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "allow gh api to repos endpoint",
			command: "gh api /repos/owner/repo",
		},
		{
			name:    "allow gh api to branches endpoint",
			command: "gh api /repos/owner/repo/branches",
		},
		{
			name:    "allow gh api to specific branch endpoint",
			command: "gh api /repos/owner/repo/branches/main",
		},
		{
			name:    "allow gh api DELETE to other endpoint",
			command: "gh api -X DELETE /repos/owner/repo/hooks/123",
		},
		{
			name:    "allow gh api PUT to other endpoint",
			command: "gh api -X PUT /repos/owner/repo/settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewBranchProtectionRule()

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

func TestBranchProtectionRule_Evaluate_AllowGetRequests(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "allow GET to branch protection endpoint",
			command: "gh api /repos/owner/repo/branches/main/protection",
		},
		{
			name:    "allow explicit GET method",
			command: "gh api -X GET /repos/owner/repo/branches/main/protection",
		},
		{
			name:    "allow --method GET",
			command: "gh api --method GET /repos/owner/repo/branches/main/protection",
		},
		{
			name:    "allow GET to protection status checks",
			command: "gh api /repos/owner/repo/branches/main/protection/required_status_checks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewBranchProtectionRule()

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

func TestBranchProtectionRule_Evaluate_BlockDeleteRequests(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "block DELETE to branch protection endpoint",
			command: "gh api -X DELETE /repos/owner/repo/branches/main/protection",
		},
		{
			name:    "block DELETE with --method flag",
			command: "gh api --method DELETE /repos/owner/repo/branches/main/protection",
		},
		{
			name:    "block DELETE to protection subpath",
			command: "gh api -X DELETE /repos/owner/repo/branches/main/protection/required_status_checks",
		},
		{
			name:    "block DELETE with flag after endpoint",
			command: "gh api /repos/owner/repo/branches/main/protection -X DELETE",
		},
		{
			name:    "block DELETE to master branch protection",
			command: "gh api -X DELETE /repos/owner/repo/branches/master/protection",
		},
		{
			name:    "block DELETE to feature branch protection",
			command: "gh api -X DELETE /repos/owner/repo/branches/feature/test/protection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewBranchProtectionRule()

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.False(t, got.Allowed)
			assert.Equal(t, "gh-branch-protection", got.RuleName)
			assert.Equal(t, "Modifying branch protection settings via gh api is not allowed", got.Message)
		})
	}
}

func TestBranchProtectionRule_Evaluate_BlockPutRequests(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "block PUT to branch protection endpoint",
			command: "gh api -X PUT /repos/owner/repo/branches/main/protection -f data='test'",
		},
		{
			name:    "block PUT with --method flag",
			command: "gh api --method PUT /repos/owner/repo/branches/main/protection -f required_status_checks=null",
		},
		{
			name:    "block PUT to protection subpath",
			command: "gh api -X PUT /repos/owner/repo/branches/main/protection/required_status_checks -f data='test'",
		},
		{
			name:    "block PUT with flag after endpoint",
			command: "gh api /repos/owner/repo/branches/main/protection -X PUT -f data='test'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewBranchProtectionRule()

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.False(t, got.Allowed)
			assert.Equal(t, "gh-branch-protection", got.RuleName)
			assert.Equal(t, "Modifying branch protection settings via gh api is not allowed", got.Message)
		})
	}
}

func TestBranchProtectionRule_Evaluate_BlockPatchRequests(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "block PATCH to branch protection endpoint",
			command: "gh api -X PATCH /repos/owner/repo/branches/main/protection -f data='test'",
		},
		{
			name:    "block PATCH with --method flag",
			command: "gh api --method PATCH /repos/owner/repo/branches/main/protection -f required_status_checks=null",
		},
		{
			name:    "block PATCH to protection subpath",
			command: "gh api -X PATCH /repos/owner/repo/branches/main/protection/required_status_checks -f data='test'",
		},
		{
			name:    "block PATCH with flag after endpoint",
			command: "gh api /repos/owner/repo/branches/main/protection -X PATCH -f data='test'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewBranchProtectionRule()

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.False(t, got.Allowed)
			assert.Equal(t, "gh-branch-protection", got.RuleName)
			assert.Equal(t, "Modifying branch protection settings via gh api is not allowed", got.Message)
		})
	}
}

func TestBranchProtectionRule_Evaluate_MethodCaseInsensitive(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "block lowercase delete",
			command: "gh api -X delete /repos/owner/repo/branches/main/protection",
		},
		{
			name:    "block lowercase put",
			command: "gh api -X put /repos/owner/repo/branches/main/protection -f data='test'",
		},
		{
			name:    "block lowercase patch",
			command: "gh api -X patch /repos/owner/repo/branches/main/protection -f data='test'",
		},
		{
			name:    "block mixed case DELETE",
			command: "gh api -X Delete /repos/owner/repo/branches/main/protection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewBranchProtectionRule()

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

func TestBranchProtectionRule_Evaluate_NoCommandArg(t *testing.T) {
	rule := NewBranchProtectionRule()

	jsonInput := `{"tool_name": "Bash", "tool_input": {}}`
	reader := strings.NewReader(jsonInput)
	toolInput, err := ParseToolInput(reader)
	require.NoError(t, err)

	got, err := rule.Evaluate(toolInput)
	require.NoError(t, err)
	assert.True(t, got.Allowed)
}
