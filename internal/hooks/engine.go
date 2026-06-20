package hooks

import "fmt"

// ruleEngine implements the rule evaluation engine.
type ruleEngine struct {
	rules []Rule
}

// NewRuleEngine creates a new rule engine with the given rules.
func NewRuleEngine(rules ...Rule) *ruleEngine {
	return &ruleEngine{
		rules: rules,
	}
}

// Evaluate evaluates all rules against the tool input.
// Returns the first blocking result, or an allowed result if no rules block.
func (e *ruleEngine) Evaluate(input *ToolInput) (*RuleResult, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	for _, rule := range e.rules {
		result, err := rule.Evaluate(input)
		if err != nil {
			return nil, fmt.Errorf("rule %s failed: %w", rule.Name(), err)
		}

		if !result.Allowed {
			return result, nil
		}
	}

	return NewAllowedResult(), nil
}
