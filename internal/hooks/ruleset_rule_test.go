package hooks

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRulesetRule(t *testing.T) {
	rule := NewRulesetRule()
	assert.NotNil(t, rule)
	assert.Equal(t, "gh-ruleset", rule.Name())
	assert.Equal(t, "Blocks gh api commands that modify repository rulesets", rule.Description())
}

func TestRulesetRule_Evaluate_NonBashTool(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		command  string
	}{
		{
			name:     "allow Write tool",
			toolName: "Write",
			command:  "gh api -X DELETE /repos/owner/repo/rulesets/123",
		},
		{
			name:     "allow Read tool",
			toolName: "Read",
			command:  "gh api -X PUT /orgs/myorg/rulesets/456",
		},
		{
			name:     "allow Edit tool",
			toolName: "Edit",
			command:  "gh api -X PATCH /repos/owner/repo/rulesets/789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewRulesetRule()

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

func TestRulesetRule_Evaluate_NonGhApiCommands(t *testing.T) {
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
			command: "echo 'gh api -X DELETE /repos/owner/repo/rulesets/123'",
		},
		{
			name:    "allow empty command",
			command: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewRulesetRule()

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

func TestRulesetRule_Evaluate_NonRulesetEndpoints(t *testing.T) {
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
			name:    "allow gh api DELETE to other endpoint",
			command: "gh api -X DELETE /repos/owner/repo/hooks/123",
		},
		{
			name:    "allow gh api PUT to other endpoint",
			command: "gh api -X PUT /repos/owner/repo/settings",
		},
		{
			name:    "allow gh api to branch protection",
			command: "gh api -X DELETE /repos/owner/repo/branches/main/protection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewRulesetRule()

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

func TestRulesetRule_Evaluate_AllowGetRequests(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "allow GET to repo rulesets endpoint",
			command: "gh api /repos/owner/repo/rulesets",
		},
		{
			name:    "allow GET to specific repo ruleset",
			command: "gh api /repos/owner/repo/rulesets/123",
		},
		{
			name:    "allow explicit GET method for repo",
			command: "gh api -X GET /repos/owner/repo/rulesets/123",
		},
		{
			name:    "allow --method GET for repo",
			command: "gh api --method GET /repos/owner/repo/rulesets/123",
		},
		{
			name:    "allow GET to org rulesets endpoint",
			command: "gh api /orgs/myorg/rulesets",
		},
		{
			name:    "allow GET to specific org ruleset",
			command: "gh api /orgs/myorg/rulesets/456",
		},
		{
			name:    "allow explicit GET method for org",
			command: "gh api -X GET /orgs/myorg/rulesets/456",
		},
		{
			name:    "allow --method GET for org",
			command: "gh api --method GET /orgs/myorg/rulesets/456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewRulesetRule()

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

func TestRulesetRule_Evaluate_BlockPostRequests(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "block POST to repo rulesets",
			command: "gh api -X POST /repos/owner/repo/rulesets -f data='test'",
		},
		{
			name:    "block --method POST to repo rulesets",
			command: "gh api --method POST /repos/owner/repo/rulesets -f data='test'",
		},
		{
			name:    "block POST to org rulesets",
			command: "gh api -X POST /orgs/myorg/rulesets -f data='test'",
		},
		{
			name:    "block --method POST to org rulesets",
			command: "gh api --method POST /orgs/myorg/rulesets -f data='test'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewRulesetRule()

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.False(t, got.Allowed)
			assert.Equal(t, "gh-ruleset", got.RuleName)
			assert.Equal(t, "Modifying repository rulesets via gh api is not allowed", got.Message)
		})
	}
}

func TestRulesetRule_Evaluate_BlockDeleteRepoRulesets(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "block DELETE to repo ruleset",
			command: "gh api -X DELETE /repos/owner/repo/rulesets/123",
		},
		{
			name:    "block DELETE with --method flag",
			command: "gh api --method DELETE /repos/owner/repo/rulesets/123",
		},
		{
			name:    "block DELETE with flag after endpoint",
			command: "gh api /repos/owner/repo/rulesets/123 -X DELETE",
		},
		{
			name:    "block DELETE to rulesets endpoint",
			command: "gh api -X DELETE /repos/owner/repo/rulesets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewRulesetRule()

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.False(t, got.Allowed)
			assert.Equal(t, "gh-ruleset", got.RuleName)
			assert.Equal(t, "Modifying repository rulesets via gh api is not allowed", got.Message)
		})
	}
}

func TestRulesetRule_Evaluate_BlockDeleteOrgRulesets(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "block DELETE to org ruleset",
			command: "gh api -X DELETE /orgs/myorg/rulesets/456",
		},
		{
			name:    "block DELETE with --method flag",
			command: "gh api --method DELETE /orgs/myorg/rulesets/456",
		},
		{
			name:    "block DELETE with flag after endpoint",
			command: "gh api /orgs/myorg/rulesets/456 -X DELETE",
		},
		{
			name:    "block DELETE to org rulesets endpoint",
			command: "gh api -X DELETE /orgs/myorg/rulesets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewRulesetRule()

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.False(t, got.Allowed)
			assert.Equal(t, "gh-ruleset", got.RuleName)
			assert.Equal(t, "Modifying repository rulesets via gh api is not allowed", got.Message)
		})
	}
}

