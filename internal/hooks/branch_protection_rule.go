package hooks

import (
	"regexp"
)

var branchProtectionPattern = regexp.MustCompile(`/?repos/[^/]+/[^/]+/branches/.+/protection`)

// branchProtectionRule blocks gh api commands that modify branch protections.
type branchProtectionRule struct{}

// NewBranchProtectionRule creates a new rule that blocks gh api commands modifying branch protections.
func NewBranchProtectionRule() Rule {
	return &branchProtectionRule{}
}

// Name returns the unique identifier for this rule.
func (r *branchProtectionRule) Name() string {
	return "gh-branch-protection"
}

// Description returns a human-readable description of what this rule does.
func (r *branchProtectionRule) Description() string {
	return "Blocks gh api commands that modify branch protection settings"
}

// Evaluate checks if the Bash command is a gh api call modifying branch protections.
func (r *branchProtectionRule) Evaluate(input *ToolInput) (*RuleResult, error) {
	if input.ToolName != "Bash" {
		return NewAllowedResult(), nil
	}

	command, ok := input.GetStringArg("command")
	if !ok {
		return NewAllowedResult(), nil
	}

	if isModifyingBranchProtection(command) {
		return NewBlockedResult(
			r.Name(),
			"Modifying branch protection settings via gh api is not allowed",
		), nil
	}

	return NewAllowedResult(), nil
}

// isModifyingBranchProtection checks if a command is a gh api call that modifies branch protections.
func isModifyingBranchProtection(command string) bool {
	if !isGhApiCommand(command) {
		return false
	}

	if !hasBranchProtectionEndpoint(command) {
		return false
	}

	method := extractHTTPMethod(command)
	return method == "DELETE" || method == "PUT" || method == "PATCH" || method == "POST"
}

// hasBranchProtectionEndpoint checks if the command targets a branch protection endpoint.
func hasBranchProtectionEndpoint(command string) bool {
	return branchProtectionPattern.MatchString(command)
}
