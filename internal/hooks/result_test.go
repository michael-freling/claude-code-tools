package hooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAllowedResult(t *testing.T) {
	tests := []struct {
		name string
		want *RuleResult
	}{
		{
			name: "creates allowed result",
			want: &RuleResult{
				Allowed:  true,
				Message:  "",
				RuleName: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewAllowedResult()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewBlockedResult(t *testing.T) {
	tests := []struct {
		name     string
		ruleName string
		message  string
		want     *RuleResult
	}{
		{
			name:     "creates blocked result with message",
			ruleName: "test-rule",
			message:  "test blocked message",
			want: &RuleResult{
				Allowed:  false,
				Message:  "test blocked message",
				RuleName: "test-rule",
			},
		},
		{
			name:     "creates blocked result with empty message",
			ruleName: "test-rule",
			message:  "",
			want: &RuleResult{
				Allowed:  false,
				Message:  "",
				RuleName: "test-rule",
			},
		},
		{
			name:     "creates blocked result with empty rule name",
			ruleName: "",
			message:  "test message",
			want: &RuleResult{
				Allowed:  false,
				Message:  "test message",
				RuleName: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewBlockedResult(tt.ruleName, tt.message)
			assert.Equal(t, tt.want, got)
		})
	}
}
