package hooks

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRule is a test implementation of the Rule interface.
type mockRule struct {
	name        string
	description string
	result      *RuleResult
	err         error
	onEvaluate  func()
}

func (m *mockRule) Name() string {
	return m.name
}

func (m *mockRule) Description() string {
	return m.description
}

func (m *mockRule) Evaluate(input *ToolInput) (*RuleResult, error) {
	if m.onEvaluate != nil {
		m.onEvaluate()
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func TestNewRuleEngine(t *testing.T) {
	tests := []struct {
		name  string
		rules []Rule
	}{
		{
			name:  "creates engine with no rules",
			rules: []Rule{},
		},
		{
			name: "creates engine with one rule",
			rules: []Rule{
				&mockRule{name: "rule1"},
			},
		},
		{
			name: "creates engine with multiple rules",
			rules: []Rule{
				&mockRule{name: "rule1"},
				&mockRule{name: "rule2"},
				&mockRule{name: "rule3"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRuleEngine(tt.rules...)
			assert.NotNil(t, got)
			assert.Equal(t, len(tt.rules), len(got.rules))
		})
	}
}

func TestRuleEngine_Evaluate(t *testing.T) {
	tests := []struct {
		name    string
		rules   []Rule
		input   *ToolInput
		want    *RuleResult
		wantErr bool
	}{
		{
			name:  "no rules returns allowed",
			rules: []Rule{},
			input: &ToolInput{ToolName: "Test"},
			want:  NewAllowedResult(),
		},
		{
			name: "all rules allow returns allowed",
			rules: []Rule{
				&mockRule{
					name:   "rule1",
					result: NewAllowedResult(),
				},
				&mockRule{
					name:   "rule2",
					result: NewAllowedResult(),
				},
			},
			input: &ToolInput{ToolName: "Test"},
			want:  NewAllowedResult(),
		},
		{
			name: "first rule blocks returns blocked",
			rules: []Rule{
				&mockRule{
					name:   "rule1",
					result: NewBlockedResult("rule1", "blocked by rule1"),
				},
				&mockRule{
					name:   "rule2",
					result: NewAllowedResult(),
				},
			},
			input: &ToolInput{ToolName: "Test"},
			want:  NewBlockedResult("rule1", "blocked by rule1"),
		},
		{
			name: "second rule blocks returns blocked",
			rules: []Rule{
				&mockRule{
					name:   "rule1",
					result: NewAllowedResult(),
				},
				&mockRule{
					name:   "rule2",
					result: NewBlockedResult("rule2", "blocked by rule2"),
				},
			},
			input: &ToolInput{ToolName: "Test"},
			want:  NewBlockedResult("rule2", "blocked by rule2"),
		},
		{
			name: "returns first blocked rule when multiple block",
			rules: []Rule{
				&mockRule{
					name:   "rule1",
					result: NewBlockedResult("rule1", "blocked by rule1"),
				},
				&mockRule{
					name:   "rule2",
					result: NewBlockedResult("rule2", "blocked by rule2"),
				},
			},
			input: &ToolInput{ToolName: "Test"},
			want:  NewBlockedResult("rule1", "blocked by rule1"),
		},
		{
			name: "rule error returns error",
			rules: []Rule{
				&mockRule{
					name: "rule1",
					err:  fmt.Errorf("rule evaluation failed"),
				},
			},
			input:   &ToolInput{ToolName: "Test"},
			wantErr: true,
		},
		{
			name: "error in second rule returns error",
			rules: []Rule{
				&mockRule{
					name:   "rule1",
					result: NewAllowedResult(),
				},
				&mockRule{
					name: "rule2",
					err:  fmt.Errorf("rule evaluation failed"),
				},
			},
			input:   &ToolInput{ToolName: "Test"},
			wantErr: true,
		},
		{
			name:    "nil input returns error",
			rules:   []Rule{},
			input:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewRuleEngine(tt.rules...)
			got, err := engine.Evaluate(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRuleEngine_Evaluate_ShortCircuit(t *testing.T) {
	evaluationCount := 0

	rule1 := &mockRule{
		name:   "rule1",
		result: NewBlockedResult("rule1", "blocked"),
	}

	rule2 := &mockRule{
		name:   "rule2",
		result: NewAllowedResult(),
		onEvaluate: func() {
			evaluationCount++
		},
	}

	engine := NewRuleEngine(rule1, rule2)
	input := &ToolInput{ToolName: "Test"}

	result, err := engine.Evaluate(input)

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, 0, evaluationCount, "second rule should not be evaluated when first rule blocks")
}
