package main

import (
	"fmt"
	"os"

	"github.com/michael-freling/claude-code-tools/internal/command"
	"github.com/michael-freling/claude-code-tools/internal/hooks"
	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "claude-hooks",
		Short: "Claude Code hook system for controlling tool usage",
		Long:  `A CLI tool that provides hook execution for Claude Code, allowing control over which tools can be used and under what conditions.`,
	}

	rootCmd.AddCommand(newPreToolUseCmd())

	return rootCmd
}

func newPreToolUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pre-tool-use",
		Short: "Evaluate rules before tool execution",
		Long:  `Reads tool input from stdin as JSON and evaluates configured rules. Returns exit code 0 to allow, exit code 2 to block.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			toolInput, err := hooks.ParseToolInput(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("failed to parse tool input: %w", err)
			}

			runner := command.NewRunner()
			gitRunner := command.NewGitRunner(runner)
			ghRunner := command.NewGhRunner(runner)

			rules := []hooks.Rule{
				hooks.NewNoVerifyRule(),
				hooks.NewGitPushRule(gitRunner),
				hooks.NewBranchProtectionRule(),
				hooks.NewRulesetRule(),
				hooks.NewPRMergeRule(ghRunner),
			}

			engine := hooks.NewRuleEngine(rules...)
			result, err := engine.Evaluate(toolInput)
			if err != nil {
				return fmt.Errorf("failed to evaluate rules: %w", err)
			}

			if !result.Allowed {
				fmt.Fprintf(cmd.ErrOrStderr(), "Blocked by rule %s: %s\n", result.RuleName, result.Message)
				os.Exit(2)
			}

			return nil
		},
	}
}
