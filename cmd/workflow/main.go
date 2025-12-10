package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/michael-freling/claude-code-tools/internal/workflow"
	"github.com/spf13/cobra"
)

var (
	baseDir                    string
	splitPR                    bool
	claudePath                 string
	dangerouslySkipPermissions bool
	timeoutPlanning            time.Duration
	timeoutImplement           time.Duration
	timeoutRefactoring         time.Duration
	timeoutPRSplit             time.Duration
	verbose                    bool
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "claude-workflow",
		Short: "Orchestrate multi-phase development workflows with Claude Code CLI",
		Long:  `A CLI tool that manages multi-phase development workflows by invoking Claude Code CLI externally with persistent state.`,
	}

	rootCmd.PersistentFlags().StringVar(&baseDir, "base-dir", ".claude/workflow", "base directory for workflows")
	rootCmd.PersistentFlags().BoolVar(&splitPR, "split-pr", false, "enable PR split phase to split large PRs into smaller child PRs")
	rootCmd.PersistentFlags().StringVar(&claudePath, "claude-path", "claude", "path to claude CLI")
	rootCmd.PersistentFlags().BoolVar(&dangerouslySkipPermissions, "dangerously-skip-permissions", false, "skip all permission prompts in Claude Code (use with caution)")
	rootCmd.PersistentFlags().DurationVar(&timeoutPlanning, "timeout-planning", 1*time.Hour, "planning phase timeout")
	rootCmd.PersistentFlags().DurationVar(&timeoutImplement, "timeout-implementation", 6*time.Hour, "implementation phase timeout")
	rootCmd.PersistentFlags().DurationVar(&timeoutRefactoring, "timeout-refactoring", 6*time.Hour, "refactoring phase timeout")
	rootCmd.PersistentFlags().DurationVar(&timeoutPRSplit, "timeout-pr-split", 1*time.Hour, "PR split phase timeout")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output showing internal operations")

	rootCmd.AddCommand(newStartCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newResumeCmd())
	rootCmd.AddCommand(newDeleteCmd())
	rootCmd.AddCommand(newCleanCmd())

	return rootCmd
}

func createOrchestrator() (*workflow.Orchestrator, error) {
	config := workflow.DefaultConfig(baseDir)
	config.SplitPR = splitPR
	config.ClaudePath = claudePath
	config.DangerouslySkipPermissions = dangerouslySkipPermissions
	config.Timeouts = workflow.PhaseTimeouts{
		Planning:       timeoutPlanning,
		Implementation: timeoutImplement,
		Refactoring:    timeoutRefactoring,
		PRSplit:        timeoutPRSplit,
	}
	if verbose {
		config.LogLevel = workflow.LogLevelVerbose
	}
	return workflow.NewOrchestratorWithConfig(config)
}

