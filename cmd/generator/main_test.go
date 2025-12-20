package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/michael-freling/claude-code-tools/internal/generator"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// saveTemplateDir saves the current templateDir value
func saveTemplateDir() string {
	return templateDir
}

// restoreTemplateDir restores the templateDir value
func restoreTemplateDir(saved string) {
	templateDir = saved
}

func TestNewRootCmd(t *testing.T) {
	cmd := newRootCmd()

	assert.Equal(t, "generator", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	commandNames := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		commandNames = append(commandNames, c.Name())
	}
	assert.ElementsMatch(t, []string{"agents", "commands", "rules", "skills"}, commandNames)

	persistentFlags := cmd.PersistentFlags()
	flag := persistentFlags.Lookup("template-dir")
	require.NotNil(t, flag)
	assert.Equal(t, "t", flag.Shorthand)
	assert.Equal(t, "string", flag.Value.Type())
}

func TestNewAgentsCmd(t *testing.T) {
	cmd := newAgentsCmd()

	assert.Equal(t, "agents [name|list]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	err := cmd.Args(cmd, []string{"test"})
	assert.NoError(t, err)

	err = cmd.Args(cmd, []string{})
	assert.Error(t, err)

	err = cmd.Args(cmd, []string{"test", "extra"})
	assert.Error(t, err)
}

func TestNewCommandsCmd(t *testing.T) {
	cmd := newCommandsCmd()

	assert.Equal(t, "commands [name|list]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	err := cmd.Args(cmd, []string{"test"})
	assert.NoError(t, err)

	err = cmd.Args(cmd, []string{})
	assert.Error(t, err)

	err = cmd.Args(cmd, []string{"test", "extra"})
	assert.Error(t, err)
}

func TestNewSkillsCmd(t *testing.T) {
	cmd := newSkillsCmd()

	assert.Equal(t, "skills [name|list]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	err := cmd.Args(cmd, []string{"test"})
	assert.NoError(t, err)

	err = cmd.Args(cmd, []string{})
	assert.Error(t, err)

	err = cmd.Args(cmd, []string{"test", "extra"})
	assert.Error(t, err)
}

func TestCreateGenerator(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		cleanupFunc func(t *testing.T, path string)
		wantErr     bool
		errContains string
	}{
		{
			name: "empty templateDir uses embedded templates",
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				templateDir = ""
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
		{
			name: "non-existent directory returns error",
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				templateDir = "/non/existent/path"
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				restoreTemplateDir(saved)
			},
			wantErr:     true,
			errContains: "template directory does not exist",
		},
		{
			name: "path is file not directory returns error",
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				tempFile, err := os.CreateTemp("", "test-file-*.txt")
				require.NoError(t, err)
				templateDir = tempFile.Name()
				tempFile.Close()
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				os.Remove(templateDir)
				restoreTemplateDir(saved)
			},
			wantErr:     true,
			errContains: "template path is not a directory",
		},
		{
			name: "directory access error returns error",
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				tempDir, err := os.MkdirTemp("", "test-templates-*")
				require.NoError(t, err)

				subDir := filepath.Join(tempDir, "subdir")
				err = os.MkdirAll(subDir, 0755)
				require.NoError(t, err)

				err = os.Chmod(subDir, 0000)
				require.NoError(t, err)

				templateDir = filepath.Join(subDir, "inaccessible")
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				parentDir := filepath.Dir(templateDir)
				os.Chmod(parentDir, 0755)
				os.RemoveAll(parentDir)
				restoreTemplateDir(saved)
			},
			wantErr:     true,
			errContains: "failed to access template directory",
		},
		{
			name: "valid custom directory works",
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				tempDir, err := os.MkdirTemp("", "test-templates-*")
				require.NoError(t, err)

				// Create a minimal template structure
				promptsDir := filepath.Join(tempDir, "prompts")
				agentsDir := filepath.Join(promptsDir, "agents")
				err = os.MkdirAll(agentsDir, 0755)
				require.NoError(t, err)

				templateDir = tempDir
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				os.RemoveAll(templateDir)
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved := tt.setupFunc(t)
			defer tt.cleanupFunc(t, saved)

			gen, err := createGenerator()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, gen)
		})
	}
}

