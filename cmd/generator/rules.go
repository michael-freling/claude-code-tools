package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/michael-freling/claude-code-tools/internal/generator"
	"github.com/spf13/cobra"
)

func isValidRuleName(name string) bool {
	if name == "" {
		return false
	}
	if strings.ContainsAny(name, "/\\") {
		return false
	}
	if strings.Contains(name, "..") {
		return false
	}
	return true
}

func newRulesCmd() *cobra.Command {
	var paths []string
	var outputDir string
	var filename string

	cmd := &cobra.Command{
		Use:   "rules [name|list]",
		Short: "Generate prompt for a specific rule or list available rules",
		Long:  `Generate prompt for a specific rule by name, or use "list" to show available rules.`,
		Example: `  # List available rules
  generator rules list

  # Generate golang rule to stdout
  generator rules golang

  # Generate with custom paths
  generator rules golang --paths "src/**/*.go" --paths "pkg/**/*.go"

  # Generate to file with default name (golang.md)
  generator rules golang --output-dir .claude/rules/

  # Generate to file with custom name
  generator rules golang --output-dir .claude/rules/ --filename custom-golang.md`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gen, err := createGenerator()
			if err != nil {
				return fmt.Errorf("failed to create generator: %w", err)
			}

			ruleName := args[0]

			if ruleName == "list" {
				rules := gen.List(generator.ItemTypeRule)
				for _, name := range rules {
					fmt.Println(name)
				}
				return nil
			}

			if !isValidRuleName(ruleName) {
				return fmt.Errorf("invalid rule name %q: rule names cannot contain path separators (/, \\) or parent directory traversal (..)", ruleName)
			}

			content, err := gen.GenerateRuleWithOptions(ruleName, generator.GenerateOptions{
				Paths: paths,
			})
			if err != nil {
				return enhanceRuleError(err, ruleName, gen)
			}

			if outputDir != "" {
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
				}

				outputFilename := filename
				if outputFilename == "" {
					outputFilename = fmt.Sprintf("%s.md", ruleName)
				}

				outputPath := filepath.Join(outputDir, outputFilename)
				if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
					return fmt.Errorf("failed to write file %s: %w", outputPath, err)
				}

				fmt.Printf("Created %s\n", outputPath)
				return nil
			}

			fmt.Println(content)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&paths, "paths", []string{}, "Override default paths from metadata")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Write output to file in specified directory")
	cmd.Flags().StringVar(&filename, "filename", "", "Custom output filename (default: {template-name}.md)")

	cmd.AddCommand(newRulesInitCmd())

	return cmd
}

func newRulesInitCmd() *cobra.Command {
	var dir string
	var rules []string
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize rules directory with default or specified rules",
		Long:  `Generate all default rules to files in .claude/rules/ directory or custom directory.`,
		Example: `  # Initialize all default rules in .claude/rules/
  generator rules init

  # Initialize rules in custom directory
  generator rules init --dir custom-rules/

  # Initialize specific rules only
  generator rules init --rules golang --rules common

  # Overwrite existing files
  generator rules init --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			gen, err := createGenerator()
			if err != nil {
				return fmt.Errorf("failed to create generator: %w", err)
			}

			rulesToGenerate := rules
			if len(rulesToGenerate) == 0 {
				rulesToGenerate = gen.GetDefaultRules()
			}

			if len(rulesToGenerate) == 0 {
				return fmt.Errorf("no rules to generate (no default rules configured and --rules not specified)")
			}

			if err := gen.InitRulesDirectory(dir, rulesToGenerate, force); err != nil {
				return fmt.Errorf("failed to initialize rules directory: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".claude/rules", "Target directory (default: .claude/rules/)")
	cmd.Flags().StringSliceVar(&rules, "rules", []string{}, "Selective rules to generate (overrides default_rules)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")

	return cmd
}

func enhanceRuleError(err error, ruleName string, gen *generator.Generator) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "not found") {
		availableRules := gen.List(generator.ItemTypeRule)
		if len(availableRules) > 0 {
			return fmt.Errorf("rule %q not found. Use 'generator rules list' to see available rules.\n\nAvailable rules:\n  - %s",
				ruleName, strings.Join(availableRules, "\n  - "))
		}
		return fmt.Errorf("rule %q not found. Use 'generator rules list' to see available rules", ruleName)
	}

	return fmt.Errorf("failed to generate rule: %w", err)
}