func TestRulesetRule_Evaluate_BlockPutRepoRulesets(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "block PUT to repo ruleset",
			command: "gh api -X PUT /repos/owner/repo/rulesets/123 -f data='test'",
		},
		{
			name:    "block PUT with --method flag",
			command: "gh api --method PUT /repos/owner/repo/rulesets/123 -f data='test'",
		},
		{
			name:    "block PUT with flag after endpoint",
			command: "gh api /repos/owner/repo/rulesets/123 -X PUT -f data='test'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewRulesetRule()

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.False(t, got.Allowed)
			assert.Equal(t, "gh-ruleset", got.RuleName)
			assert.Equal(t, "Modifying repository rulesets via gh api is not allowed", got.Message)
		})
	}
}

func TestRulesetRule_Evaluate_BlockPutOrgRulesets(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "block PUT to org ruleset",
			command: "gh api -X PUT /orgs/myorg/rulesets/456 -f data='test'",
		},
		{
			name:    "block PUT with --method flag",
			command: "gh api --method PUT /orgs/myorg/rulesets/456 -f data='test'",
		},
		{
			name:    "block PUT with flag after endpoint",
			command: "gh api /orgs/myorg/rulesets/456 -X PUT -f data='test'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewRulesetRule()

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.False(t, got.Allowed)
			assert.Equal(t, "gh-ruleset", got.RuleName)
			assert.Equal(t, "Modifying repository rulesets via gh api is not allowed", got.Message)
		})
	}
}

func TestRulesetRule_Evaluate_BlockPatchRepoRulesets(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "block PATCH to repo ruleset",
			command: "gh api -X PATCH /repos/owner/repo/rulesets/123 -f data='test'",
		},
		{
			name:    "block PATCH with --method flag",
			command: "gh api --method PATCH /repos/owner/repo/rulesets/123 -f data='test'",
		},
		{
			name:    "block PATCH with flag after endpoint",
			command: "gh api /repos/owner/repo/rulesets/123 -X PATCH -f data='test'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewRulesetRule()

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.False(t, got.Allowed)
			assert.Equal(t, "gh-ruleset", got.RuleName)
			assert.Equal(t, "Modifying repository rulesets via gh api is not allowed", got.Message)
		})
	}
}

func TestRulesetRule_Evaluate_BlockPatchOrgRulesets(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "block PATCH to org ruleset",
			command: "gh api -X PATCH /orgs/myorg/rulesets/456 -f data='test'",
		},
		{
			name:    "block PATCH with --method flag",
			command: "gh api --method PATCH /orgs/myorg/rulesets/456 -f data='test'",
		},
		{
			name:    "block PATCH with flag after endpoint",
			command: "gh api /orgs/myorg/rulesets/456 -X PATCH -f data='test'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewRulesetRule()

			jsonInput := `{"tool_name": "Bash", "tool_input": {"command": "` + escapeJSON(tt.command) + `"}}`
			reader := strings.NewReader(jsonInput)
			toolInput, err := ParseToolInput(reader)
			require.NoError(t, err)

			got, err := rule.Evaluate(toolInput)
			require.NoError(t, err)
			assert.False(t, got.Allowed)
			assert.Equal(t, "gh-ruleset", got.RuleName)
			assert.Equal(t, "Modifying repository rulesets via gh api is not allowed", got.Message)
		})
	}
}

func TestRulesetRule_Evaluate_MethodCaseInsensitive(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "block lowercase delete for repo",
			command: "gh api -X delete /repos/owner/repo/rulesets/123",
		},
		{
			name:    "block lowercase put for repo",
			command: "gh api -X put /repos/owner/repo/rulesets/123 -f data='test'",
		},
		{
			name:    "block lowercase patch for repo",
			command: "gh api -X patch /repos/owner/repo/rulesets/123 -f data='test'",
		},
		{
			name:    "block mixed case DELETE for repo",
			command: "gh api -X Delete /repos/owner/repo/rulesets/123",
		},
		{
			name:    "block lowercase delete for org",
			command: "gh api -X delete /orgs/myorg/rulesets/456",
		},
		{
			name:    "block lowercase put for org",
			command: "gh api -X put /orgs/myorg/rulesets/456 -f data='test'",
		},
		{
			name:    "block lowercase patch for org",
			command: "gh api -X patch /orgs/myorg/rulesets/456 -f data='test'",
		},
		{
			name:    "block mixed case DELETE for org",
			command: "gh api -X Delete /orgs/myorg/rulesets/456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewRulesetRule()

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

func TestRulesetRule_Evaluate_NoCommandArg(t *testing.T) {
	rule := NewRulesetRule()

	jsonInput := `{"tool_name": "Bash", "tool_input": {}}`
	reader := strings.NewReader(jsonInput)
	toolInput, err := ParseToolInput(reader)
	require.NoError(t, err)

	got, err := rule.Evaluate(toolInput)
	require.NoError(t, err)
	assert.True(t, got.Allowed)
}
