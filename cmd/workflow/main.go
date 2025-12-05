package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/michael-freling/claude-code-config/internal/workflow"
	"github.com/spf13/cobra"
)

var (
	baseDir            string
	maxLines           int
	maxFiles           int
	claudePath         string
	timeoutPlanning    time.Duration
	timeoutImplement   time.Duration
	timeoutRefactoring time.Duration
	timeoutPRSplit     time.Duration
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
	rootCmd.PersistentFlags().IntVar(&maxLines, "max-lines", 100, "PR split threshold for lines")
	rootCmd.PersistentFlags().IntVar(&maxFiles, "max-files", 10, "PR split threshold for files")
	rootCmd.PersistentFlags().StringVar(&claudePath, "claude-path", "claude", "path to claude CLI")
	rootCmd.PersistentFlags().DurationVar(&timeoutPlanning, "timeout-planning", 5*time.Minute, "planning phase timeout")
	rootCmd.PersistentFlags().DurationVar(&timeoutImplement, "timeout-implementation", 30*time.Minute, "implementation phase timeout")
	rootCmd.PersistentFlags().DurationVar(&timeoutRefactoring, "timeout-refactoring", 15*time.Minute, "refactoring phase timeout")
	rootCmd.PersistentFlags().DurationVar(&timeoutPRSplit, "timeout-pr-split", 10*time.Minute, "PR split phase timeout")

	rootCmd.AddCommand(newStartCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newResumeCmd())
	rootCmd.AddCommand(newDeleteCmd())
	rootCmd.AddCommand(newCleanCmd())

	return rootCmd
}

func createOrchestrator() (*workflow.Orchestrator, error) {
	config := &workflow.Config{
		BaseDir:    baseDir,
		MaxLines:   maxLines,
		MaxFiles:   maxFiles,
		ClaudePath: claudePath,
		Timeouts: workflow.PhaseTimeouts{
			Planning:       timeoutPlanning,
			Implementation: timeoutImplement,
			Refactoring:    timeoutRefactoring,
			PRSplit:        timeoutPRSplit,
		},
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

			fmt.Printf("Starting workflow '%s' (type: %s)...\n", name, workflowType)
			fmt.Printf("Description: %s\n\n", description)

			ctx := context.Background()
			if err := orchestrator.Start(ctx, name, description, wfType); err != nil {
				return fmt.Errorf("workflow failed: %w", err)
			}

			fmt.Printf("\nWorkflow '%s' completed successfully!\n", name)
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
				fmt.Println("No workflows found.")
				return nil
			}

			fmt.Println("NAME\tTYPE\tPHASE\tSTATUS\tCREATED\tUPDATED")
			for _, wf := range workflows {
				fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\n",
					wf.Name,
					wf.Type,
					wf.CurrentPhase,
					wf.Status,
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

			fmt.Printf("Workflow: %s\n", state.Name)
			fmt.Printf("Type: %s\n", state.Type)
			fmt.Printf("Description: %s\n", state.Description)
			fmt.Printf("Current Phase: %s\n", state.CurrentPhase)
			fmt.Printf("Created: %s\n", state.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated: %s\n\n", state.UpdatedAt.Format("2006-01-02 15:04:05"))

			fmt.Println("Phases:")
			phases := []workflow.Phase{
				workflow.PhasePlanning,
				workflow.PhaseConfirmation,
				workflow.PhaseImplementation,
				workflow.PhaseRefactoring,
				workflow.PhasePRSplit,
			}

			for _, phase := range phases {
				phaseState := state.Phases[phase]
				if phaseState == nil {
					continue
				}

				fmt.Printf("  %s: %s", phase, phaseState.Status)
				if phaseState.Attempts > 0 {
					fmt.Printf(" (attempts: %d)", phaseState.Attempts)
				}
				if phaseState.StartedAt != nil {
					fmt.Printf(" - started: %s", phaseState.StartedAt.Format("15:04:05"))
				}
				if phaseState.CompletedAt != nil {
					fmt.Printf(" - completed: %s", phaseState.CompletedAt.Format("15:04:05"))
				}
				fmt.Println()

				if len(phaseState.Feedback) > 0 {
					fmt.Println("    Feedback:")
					for _, fb := range phaseState.Feedback {
						fmt.Printf("      - %s\n", fb)
					}
				}

				if phaseState.Metrics != nil {
					fmt.Printf("    Metrics: %d lines, %d files\n",
						phaseState.Metrics.LinesChanged,
						phaseState.Metrics.FilesChanged,
					)
				}
			}

			if state.Error != nil {
				fmt.Printf("\nError: %s\n", state.Error.Message)
				fmt.Printf("Phase: %s\n", state.Error.Phase)
				fmt.Printf("Timestamp: %s\n", state.Error.Timestamp.Format("2006-01-02 15:04:05"))
				fmt.Printf("Recoverable: %v\n", state.Error.Recoverable)
			}

			plan, err := orchestrator.Status(name)
			if err == nil && plan != nil {
				planPath := fmt.Sprintf("%s/%s/plan.json", baseDir, name)
				if _, err := os.Stat(planPath); err == nil {
					fmt.Printf("\nPlan: %s\n", planPath)
				}
			}

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

			fmt.Printf("Resuming workflow '%s'...\n\n", name)

			ctx := context.Background()
			if err := orchestrator.Resume(ctx, name); err != nil {
				return fmt.Errorf("workflow failed: %w", err)
			}

			fmt.Printf("\nWorkflow '%s' completed successfully!\n", name)
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
				fmt.Printf("Are you sure you want to delete workflow '%s'? (y/n): ", name)
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "yes" {
					fmt.Println("Deletion cancelled.")
					return nil
				}
			}

			if err := orchestrator.Delete(name); err != nil {
				return fmt.Errorf("failed to delete workflow: %w", err)
			}

			fmt.Printf("Workflow '%s' deleted successfully.\n", name)
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
				fmt.Println("No completed workflows to clean.")
				return nil
			}

			fmt.Printf("Found %d completed workflow(s):\n", len(completedWorkflows))
			for _, name := range completedWorkflows {
				fmt.Printf("  - %s\n", name)
			}

			if !force {
				fmt.Print("\nDelete all completed workflows? (y/n): ")
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "yes" {
					fmt.Println("Clean cancelled.")
					return nil
				}
			}

			deleted, err := orchestrator.Clean()
			if err != nil {
				return fmt.Errorf("failed to clean workflows: %w", err)
			}

			fmt.Printf("\nDeleted %d workflow(s):\n", len(deleted))
			for _, name := range deleted {
				fmt.Printf("  - %s\n", name)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")

	return cmd
}
