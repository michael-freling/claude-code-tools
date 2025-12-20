//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/michael-freling/claude-code-tools/test/e2e/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeneratorRules_List tests the 'generator rules list' command
func TestGeneratorRules_List(t *testing.T) {
	tests := []struct {
		name         string
		wantContains []string
	}{
		{
			name: "lists available rules",
			wantContains: []string{
				"golang",
				"common",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("generator", "rules", "list")
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "command should succeed")

			outputStr := string(output)
			for _, want := range tt.wantContains {
				assert.Contains(t, outputStr, want, "output should contain rule name")
			}
		})
	}
}

// TestGeneratorRules_Generate tests the 'generator rules' command
func TestGeneratorRules_Generate(t *testing.T) {
	tests := []struct {
		name         string
		ruleName     string
		args         []string
		wantContains []string
		wantErr      bool
	}{
		{
			name:     "generates golang rule to stdout",
			ruleName: "golang",
			args:     []string{},
			wantContains: []string{
				"paths:",
			},
			wantErr: false,
		},
		{
			name:     "generates common rule to stdout",
			ruleName: "common",
			args:     []string{},
			wantContains: []string{
				"paths:",
			},
			wantErr: false,
		},
		{
			name:     "generates rule with custom paths",
			ruleName: "golang",
			args:     []string{"--paths", "src/**/*.go", "--paths", "pkg/**/*.go"},
			wantContains: []string{
				"src/**/*.go",
				"pkg/**/*.go",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"rules", tt.ruleName}, tt.args...)
			cmd := exec.Command("generator", args...)
			output, err := cmd.CombinedOutput()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			outputStr := string(output)
			for _, want := range tt.wantContains {
				assert.Contains(t, outputStr, want)
			}
		})
	}
}

// TestGeneratorRules_GenerateToFile tests generating rules to files
func TestGeneratorRules_GenerateToFile(t *testing.T) {
	tests := []struct {
		name         string
		ruleName     string
		filename     string
		args         []string
		wantContains []string
		wantErr      bool
	}{
		{
			name:     "generates rule to default filename",
			ruleName: "golang",
			filename: "",
			args:     []string{},
			wantContains: []string{
				"paths:",
			},
			wantErr: false,
		},
		{
			name:     "generates rule to custom filename",
			ruleName: "golang",
			filename: "custom-golang.md",
			args:     []string{"--filename", "custom-golang.md"},
			wantContains: []string{
				"paths:",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "rules-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			args := append([]string{"rules", tt.ruleName, "--output-dir", tmpDir}, tt.args...)
			cmd := exec.Command("generator", args...)
			output, err := cmd.CombinedOutput()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err, "command output: %s", string(output))

			expectedFilename := tt.filename
			if expectedFilename == "" {
				expectedFilename = tt.ruleName + ".md"
			}
			expectedPath := filepath.Join(tmpDir, expectedFilename)

			require.FileExists(t, expectedPath)

			content, err := os.ReadFile(expectedPath)
			require.NoError(t, err)

			contentStr := string(content)
			for _, want := range tt.wantContains {
				assert.Contains(t, contentStr, want)
			}
		})
	}
}

