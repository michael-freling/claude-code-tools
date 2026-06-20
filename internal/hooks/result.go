package hooks

// RuleResult represents the result of evaluating a rule.
type RuleResult struct {
	// Allowed indicates whether the tool usage should be allowed.
	Allowed bool

	// Message provides additional context about the decision.
	// For blocked results, this explains why the tool was blocked.
	Message string

	// RuleName identifies which rule produced this result.
	RuleName string
}

// NewAllowedResult creates a result that allows the tool usage.
func NewAllowedResult() *RuleResult {
	return &RuleResult{
		Allowed:  true,
		Message:  "",
		RuleName: "",
	}
}

// NewBlockedResult creates a result that blocks the tool usage.
func NewBlockedResult(ruleName, message string) *RuleResult {
	return &RuleResult{
		Allowed:  false,
		Message:  message,
		RuleName: ruleName,
	}
}