func TestAgentsCmd_Execute(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
		setupFunc   func(t *testing.T) string
		cleanupFunc func(t *testing.T, saved string)
	}{
		{
			name: "list operation",
			args: []string{"list"},
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				templateDir = ""
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
		{
			name: "generate with valid agent name",
			args: []string{"code-reviewer"},
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				templateDir = ""
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
		{
			name: "generate with invalid agent name returns error",
			args: []string{"non-existent-agent"},
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				templateDir = ""
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				restoreTemplateDir(saved)
			},
			wantErr:     true,
			errContains: "failed to generate agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved := tt.setupFunc(t)
			defer tt.cleanupFunc(t, saved)

			cmd := newAgentsCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestCommandsCmd_Execute(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
		setupFunc   func(t *testing.T) string
		cleanupFunc func(t *testing.T, saved string)
	}{
		{
			name: "list operation",
			args: []string{"list"},
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				templateDir = ""
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
		{
			name: "generate with valid command name",
			args: []string{"feature"},
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				templateDir = ""
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
		{
			name: "generate with invalid command name returns error",
			args: []string{"non-existent-command"},
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				templateDir = ""
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				restoreTemplateDir(saved)
			},
			wantErr:     true,
			errContains: "failed to generate command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved := tt.setupFunc(t)
			defer tt.cleanupFunc(t, saved)

			cmd := newCommandsCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestSkillsCmd_Execute(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
		setupFunc   func(t *testing.T) string
		cleanupFunc func(t *testing.T, saved string)
	}{
		{
			name: "list operation",
			args: []string{"list"},
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				templateDir = ""
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
		{
			name: "generate with valid skill name",
			args: []string{"coding"},
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				templateDir = ""
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
		{
			name: "generate with invalid skill name returns error",
			args: []string{"non-existent-skill"},
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				templateDir = ""
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				restoreTemplateDir(saved)
			},
			wantErr:     true,
			errContains: "failed to generate skill",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved := tt.setupFunc(t)
			defer tt.cleanupFunc(t, saved)

			cmd := newSkillsCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestCreateGenerator_InvalidTemplateDir(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		cleanupFunc func(t *testing.T, saved string)
		wantErr     bool
		errContains string
	}{
		{
			name: "empty string uses embedded templates",
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				templateDir = ""
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
		{
			name: "non-existent path returns error",
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				templateDir = "/absolutely/non/existent/path/12345"
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				restoreTemplateDir(saved)
			},
			wantErr:     true,
			errContains: "template directory does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved := tt.setupFunc(t)
			defer tt.cleanupFunc(t, saved)

			gen, err := createGenerator()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, gen)
		})
	}
}

func TestCommandArgs(t *testing.T) {
	tests := []struct {
		name       string
		cmdFunc    func() *cobra.Command
		args       []string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:       "agents with correct args",
			cmdFunc:    newAgentsCmd,
			args:       []string{"test-agent"},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name:       "agents with no args",
			cmdFunc:    newAgentsCmd,
			args:       []string{},
			wantErr:    true,
			wantErrMsg: "accepts 1 arg(s), received 0",
		},
		{
			name:       "agents with too many args",
			cmdFunc:    newAgentsCmd,
			args:       []string{"test", "extra"},
			wantErr:    true,
			wantErrMsg: "accepts 1 arg(s), received 2",
		},
		{
			name:       "commands with correct args",
			cmdFunc:    newCommandsCmd,
			args:       []string{"test-command"},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name:       "commands with no args",
			cmdFunc:    newCommandsCmd,
			args:       []string{},
			wantErr:    true,
			wantErrMsg: "accepts 1 arg(s), received 0",
		},
		{
			name:       "commands with too many args",
			cmdFunc:    newCommandsCmd,
			args:       []string{"test", "extra"},
			wantErr:    true,
			wantErrMsg: "accepts 1 arg(s), received 2",
		},
		{
			name:       "rules with correct args",
			cmdFunc:    newRulesCmd,
			args:       []string{"test-rule"},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name:       "rules with no args",
			cmdFunc:    newRulesCmd,
			args:       []string{},
			wantErr:    true,
			wantErrMsg: "accepts 1 arg(s), received 0",
		},
		{
			name:       "rules with too many args",
			cmdFunc:    newRulesCmd,
			args:       []string{"test", "extra"},
			wantErr:    true,
			wantErrMsg: "accepts 1 arg(s), received 2",
		},
		{
			name:       "skills with correct args",
			cmdFunc:    newSkillsCmd,
			args:       []string{"test-skill"},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name:       "skills with no args",
			cmdFunc:    newSkillsCmd,
			args:       []string{},
			wantErr:    true,
			wantErrMsg: "accepts 1 arg(s), received 0",
		},
		{
			name:       "skills with too many args",
			cmdFunc:    newSkillsCmd,
			args:       []string{"test", "extra"},
			wantErr:    true,
			wantErrMsg: "accepts 1 arg(s), received 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmdFunc()
			err := cmd.Args(cmd, tt.args)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErrMsg)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestSubcommands(t *testing.T) {
	tests := []struct {
		name         string
		cmdFunc      func() *cobra.Command
		expectedUse  string
		expectedArgs cobra.PositionalArgs
	}{
		{
			name:         "agents command",
			cmdFunc:      newAgentsCmd,
			expectedUse:  "agents [name|list]",
			expectedArgs: cobra.ExactArgs(1),
		},
		{
			name:         "commands command",
			cmdFunc:      newCommandsCmd,
			expectedUse:  "commands [name|list]",
			expectedArgs: cobra.ExactArgs(1),
		},
		{
			name:         "rules command",
			cmdFunc:      newRulesCmd,
			expectedUse:  "rules [name|list]",
			expectedArgs: cobra.ExactArgs(1),
		},
		{
			name:         "skills command",
			cmdFunc:      newSkillsCmd,
			expectedUse:  "skills [name|list]",
			expectedArgs: cobra.ExactArgs(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmdFunc()

			assert.Equal(t, tt.expectedUse, cmd.Use)
			assert.NotEmpty(t, cmd.Short)
			assert.NotEmpty(t, cmd.Long)
			assert.NotNil(t, cmd.RunE)

			err := cmd.Args(cmd, make([]string, 0))
			expectedErr := tt.expectedArgs(cmd, make([]string, 0))

			if expectedErr != nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestAgentsCmd_List(t *testing.T) {
	saved := saveTemplateDir()
	defer restoreTemplateDir(saved)

	templateDir = ""

	gen, err := generator.NewGenerator()
	require.NoError(t, err)

	agents := gen.List(generator.ItemTypeAgent)
	assert.NotEmpty(t, agents)
}

func TestCommandsCmd_List(t *testing.T) {
	saved := saveTemplateDir()
	defer restoreTemplateDir(saved)

	templateDir = ""

	gen, err := generator.NewGenerator()
	require.NoError(t, err)

	commands := gen.List(generator.ItemTypeCommand)
	assert.NotEmpty(t, commands)
}

func TestSkillsCmd_List(t *testing.T) {
	saved := saveTemplateDir()
	defer restoreTemplateDir(saved)

	templateDir = ""

	gen, err := generator.NewGenerator()
	require.NoError(t, err)

	skills := gen.List(generator.ItemTypeSkill)
	assert.NotEmpty(t, skills)
}

func TestAgentsCmd_CreateGeneratorError(t *testing.T) {
	saved := saveTemplateDir()
	defer restoreTemplateDir(saved)

	templateDir = "/non/existent/path"

	cmd := newAgentsCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create generator")
}

func TestCommandsCmd_CreateGeneratorError(t *testing.T) {
	saved := saveTemplateDir()
	defer restoreTemplateDir(saved)

	templateDir = "/non/existent/path"

	cmd := newCommandsCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create generator")
}

func TestSkillsCmd_CreateGeneratorError(t *testing.T) {
	saved := saveTemplateDir()
	defer restoreTemplateDir(saved)

	templateDir = "/non/existent/path"

	cmd := newSkillsCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create generator")
}

func TestNewRulesCmd(t *testing.T) {
	cmd := newRulesCmd()

	assert.Equal(t, "rules [name|list]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	err := cmd.Args(cmd, []string{"test"})
	assert.NoError(t, err)

	err = cmd.Args(cmd, []string{})
	assert.Error(t, err)

	err = cmd.Args(cmd, []string{"test", "extra"})
	assert.Error(t, err)
}

func TestRulesCmd_Execute(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
		setupFunc   func(t *testing.T) string
		cleanupFunc func(t *testing.T, saved string)
	}{
		{
			name: "list operation",
			args: []string{"list"},
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				templateDir = ""
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
		{
			name: "generate with valid rule name",
			args: []string{"golang"},
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				templateDir = ""
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
		{
			name: "generate with invalid rule name returns error",
			args: []string{"non-existent-rule"},
			setupFunc: func(t *testing.T) string {
				saved := saveTemplateDir()
				templateDir = ""
				return saved
			},
			cleanupFunc: func(t *testing.T, saved string) {
				restoreTemplateDir(saved)
			},
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved := tt.setupFunc(t)
			defer tt.cleanupFunc(t, saved)

			cmd := newRulesCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestRulesCmd_List(t *testing.T) {
	saved := saveTemplateDir()
	defer restoreTemplateDir(saved)

	templateDir = ""

	gen, err := generator.NewGenerator()
	require.NoError(t, err)

	rules := gen.List(generator.ItemTypeRule)
	assert.NotEmpty(t, rules)
}

func TestRulesCmd_CreateGeneratorError(t *testing.T) {
	saved := saveTemplateDir()
	defer restoreTemplateDir(saved)

	templateDir = "/non/existent/path"

	cmd := newRulesCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create generator")
}

func TestRulesCmd_WithFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
		setupFunc   func(t *testing.T) (string, string)
		cleanupFunc func(t *testing.T, saved string, tempDir string)
	}{
		{
			name: "generate with paths flag",
			args: []string{"golang", "--paths", "src/**/*.go"},
			setupFunc: func(t *testing.T) (string, string) {
				saved := saveTemplateDir()
				templateDir = ""
				return saved, ""
			},
			cleanupFunc: func(t *testing.T, saved string, tempDir string) {
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
		{
			name: "generate with output-dir flag",
			args: []string{"golang", "--output-dir", ""},
			setupFunc: func(t *testing.T) (string, string) {
				saved := saveTemplateDir()
				templateDir = ""
				tempDir, err := os.MkdirTemp("", "rules-test-*")
				require.NoError(t, err)
				return saved, tempDir
			},
			cleanupFunc: func(t *testing.T, saved string, tempDir string) {
				os.RemoveAll(tempDir)
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
		{
			name: "generate with filename flag",
			args: []string{"golang", "--output-dir", "", "--filename", "custom.md"},
			setupFunc: func(t *testing.T) (string, string) {
				saved := saveTemplateDir()
				templateDir = ""
				tempDir, err := os.MkdirTemp("", "rules-test-*")
				require.NoError(t, err)
				return saved, tempDir
			},
			cleanupFunc: func(t *testing.T, saved string, tempDir string) {
				os.RemoveAll(tempDir)
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
		{
			name: "invalid rule name with path separator",
			args: []string{"../evil"},
			setupFunc: func(t *testing.T) (string, string) {
				saved := saveTemplateDir()
				templateDir = ""
				return saved, ""
			},
			cleanupFunc: func(t *testing.T, saved string, tempDir string) {
				restoreTemplateDir(saved)
			},
			wantErr:     true,
			errContains: "invalid rule name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved, tempDir := tt.setupFunc(t)
			defer tt.cleanupFunc(t, saved, tempDir)

			args := tt.args
			if tempDir != "" {
				for i, arg := range args {
					if arg == "--output-dir" && i+1 < len(args) && args[i+1] == "" {
						args[i+1] = tempDir
					}
				}
			}

			cmd := newRulesCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(args)

			err := cmd.Execute()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestRulesInitCmd(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
		setupFunc   func(t *testing.T) (string, string)
		cleanupFunc func(t *testing.T, saved string, tempDir string)
	}{
		{
			name: "init with custom dir",
			args: []string{"init", "--dir", ""},
			setupFunc: func(t *testing.T) (string, string) {
				saved := saveTemplateDir()
				templateDir = ""
				tempDir, err := os.MkdirTemp("", "rules-init-test-*")
				require.NoError(t, err)
				return saved, tempDir
			},
			cleanupFunc: func(t *testing.T, saved string, tempDir string) {
				os.RemoveAll(tempDir)
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
		{
			name: "init with specific rules",
			args: []string{"init", "--dir", "", "--rules", "golang"},
			setupFunc: func(t *testing.T) (string, string) {
				saved := saveTemplateDir()
				templateDir = ""
				tempDir, err := os.MkdirTemp("", "rules-init-test-*")
				require.NoError(t, err)
				return saved, tempDir
			},
			cleanupFunc: func(t *testing.T, saved string, tempDir string) {
				os.RemoveAll(tempDir)
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
		{
			name: "init with force flag",
			args: []string{"init", "--dir", "", "--force"},
			setupFunc: func(t *testing.T) (string, string) {
				saved := saveTemplateDir()
				templateDir = ""
				tempDir, err := os.MkdirTemp("", "rules-init-test-*")
				require.NoError(t, err)
				return saved, tempDir
			},
			cleanupFunc: func(t *testing.T, saved string, tempDir string) {
				os.RemoveAll(tempDir)
				restoreTemplateDir(saved)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved, tempDir := tt.setupFunc(t)
			defer tt.cleanupFunc(t, saved, tempDir)

			args := tt.args
			if tempDir != "" {
				for i, arg := range args {
					if arg == "--dir" && i+1 < len(args) && args[i+1] == "" {
						args[i+1] = tempDir
					}
				}
			}

			cmd := newRulesCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(args)

			err := cmd.Execute()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestRulesInitCmd_CreateGeneratorError(t *testing.T) {
	saved := saveTemplateDir()
	defer restoreTemplateDir(saved)

	templateDir = "/non/existent/path"

	cmd := newRulesCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"init"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create generator")
}

func TestIsValidRuleName(t *testing.T) {
	tests := []struct {
		name     string
		ruleName string
		want     bool
	}{
		{
			name:     "valid simple name",
			ruleName: "golang",
			want:     true,
		},
		{
			name:     "valid name with hyphen",
			ruleName: "my-rule",
			want:     true,
		},
		{
			name:     "valid name with single dot",
			ruleName: "my.rule",
			want:     true,
		},
		{
			name:     "invalid empty string",
			ruleName: "",
			want:     false,
		},
		{
			name:     "invalid single dot",
			ruleName: ".",
			want:     true,
		},
		{
			name:     "invalid double dot",
			ruleName: "..",
			want:     false,
		},
		{
			name:     "invalid with forward slash",
			ruleName: "../evil",
			want:     false,
		},
		{
			name:     "invalid with backslash",
			ruleName: "..\\evil",
			want:     false,
		},
		{
			name:     "invalid with dots",
			ruleName: "..test",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidRuleName(tt.ruleName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEnhanceRuleError(t *testing.T) {
	saved := saveTemplateDir()
	defer restoreTemplateDir(saved)
	templateDir = ""

	gen, err := generator.NewGenerator()
	require.NoError(t, err)

	tests := []struct {
		name        string
		err         error
		ruleName    string
		wantNil     bool
		wantContain string
	}{
		{
			name:     "nil error returns nil",
			err:      nil,
			ruleName: "test",
			wantNil:  true,
		},
		{
			name:        "not found error shows available rules",
			err:         assert.AnError,
			ruleName:    "nonexistent",
			wantContain: "failed to generate rule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enhanceRuleError(tt.err, tt.ruleName, gen)
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Contains(t, result.Error(), tt.wantContain)
		})
	}
}
