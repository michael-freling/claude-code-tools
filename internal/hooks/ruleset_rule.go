package hooks

import (
	"regexp"
)

var (
	repoRulesetPattern = regexp.MustCompile(`/repos/[^/]+/[^/]+/rulesets`)
	orgRulesetPattern  = regexp.MustCompile(`/orgs/[^/]+/rulesets`)
)

// rulesetRule blocks gh api commands that modify repository rulesets.
type rulesetRule struct{}

// NewRulesetRule creates a new rule that blocks gh api commands modifying rulesets.
func NewRulesetRule() Rule {
	return &rulesetRule{}
}

// Name returns the unique identifier for this rule.
func (r *rulesetRule) Name() string {
	return "gh-ruleset"
}

// Description returns a human-readable description of what this rule does.
func (r *rulesetRule) Description() string {
	return "Blocks gh api commands that modify repository rulesets"
}

// Evaluate checks if the Bash command is a gh api call modifying rulesets.
func (r *rulesetRule) Evaluate(input *ToolInput) (*RuleResult, error) {
	if input.ToolName != "Bash" {
		return NewAllowedResult(), nil
	}

	command, ok := input.GetStringArg("command")
	if !ok {
		return NewAllowedResult(), nil
	}

	if isModifyingRuleset(command) {
		return NewBlockedResult(
			r.Name(),
			"Modifying repository rulesets via gh api is not allowed",
		), nil
	}

	return NewAllowedResult(), nil
}

// isModifyingRuleset checks if a command is a gh api call that modifies rulesets.
func isModifyingRuleset(command string) bool {
	if !isGhApiCommand(command) {
		return false
	}

	if !hasRulesetEndpoint(command) {
		return false
	}

	method := extractHTTPMethod(command)
	return method == "DELETE" || method == "PUT" || method == "PATCH"
}

// hasRulesetEndpoint checks if the command targets a ruleset endpoint.
func hasRulesetEndpoint(command string) bool {
	return repoRulesetPattern.MatchString(command) || orgRulesetPattern.MatchString(command)
}
