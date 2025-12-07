package main

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootCmd(t *testing.T) {
	cmd := newRootCmd()

	assert.Equal(t, "claude-workflow", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	commandNames := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		commandNames = append(commandNames, c.Name())
	}
	assert.ElementsMatch(t, []string{"start", "list", "status", "resume", "delete", "clean"}, commandNames)

	persistentFlags := cmd.PersistentFlags()
	assert.NotNil(t, persistentFlags.Lookup("base-dir"))
	assert.NotNil(t, persistentFlags.Lookup("max-lines"))
	assert.NotNil(t, persistentFlags.Lookup("max-files"))
	assert.NotNil(t, persistentFlags.Lookup("claude-path"))
	assert.NotNil(t, persistentFlags.Lookup("dangerously-skip-permissions"))
	assert.NotNil(t, persistentFlags.Lookup("timeout-planning"))
	assert.NotNil(t, persistentFlags.Lookup("timeout-implementation"))
	assert.NotNil(t, persistentFlags.Lookup("timeout-refactoring"))
	assert.NotNil(t, persistentFlags.Lookup("timeout-pr-split"))
}

func TestSubcommands(t *testing.T) {
	tests := []struct {
		name         string
		cmdFunc      func() *cobra.Command
		expectedUse  string
		expectedArgs cobra.PositionalArgs
	}{
		{
			name:         "start command",
			cmdFunc:      newStartCmd,
			expectedUse:  "start <name> <description>",
			expectedArgs: cobra.ExactArgs(2),
		},
		{
			name:         "list command",
			cmdFunc:      newListCmd,
			expectedUse:  "list",
			expectedArgs: cobra.NoArgs,
		},
		{
			name:         "status command",
			cmdFunc:      newStatusCmd,
			expectedUse:  "status <name>",
			expectedArgs: cobra.ExactArgs(1),
		},
		{
			name:         "resume command",
			cmdFunc:      newResumeCmd,
			expectedUse:  "resume <name>",
			expectedArgs: cobra.ExactArgs(1),
		},
		{
			name:         "delete command",
			cmdFunc:      newDeleteCmd,
			expectedUse:  "delete <name>",
			expectedArgs: cobra.ExactArgs(1),
		},
		{
			name:         "clean command",
			cmdFunc:      newCleanCmd,
			expectedUse:  "clean",
			expectedArgs: cobra.NoArgs,
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

func TestPersistentFlags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		flagType     string
		defaultValue string
	}{
		{
			name:         "base-dir flag",
			flagName:     "base-dir",
			flagType:     "string",
			defaultValue: ".claude/workflow",
		},
		{
			name:         "max-lines flag",
			flagName:     "max-lines",
			flagType:     "int",
			defaultValue: "100",
		},
		{
			name:         "max-files flag",
			flagName:     "max-files",
			flagType:     "int",
			defaultValue: "10",
		},
		{
			name:         "claude-path flag",
			flagName:     "claude-path",
			flagType:     "string",
			defaultValue: "claude",
		},
		{
			name:         "dangerously-skip-permissions flag",
			flagName:     "dangerously-skip-permissions",
			flagType:     "bool",
			defaultValue: "false",
		},
		{
			name:         "timeout-planning flag",
			flagName:     "timeout-planning",
			flagType:     "duration",
			defaultValue: "1h0m0s",
		},
		{
			name:         "timeout-implementation flag",
			flagName:     "timeout-implementation",
			flagType:     "duration",
			defaultValue: "6h0m0s",
		},
		{
			name:         "timeout-refactoring flag",
			flagName:     "timeout-refactoring",
			flagType:     "duration",
			defaultValue: "6h0m0s",
		},
		{
			name:         "timeout-pr-split flag",
			flagName:     "timeout-pr-split",
			flagType:     "duration",
			defaultValue: "1h0m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newRootCmd()
			flag := cmd.PersistentFlags().Lookup(tt.flagName)

			require.NotNil(t, flag, "flag %s should exist", tt.flagName)
			assert.Equal(t, tt.flagType, flag.Value.Type())
			assert.Equal(t, tt.defaultValue, flag.DefValue)
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
			name:       "start with correct args",
			cmdFunc:    newStartCmd,
			args:       []string{"test-workflow", "test description"},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name:       "start with too few args",
			cmdFunc:    newStartCmd,
			args:       []string{"test-workflow"},
			wantErr:    true,
			wantErrMsg: "accepts 2 arg(s), received 1",
		},
		{
			name:       "start with too many args",
			cmdFunc:    newStartCmd,
			args:       []string{"test-workflow", "description", "extra"},
			wantErr:    true,
			wantErrMsg: "accepts 2 arg(s), received 3",
		},
		{
			name:       "list with no args",
			cmdFunc:    newListCmd,
			args:       []string{},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name:       "list with args",
			cmdFunc:    newListCmd,
			args:       []string{"extra"},
			wantErr:    true,
			wantErrMsg: "unknown command",
		},
		{
			name:       "status with correct args",
			cmdFunc:    newStatusCmd,
			args:       []string{"test-workflow"},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name:       "status with no args",
			cmdFunc:    newStatusCmd,
			args:       []string{},
			wantErr:    true,
			wantErrMsg: "accepts 1 arg(s), received 0",
		},
		{
			name:       "resume with correct args",
			cmdFunc:    newResumeCmd,
			args:       []string{"test-workflow"},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name:       "resume with no args",
			cmdFunc:    newResumeCmd,
			args:       []string{},
			wantErr:    true,
			wantErrMsg: "accepts 1 arg(s), received 0",
		},
		{
			name:       "delete with correct args",
			cmdFunc:    newDeleteCmd,
			args:       []string{"test-workflow"},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name:       "delete with no args",
			cmdFunc:    newDeleteCmd,
			args:       []string{},
			wantErr:    true,
			wantErrMsg: "accepts 1 arg(s), received 0",
		},
		{
			name:       "clean with no args",
			cmdFunc:    newCleanCmd,
			args:       []string{},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name:       "clean with args",
			cmdFunc:    newCleanCmd,
			args:       []string{"extra"},
			wantErr:    true,
			wantErrMsg: "unknown command",
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

func TestHelpText(t *testing.T) {
	tests := []struct {
		name    string
		cmdFunc func() *cobra.Command
	}{
		{
			name:    "root command help",
			cmdFunc: newRootCmd,
		},
		{
			name:    "start command help",
			cmdFunc: newStartCmd,
		},
		{
			name:    "list command help",
			cmdFunc: newListCmd,
		},
		{
			name:    "status command help",
			cmdFunc: newStatusCmd,
		},
		{
			name:    "resume command help",
			cmdFunc: newResumeCmd,
		},
		{
			name:    "delete command help",
			cmdFunc: newDeleteCmd,
		},
		{
			name:    "clean command help",
			cmdFunc: newCleanCmd,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmdFunc()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			err := cmd.Help()
			assert.NoError(t, err)
			assert.NotEmpty(t, buf.String())
		})
	}
}

func TestCommandFlags(t *testing.T) {
	tests := []struct {
		name         string
		cmdFunc      func() *cobra.Command
		flagName     string
		flagType     string
		defaultValue string
	}{
		{
			name:         "start command type flag",
			cmdFunc:      newStartCmd,
			flagName:     "type",
			flagType:     "string",
			defaultValue: "",
		},
		{
			name:         "delete command force flag",
			cmdFunc:      newDeleteCmd,
			flagName:     "force",
			flagType:     "bool",
			defaultValue: "false",
		},
		{
			name:         "clean command force flag",
			cmdFunc:      newCleanCmd,
			flagName:     "force",
			flagType:     "bool",
			defaultValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmdFunc()
			flag := cmd.Flags().Lookup(tt.flagName)

			require.NotNil(t, flag)
			assert.Equal(t, tt.flagType, flag.Value.Type())
			assert.Equal(t, tt.defaultValue, flag.DefValue)
		})
	}
}