func newStartCmd() *cobra.Command {
	var workflowType string

	cmd := &cobra.Command{
		Use:   "start <name> <description>",
		Short: "Start a new workflow",
		Long:  `Start a new workflow with the given name and description.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			description := args[1]

			var wfType workflow.WorkflowType
			switch workflowType {
			case "feature":
				wfType = workflow.WorkflowTypeFeature
			case "fix":
				wfType = workflow.WorkflowTypeFix
			default:
				return fmt.Errorf("invalid workflow type: %s (must be 'feature' or 'fix')", workflowType)
			}

			orchestrator, err := createOrchestrator()
			if err != nil {
				return fmt.Errorf("failed to create orchestrator: %w", err)
			}

			ctx := context.Background()
			if err := orchestrator.Start(ctx, name, description, wfType); err != nil {
				fmt.Printf("\n%s %s\n", workflow.Red("✗"), err.Error())
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&workflowType, "type", "", "workflow type (feature or fix)")
	cmd.MarkFlagRequired("type")

	return cmd
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all workflows",
		Long:  `List all workflows with their current status.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			orchestrator, err := createOrchestrator()
			if err != nil {
				return fmt.Errorf("failed to create orchestrator: %w", err)
			}

			workflows, err := orchestrator.List()
			if err != nil {
				return fmt.Errorf("failed to list workflows: %w", err)
			}

			if len(workflows) == 0 {
				fmt.Println(workflow.Yellow("No workflows found."))
				return nil
			}

			fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\n",
				workflow.Bold("NAME"),
				workflow.Bold("TYPE"),
				workflow.Bold("PHASE"),
				workflow.Bold("STATUS"),
				workflow.Bold("CREATED"),
				workflow.Bold("UPDATED"),
			)
			for _, wf := range workflows {
				statusStr := wf.Status
				switch wf.Status {
				case "completed":
					statusStr = workflow.Green(wf.Status)
				case "failed":
					statusStr = workflow.Red(wf.Status)
				default:
					statusStr = workflow.Yellow(wf.Status)
				}

				fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\n",
					wf.Name,
					wf.Type,
					wf.CurrentPhase,
					statusStr,
					wf.CreatedAt.Format("2006-01-02 15:04"),
					wf.UpdatedAt.Format("2006-01-02 15:04"),
				)
			}

			return nil
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <name>",
		Short: "Show workflow status",
		Long:  `Show detailed status of a specific workflow.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			orchestrator, err := createOrchestrator()
			if err != nil {
				return fmt.Errorf("failed to create orchestrator: %w", err)
			}

			state, err := orchestrator.Status(name)
			if err != nil {
				return fmt.Errorf("failed to get workflow status: %w", err)
			}

			fmt.Println(workflow.FormatWorkflowStatus(state))

			return nil
		},
	}
}

func newResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <name>",
		Short: "Resume an interrupted workflow",
		Long:  `Resume a workflow from its current phase.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			orchestrator, err := createOrchestrator()
			if err != nil {
				return fmt.Errorf("failed to create orchestrator: %w", err)
			}

			ctx := context.Background()
			if err := orchestrator.Resume(ctx, name); err != nil {
				fmt.Printf("\n%s %s\n", workflow.Red("✗"), err.Error())
				return err
			}

			return nil
		},
	}
}

func newDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a workflow",
		Long:  `Delete a workflow and all its state.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			orchestrator, err := createOrchestrator()
			if err != nil {
				return fmt.Errorf("failed to create orchestrator: %w", err)
			}

			if !force {
				fmt.Printf("%s ", workflow.Yellow("Are you sure you want to delete workflow '"+name+"'? (y/n):"))
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "yes" {
					fmt.Println(workflow.Yellow("Deletion cancelled."))
					return nil
				}
			}

			if err := orchestrator.Delete(name); err != nil {
				return fmt.Errorf("failed to delete workflow: %w", err)
			}

			fmt.Printf("%s Workflow '%s' deleted successfully.\n", workflow.Green("✓"), name)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")

	return cmd
}

func newCleanCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Delete all completed workflows",
		Long:  `Delete all workflows that have completed successfully.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			orchestrator, err := createOrchestrator()
			if err != nil {
				return fmt.Errorf("failed to create orchestrator: %w", err)
			}

			workflows, err := orchestrator.List()
			if err != nil {
				return fmt.Errorf("failed to list workflows: %w", err)
			}

			var completedWorkflows []string
			for _, wf := range workflows {
				if wf.Status == "completed" {
					completedWorkflows = append(completedWorkflows, wf.Name)
				}
			}

			if len(completedWorkflows) == 0 {
				fmt.Println(workflow.Yellow("No completed workflows to clean."))
				return nil
			}

			fmt.Printf("%s Found %d completed workflow(s):\n", workflow.Cyan("ℹ"), len(completedWorkflows))
			for _, name := range completedWorkflows {
				fmt.Printf("  %s %s\n", workflow.Green("✓"), name)
			}

			if !force {
				fmt.Print(workflow.Yellow("\nDelete all completed workflows? (y/n): "))
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "yes" {
					fmt.Println(workflow.Yellow("Clean cancelled."))
					return nil
				}
			}

			deleted, err := orchestrator.Clean()
			if err != nil {
				return fmt.Errorf("failed to clean workflows: %w", err)
			}

			fmt.Printf("\n%s Deleted %d workflow(s):\n", workflow.Green("✓"), len(deleted))
			for _, name := range deleted {
				fmt.Printf("  %s %s\n", workflow.Green("✓"), name)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")

	return cmd
}
