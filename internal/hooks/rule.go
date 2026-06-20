package hooks

// Rule represents a rule that evaluates whether a tool usage should be allowed.
type Rule interface {
	// Name returns the unique identifier for this rule.
	Name() string

	// Description returns a human-readable description of what this rule does.
	Description() string

	// Evaluate checks if the tool input should be allowed.
	// Returns a RuleResult indicating whether to allow or block the tool usage.
	Evaluate(input *ToolInput) (*RuleResult, error)
}
