package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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

	rootCmd.AddCommand(newStartCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newResumeCmd())
	rootCmd.AddCommand(newDeleteCmd())
	rootCmd.AddCommand(newCleanCmd())

	return rootCmd
}

func newStartCmd() *cobra.Command {
	var workflowType string

	cmd := &cobra.Command{
		Use:   "start <name> <description>",
		Short: "Start a new workflow",
		Long:  `Start a new workflow with the given name and description.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented")
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
			return fmt.Errorf("not implemented")
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
			return fmt.Errorf("not implemented")
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
			return fmt.Errorf("not implemented")
		},
	}
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a workflow",
		Long:  `Delete a workflow and all its state.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented")
		},
	}
}

func newCleanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Delete all completed workflows",
		Long:  `Delete all workflows that have completed successfully.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented")
		},
	}
}
