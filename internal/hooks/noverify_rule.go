package hooks

// noVerifyRule blocks Bash commands containing the --no-verify flag.
type noVerifyRule struct{}

// NewNoVerifyRule creates a new rule that blocks commands with --no-verify.
func NewNoVerifyRule() Rule {
	return &noVerifyRule{}
}

// Name returns the unique identifier for this rule.
func (r *noVerifyRule) Name() string {
	return "no-verify"
}

// Description returns a human-readable description of what this rule does.
func (r *noVerifyRule) Description() string {
	return "Blocks Bash commands containing the --no-verify flag"
}

// Evaluate checks if the Bash command contains --no-verify flag.
func (r *noVerifyRule) Evaluate(input *ToolInput) (*RuleResult, error) {
	if input.ToolName != "Bash" {
		return NewAllowedResult(), nil
	}

	command, ok := input.GetStringArg("command")
	if !ok {
		return NewAllowedResult(), nil
	}

	if containsNoVerifyFlag(command) {
		return NewBlockedResult(
			r.Name(),
			"Command contains --no-verify flag which bypasses git hooks",
		), nil
	}

	return NewAllowedResult(), nil
}

// containsNoVerifyFlag checks if a command contains the --no-verify flag.
// It performs basic parsing to avoid false positives in string literals.
func containsNoVerifyFlag(command string) bool {
	tokens := parseCommandTokens(command)
	for _, token := range tokens {
		if token == "--no-verify" {
			return true
		}
	}
	return false
}
