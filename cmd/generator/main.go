package main

import (
	"fmt"
	"os"

	"github.com/michael-freling/claude-code-tools/internal/generator"
	"github.com/spf13/cobra"
)

var templateDir string

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "generator",
		Short: "Generate Claude Code prompts for skills, agents, commands, and rules",
		Long:  `A CLI tool to generate Claude Code prompts from templates for skills, agents, commands, and rules.`,
	}

	rootCmd.PersistentFlags().StringVarP(&templateDir, "template-dir", "t", "", "directory containing custom templates (default: use embedded templates)")

	rootCmd.AddCommand(newAgentsCmd())
	rootCmd.AddCommand(newCommandsCmd())
	rootCmd.AddCommand(newRulesCmd())
	rootCmd.AddCommand(newSkillsCmd())

	return rootCmd
}

func createGenerator() (*generator.Generator, error) {
	if templateDir == "" {
		return generator.NewGenerator()
	}

	// Validate that the directory exists and is readable
	info, err := os.Stat(templateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("template directory does not exist: %s", templateDir)
		}
		return nil, fmt.Errorf("failed to access template directory %s: %w", templateDir, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("template path is not a directory: %s", templateDir)
	}

	fsys := os.DirFS(templateDir)
	return generator.NewGeneratorWithFS(fsys)
}

func newAgentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents [name|list]",
		Short: "Generate prompt for a specific agent or list available agents",
		Long:  `Generate prompt for a specific agent by name, or use "list" to show available agents.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gen, err := createGenerator()
			if err != nil {
				return fmt.Errorf("failed to create generator: %w", err)
			}

			if args[0] == "list" {
				agents := gen.List(generator.ItemTypeAgent)
				for _, name := range agents {
					fmt.Println(name)
				}
				return nil
			}

			if err := gen.Generate(generator.ItemTypeAgent, args[0]); err != nil {
				return fmt.Errorf("failed to generate agent: %w", err)
			}

			return nil
		},
	}

	return cmd
}

func newCommandsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commands [name|list]",
		Short: "Generate prompt for a specific command or list available commands",
		Long:  `Generate prompt for a specific command by name, or use "list" to show available commands.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gen, err := createGenerator()
			if err != nil {
				return fmt.Errorf("failed to create generator: %w", err)
			}

			if args[0] == "list" {
				commands := gen.List(generator.ItemTypeCommand)
				for _, name := range commands {
					fmt.Println(name)
				}
				return nil
			}

			if err := gen.Generate(generator.ItemTypeCommand, args[0]); err != nil {
				return fmt.Errorf("failed to generate command: %w", err)
			}

			return nil
		},
	}

	return cmd
}

func newSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills [name|list]",
		Short: "Generate prompt for a specific skill or list available skills",
		Long:  `Generate prompt for a specific skill by name, or use "list" to show available skills.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gen, err := createGenerator()
			if err != nil {
				return fmt.Errorf("failed to create generator: %w", err)
			}

			if args[0] == "list" {
				skills := gen.List(generator.ItemTypeSkill)
				for _, name := range skills {
					fmt.Println(name)
				}
				return nil
			}

			if err := gen.Generate(generator.ItemTypeSkill, args[0]); err != nil {
				return fmt.Errorf("failed to generate skill: %w", err)
			}

			return nil
		},
	}

	return cmd
}