// TestGeneratorRules_Init tests the 'generator rules init' command
func TestGeneratorRules_Init(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantFiles []string
		wantErr   bool
	}{
		{
			name: "initializes default rules directory",
			args: []string{},
			wantFiles: []string{
				"golang.md",
				"common.md",
			},
			wantErr: false,
		},
		{
			name: "initializes with custom directory",
			args: []string{"--dir", "custom-rules"},
			wantFiles: []string{
				"golang.md",
				"common.md",
			},
			wantErr: false,
		},
		{
			name: "initializes with selective rules",
			args: []string{"--rules", "golang"},
			wantFiles: []string{
				"golang.md",
			},
			wantErr: false,
		},
		{
			name: "initializes multiple selective rules",
			args: []string{"--rules", "golang", "--rules", "common"},
			wantFiles: []string{
				"golang.md",
				"common.md",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "rules-init-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			targetDir := ".claude/rules"
			for i, arg := range tt.args {
				if arg == "--dir" && i+1 < len(tt.args) {
					targetDir = tt.args[i+1]
					break
				}
			}

			fullTargetDir := filepath.Join(tmpDir, targetDir)

			args := []string{"rules", "init"}
			if targetDir != ".claude/rules" {
				args = append(args, tt.args...)
			} else {
				args = append(args, "--dir", fullTargetDir)
				args = append(args, tt.args...)
			}

			cmd := exec.Command("generator", args...)
			cmd.Dir = tmpDir
			output, err := cmd.CombinedOutput()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err, "command output: %s", string(output))

			for _, wantFile := range tt.wantFiles {
				expectedPath := filepath.Join(fullTargetDir, wantFile)
				require.FileExists(t, expectedPath, "expected file %s to exist", wantFile)

				content, err := os.ReadFile(expectedPath)
				require.NoError(t, err)
				assert.NotEmpty(t, content, "file %s should not be empty", wantFile)
			}
		})
	}
}

// TestGeneratorRules_Init_Force tests the --force flag behavior
func TestGeneratorRules_Init_Force(t *testing.T) {
	tests := []struct {
		name           string
		setupFiles     []string
		useForce       bool
		wantErr        bool
		wantErrContain string
	}{
		{
			name:           "fails when file exists without force",
			setupFiles:     []string{"golang.md"},
			useForce:       false,
			wantErr:        true,
			wantErrContain: "already exists",
		},
		{
			name:       "succeeds when file exists with force",
			setupFiles: []string{"golang.md"},
			useForce:   true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "rules-force-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			targetDir := filepath.Join(tmpDir, ".claude/rules")
			err = os.MkdirAll(targetDir, 0755)
			require.NoError(t, err)

			for _, file := range tt.setupFiles {
				filePath := filepath.Join(targetDir, file)
				err := os.WriteFile(filePath, []byte("existing content"), 0644)
				require.NoError(t, err)
			}

			args := []string{"rules", "init", "--dir", targetDir, "--rules", "golang"}
			if tt.useForce {
				args = append(args, "--force")
			}

			cmd := exec.Command("generator", args...)
			output, err := cmd.CombinedOutput()

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrContain != "" {
					assert.Contains(t, string(output), tt.wantErrContain)
				}
				return
			}

			require.NoError(t, err, "command output: %s", string(output))
		})
	}
}

// TestGeneratorRules_ErrorCases tests error handling
func TestGeneratorRules_ErrorCases(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantErr        bool
		wantErrContain string
	}{
		{
			name:           "fails with non-existent rule",
			args:           []string{"rules", "non-existent-rule"},
			wantErr:        true,
			wantErrContain: "not found",
		},
		{
			name:    "fails with no arguments",
			args:    []string{"rules"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("generator", tt.args...)
			output, err := cmd.CombinedOutput()

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrContain != "" {
					outputStr := strings.ToLower(string(output))
					assert.Contains(t, outputStr, strings.ToLower(tt.wantErrContain))
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

// TestGeneratorRules_Help tests help text
func TestGeneratorRules_Help(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantContains []string
	}{
		{
			name: "shows rules command help",
			args: []string{"rules", "--help"},
			wantContains: []string{
				"Usage:",
				"Examples:",
				"--paths",
				"--output-dir",
				"--filename",
			},
		},
		{
			name: "shows init subcommand help",
			args: []string{"rules", "init", "--help"},
			wantContains: []string{
				"Usage:",
				"Examples:",
				"--dir",
				"--rules",
				"--force",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("generator", tt.args...)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err)

			outputStr := string(output)
			for _, want := range tt.wantContains {
				assert.Contains(t, outputStr, want)
			}
		})
	}
}

// TestGeneratorRules_CLI_Available tests that the generator CLI is available
func TestGeneratorRules_CLI_Available(t *testing.T) {
	if !helpers.IsCLIAvailable() {
		t.Skip("generator CLI not available in PATH")
	}

	cmd := exec.Command("generator", "rules", "list")
	err := cmd.Run()
	require.NoError(t, err, "generator CLI should be available and executable")
}
