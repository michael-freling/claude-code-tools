package hooks

import (
	"context"
	"regexp"
	"strings"

	"github.com/michael-freling/claude-code-tools/internal/command"
)

var (
	prURLPattern    = regexp.MustCompile(`/pull/(\d+)`)
	prNumberPattern = regexp.MustCompile(`^\d+$`)
	apiMergePattern = regexp.MustCompile(`repos/[^/]+/[^/]+/pulls/(\d+)/merge`)
)

// prMergeRule blocks PR merge commands to main/master branches.
type prMergeRule struct {
	ghRunner command.GhRunner
}

// NewPRMergeRule creates a new rule that blocks PR merges to main/master branches.
func NewPRMergeRule(ghRunner command.GhRunner) Rule {
	return &prMergeRule{
		ghRunner: ghRunner,
	}
}

// Name returns the unique identifier for this rule.
func (r *prMergeRule) Name() string {
	return "gh-pr-merge"
}

// Description returns a human-readable description of what this rule does.
func (r *prMergeRule) Description() string {
	return "Blocks PR merge commands to main/master branches"
}

// Evaluate checks if the Bash command is a PR merge to main/master.
func (r *prMergeRule) Evaluate(input *ToolInput) (*RuleResult, error) {
	if input.ToolName != "Bash" {
		return NewAllowedResult(), nil
	}

	command, ok := input.GetStringArg("command")
	if !ok {
		return NewAllowedResult(), nil
	}

	command = strings.TrimSpace(command)

	prNumber := extractPRNumber(command)
	if prNumber == "" {
		return NewAllowedResult(), nil
	}

	baseBranch, err := r.ghRunner.GetPRBaseBranch(context.Background(), "", prNumber)
	if err != nil {
		// Fail open - allow the command if we can't determine the base branch
		return NewAllowedResult(), nil
	}

	if isProtectedBranch(baseBranch) {
		return NewBlockedResult(
			r.Name(),
			"Merging PR to main/master branch is not allowed",
		), nil
	}

	return NewAllowedResult(), nil
}

// extractPRNumber extracts the PR number from gh pr merge or gh api merge commands.
// Returns empty string if no PR number is found.
func extractPRNumber(command string) string {
	// Pattern 1: gh pr merge <number or URL>
	prNumber := extractPRNumberFromPRMerge(command)
	if prNumber != "" {
		return prNumber
	}

	// Pattern 2: gh api PUT repos/.../pulls/<number>/merge
	prNumber = extractPRNumberFromApiMerge(command)
	if prNumber != "" {
		return prNumber
	}

	return ""
}

// extractPRNumberFromPRMerge extracts PR number from gh pr merge commands.
func extractPRNumberFromPRMerge(command string) string {
	tokens := strings.Fields(command)
	if len(tokens) < 3 {
		return ""
	}

	if tokens[0] != "gh" || tokens[1] != "pr" || tokens[2] != "merge" {
		return ""
	}

	// The PR number/URL should be the first non-flag argument after "gh pr merge"
	for i := 3; i < len(tokens); i++ {
		token := tokens[i]

		// Skip flags
		if strings.HasPrefix(token, "-") {
			continue
		}

		// Check if it's a GitHub URL
		if strings.Contains(token, "github.com") {
			// Extract PR number from URL like https://github.com/owner/repo/pull/123
			matches := prURLPattern.FindStringSubmatch(token)
			if len(matches) > 1 {
				return matches[1]
			}
		} else {
			// Assume it's a PR number
			// Validate it's numeric
			if prNumberPattern.MatchString(token) {
				return token
			}
		}

		// Only check the first non-flag argument
		break
	}

	return ""
}

// extractPRNumberFromApiMerge extracts PR number from gh api PUT merge commands.
func extractPRNumberFromApiMerge(command string) string {
	tokens := strings.Fields(command)
	if len(tokens) < 2 {
		return ""
	}

	if tokens[0] != "gh" || tokens[1] != "api" {
		return ""
	}

	// Check if this is a PUT request to merge endpoint
	method := extractHTTPMethod(command)
	if method != "PUT" {
		return ""
	}

	// Look for repos/.../pulls/<number>/merge pattern
	matches := apiMergePattern.FindStringSubmatch(command)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}
