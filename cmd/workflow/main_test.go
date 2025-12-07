package main

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootCmd(t *testing.T) {
	cmd := newRootCmd()

	assert.Equal(t, "claude-workflow", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.Len(t, cmd.Commands(), 6)

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
		hasRunE      bool
	}{
		{
			name:         "start command",
			cmdFunc:      newStartCmd,
			expectedUse:  "start <name> <description>",
			expectedArgs: cobra.ExactArgs(2),
			hasRunE:      true,
		},
		{
			name:         "list command",
			cmdFunc:      newListCmd,
			expectedUse:  "list",
			expectedArgs: cobra.NoArgs,
			hasRunE:      true,
		},
		{
			name:         "status command",
			cmdFunc:      newStatusCmd,
			expectedUse:  "status <name>",
			expectedArgs: cobra.ExactArgs(1),
			hasRunE:      true,
		},
		{
			name:         "resume command",
			cmdFunc:      newResumeCmd,
			expectedUse:  "resume <name>",
			expectedArgs: cobra.ExactArgs(1),
			hasRunE:      true,
		},
		{
			name:         "delete command",
			cmdFunc:      newDeleteCmd,
			expectedUse:  "delete <name>",
			expectedArgs: cobra.ExactArgs(1),
			hasRunE:      true,
		},
		{
			name:         "clean command",
			cmdFunc:      newCleanCmd,
			expectedUse:  "clean",
			expectedArgs: cobra.NoArgs,
			hasRunE:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmdFunc()

			assert.Equal(t, tt.expectedUse, cmd.Use)
			assert.NotEmpty(t, cmd.Short)
			assert.NotEmpty(t, cmd.Long)

			if tt.hasRunE {
				assert.NotNil(t, cmd.RunE)
			}

			if tt.expectedArgs != nil {
				err := cmd.Args(cmd, make([]string, 0))
				expectedErr := tt.expectedArgs(cmd, make([]string, 0))

				if expectedErr != nil {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

func TestPersistentFlags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		flagType     string
		defaultValue interface{}
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
			defaultValue: 100,
		},
		{
			name:         "max-files flag",
			flagName:     "max-files",
			flagType:     "int",
			defaultValue: 10,
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
			defaultValue: false,
		},
		{
			name:         "timeout-planning flag",
			flagName:     "timeout-planning",
			flagType:     "duration",
			defaultValue: 1 * time.Hour,
		},
		{
			name:         "timeout-implementation flag",
			flagName:     "timeout-implementation",
			flagType:     "duration",
			defaultValue: 6 * time.Hour,
		},
		{
			name:         "timeout-refactoring flag",
			flagName:     "timeout-refactoring",
			flagType:     "duration",
			defaultValue: 6 * time.Hour,
		},
		{
			name:         "timeout-pr-split flag",
			flagName:     "timeout-pr-split",
			flagType:     "duration",
			defaultValue: 1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newRootCmd()
			flag := cmd.PersistentFlags().Lookup(tt.flagName)

			require.NotNil(t, flag, "flag %s should exist", tt.flagName)
			assert.Equal(t, tt.flagType, flag.Value.Type())

			switch v := tt.defaultValue.(type) {
			case string:
				assert.Equal(t, v, flag.DefValue)
			case int:
				assert.Equal(t, fmt.Sprintf("%d", v), flag.DefValue)
			case bool:
				if v {
					assert.Equal(t, "true", flag.DefValue)
				} else {
					assert.Equal(t, "false", flag.DefValue)
				}
			case time.Duration:
				assert.Equal(t, v.String(), flag.DefValue)
			}
		})
	}
}

func TestCommandArgs(t *testing.T) {
	tests := []struct {
		name      string
		cmdFunc   func() *cobra.Command
		args      []string
		wantErr   bool
		errString string
	}{
		{
			name:      "start with correct args",
			cmdFunc:   newStartCmd,
			args:      []string{"test-workflow", "test description"},
			wantErr:   false,
			errString: "",
		},
		{
			name:      "start with too few args",
			cmdFunc:   newStartCmd,
			args:      []string{"test-workflow"},
			wantErr:   true,
			errString: "accepts 2 arg(s), received 1",
		},
		{
			name:      "start with too many args",
			cmdFunc:   newStartCmd,
			args:      []string{"test-workflow", "description", "extra"},
			wantErr:   true,
			errString: "accepts 2 arg(s), received 3",
		},
		{
			name:      "list with no args",
			cmdFunc:   newListCmd,
			args:      []string{},
			wantErr:   false,
			errString: "",
		},
		{
			name:      "list with args",
			cmdFunc:   newListCmd,
			args:      []string{"extra"},
			wantErr:   true,
			errString: "unknown command",
		},
		{
			name:      "status with correct args",
			cmdFunc:   newStatusCmd,
			args:      []string{"test-workflow"},
			wantErr:   false,
			errString: "",
		},
		{
			name:      "status with no args",
			cmdFunc:   newStatusCmd,
			args:      []string{},
			wantErr:   true,
			errString: "accepts 1 arg(s), received 0",
		},
		{
			name:      "resume with correct args",
			cmdFunc:   newResumeCmd,
			args:      []string{"test-workflow"},
			wantErr:   false,
			errString: "",
		},
		{
			name:      "resume with no args",
			cmdFunc:   newResumeCmd,
			args:      []string{},
			wantErr:   true,
			errString: "accepts 1 arg(s), received 0",
		},
		{
			name:      "delete with correct args",
			cmdFunc:   newDeleteCmd,
			args:      []string{"test-workflow"},
			wantErr:   false,
			errString: "",
		},
		{
			name:      "delete with no args",
			cmdFunc:   newDeleteCmd,
			args:      []string{},
			wantErr:   true,
			errString: "accepts 1 arg(s), received 0",
		},
		{
			name:      "clean with no args",
			cmdFunc:   newCleanCmd,
			args:      []string{},
			wantErr:   false,
			errString: "",
		},
		{
			name:      "clean with args",
			cmdFunc:   newCleanCmd,
			args:      []string{"extra"},
			wantErr:   true,
			errString: "unknown command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmdFunc()
			err := cmd.Args(cmd, tt.args)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errString)
			} else {
				assert.NoError(t, err)
			}
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

func TestStartCommandFlags(t *testing.T) {
	cmd := newStartCmd()

	typeFlag := cmd.Flags().Lookup("type")
	require.NotNil(t, typeFlag)
	assert.Equal(t, "string", typeFlag.Value.Type())
	assert.Equal(t, "", typeFlag.DefValue)

	annotations := cmd.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	for annotation := range cmd.Annotations {
		if annotation == cobra.BashCompOneRequiredFlag {
			assert.Contains(t, cmd.Annotations[cobra.BashCompOneRequiredFlag], "type")
		}
	}
}

func TestDeleteCommandFlags(t *testing.T) {
	cmd := newDeleteCmd()

	forceFlag := cmd.Flags().Lookup("force")
	require.NotNil(t, forceFlag)
	assert.Equal(t, "bool", forceFlag.Value.Type())
	assert.Equal(t, "false", forceFlag.DefValue)
}

func TestCleanCommandFlags(t *testing.T) {
	cmd := newCleanCmd()

	forceFlag := cmd.Flags().Lookup("force")
	require.NotNil(t, forceFlag)
	assert.Equal(t, "bool", forceFlag.Value.Type())
	assert.Equal(t, "false", forceFlag.DefValue)
}
