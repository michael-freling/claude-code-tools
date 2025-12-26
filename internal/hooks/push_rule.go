package hooks

import (
	"context"
	"strings"

	"github.com/michael-freling/claude-code-tools/internal/command"
)

const gitCommandArgsStartIndex = 2 // Skip "git" and subcommand

// gitPushRule blocks git push commands to main/master branches.
type gitPushRule struct {
	gitRunner command.GitRunner
}

// NewGitPushRule creates a new rule that blocks pushes to main/master branches.
func NewGitPushRule(gitRunner command.GitRunner) Rule {
	return &gitPushRule{
		gitRunner: gitRunner,
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

	// Split command on shell operators to handle chaining, pipes, backgrounding
	subCommands := splitShellCommands(command)

	// Check each sub-command
	for _, subCmd := range subCommands {
		if result := r.evaluateSingleCommand(subCmd); result != nil && !result.Allowed {
			return result, nil
		}
	}

	return NewAllowedResult(), nil
}

// evaluateSingleCommand checks if a single command (not chained) is a blocked git push.
func (r *gitPushRule) evaluateSingleCommand(command string) *RuleResult {
	command = strings.TrimSpace(command)

	// Parse the command to check if it's a git push
	args := parseGitPushArgs(command)
	if len(args) < 2 || args[0] != "git" || args[1] != "push" {
		return nil
	}

	// Check for --all or --mirror flags (pushes to all branches including protected ones)
	if containsPushAllFlag(args) {
		return NewBlockedResult(
			r.Name(),
			"Push --all/--mirror includes protected branches and is not allowed",
		)
	}

	// Check for delete operations on protected branches
	if result := r.checkDeleteOperation(args); result != nil {
		return result
	}

	// Check for refspec-based push to protected branches (including force push with +)
	if result := r.checkRefspecPush(args); result != nil {
		return result
	}

	// Check for explicit branch name
	if isExplicitPushToProtectedBranch(command) {
		return NewBlockedResult(
			r.Name(),
			"Direct push to main/master branch is not allowed",
		)
	}

	// Check for implicit push (no branch specified)
	if isImplicitPush(command) {
		// Get current branch
		currentBranch, err := r.gitRunner.GetCurrentBranch(context.Background(), "")
		if err != nil {
			// Fail open - allow the command if we can't determine the branch
			return nil
		}

		if isProtectedBranch(currentBranch) {
			return NewBlockedResult(
				r.Name(),
				"Direct push to main/master branch is not allowed",
			)
		}
	}

	return nil
}

// checkDeleteOperation checks for delete operations on protected branches.
func (r *gitPushRule) checkDeleteOperation(args []string) *RuleResult {
	flagsWithValues := []string{"--repo", "--exec", "--receive-pack"}
	nonFlagArgs := findNonFlagArgs(args, gitCommandArgsStartIndex, flagsWithValues)

	// Check for --delete or -d flag with protected branch
	if containsDeleteFlag(args) {
		for _, arg := range nonFlagArgs {
			if isProtectedBranch(arg) {
				return NewBlockedResult(
					r.Name(),
					"Deleting main/master branch is not allowed",
				)
			}
		}
	}

	// Check for delete refspec (:main or :master)
	for _, arg := range nonFlagArgs {
		if isDeleteRefspec(arg) {
			target := extractTargetFromRefspec(arg)
			if isProtectedBranch(target) {
				return NewBlockedResult(
					r.Name(),
					"Deleting main/master branch is not allowed",
				)
			}
		}
	}

	return nil
}

// checkRefspecPush checks for refspec-based pushes to protected branches.
func (r *gitPushRule) checkRefspecPush(args []string) *RuleResult {
	flagsWithValues := []string{"--repo", "--exec", "--receive-pack"}
	nonFlagArgs := findNonFlagArgs(args, gitCommandArgsStartIndex, flagsWithValues)

	for _, arg := range nonFlagArgs {
		// Skip delete refspecs (handled separately)
		if isDeleteRefspec(arg) {
			continue
		}

		// Check if this is a refspec (contains : or starts with +)
		if strings.Contains(arg, ":") || isForcePushRefspec(arg) {
			target := extractTargetFromRefspec(arg)
			if isProtectedBranch(target) {
				if isForcePushRefspec(arg) {
					return NewBlockedResult(
						r.Name(),
						"Force push to main/master branch is not allowed",
					)
				}
				return NewBlockedResult(
					r.Name(),
					"Direct push to main/master branch is not allowed",
				)
			}
		}
	}

	return nil
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
