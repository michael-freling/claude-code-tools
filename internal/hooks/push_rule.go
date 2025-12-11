package hooks

import (
	"strings"
)

const gitCommandArgsStartIndex = 2 // Skip "git" and subcommand

// gitPushRule blocks git push commands to main/master branches.
type gitPushRule struct {
	gitHelper GitHelper
}

// NewGitPushRule creates a new rule that blocks pushes to main/master branches.
func NewGitPushRule(gitHelper GitHelper) Rule {
	return &gitPushRule{
		gitHelper: gitHelper,
	}
}

// Name returns the unique identifier for this rule.
func (r *gitPushRule) Name() string {
	return "git-push"
}

// Description returns a human-readable description of what this rule does.
func (r *gitPushRule) Description() string {
	return "Blocks git push commands to main/master branches"
}

// Evaluate checks if the Bash command is a git push to main/master.
func (r *gitPushRule) Evaluate(input *ToolInput) (*RuleResult, error) {
	if input.ToolName != "Bash" {
		return NewAllowedResult(), nil
	}

	command, ok := input.GetStringArg("command")
	if !ok {
		return NewAllowedResult(), nil
	}

	command = strings.TrimSpace(command)

	// Parse the command to check if it's a git push
	args := parseGitPushArgs(command)
	if len(args) < 2 || args[0] != "git" || args[1] != "push" {
		return NewAllowedResult(), nil
	}

	// Check for explicit branch name
	if isExplicitPushToProtectedBranch(command) {
		return NewBlockedResult(
			r.Name(),
			"Direct push to main/master branch is not allowed",
		), nil
	}

	// Check for implicit push (no branch specified)
	if isImplicitPush(command) {
		// Get current branch
		currentBranch, err := r.gitHelper.GetCurrentBranch()
		if err != nil {
			// Fail open - allow the command if we can't determine the branch
			return NewAllowedResult(), nil
		}

		if isProtectedBranch(currentBranch) {
			return NewBlockedResult(
				r.Name(),
				"Direct push to main/master branch is not allowed",
			), nil
		}
	}

	return NewAllowedResult(), nil
}

// isExplicitPushToProtectedBranch checks if the command explicitly pushes to main/master.
func isExplicitPushToProtectedBranch(command string) bool {
	args := parseGitPushArgs(command)

	flagsWithValues := []string{"--repo", "--exec", "--receive-pack"}
	nonFlagArgs := findNonFlagArgs(args, gitCommandArgsStartIndex, flagsWithValues)

	if len(nonFlagArgs) == 0 {
		return false
	}

	lastNonFlagArg := nonFlagArgs[len(nonFlagArgs)-1]
	return isProtectedBranch(lastNonFlagArg)
}

// isImplicitPush checks if the command is a git push without a branch specified.
func isImplicitPush(command string) bool {
	args := parseGitPushArgs(command)

	flagsWithValues := []string{"--repo", "--exec", "--receive-pack"}
	nonFlagArgs := findNonFlagArgs(args, gitCommandArgsStartIndex, flagsWithValues)

	// If we have 0 or 1 non-flag args (no args, or just remote), it's implicit
	// If we have 2+ non-flag args (remote and branch), it's explicit
	return len(nonFlagArgs) < 2
}

// parseGitPushArgs parses a git push command into arguments.
// This is a simple parser that handles basic quoting and strips quotes.
func parseGitPushArgs(command string) []string {
	return parseTokensStripQuotes(command)
}
