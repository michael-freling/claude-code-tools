package workflow

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
)

// CommandRunner abstracts command execution for testability
type CommandRunner interface {
	// Run executes a command and returns stdout, stderr, and error
	Run(ctx context.Context, name string, args ...string) (stdout string, stderr string, err error)
	// RunInDir executes a command in a specific directory
	RunInDir(ctx context.Context, dir string, name string, args ...string) (stdout string, stderr string, err error)
}

// commandRunner implements CommandRunner interface
type commandRunner struct{}

// NewCommandRunner creates a new command runner
func NewCommandRunner() CommandRunner {
	return &commandRunner{}
}

// Run executes a command and returns stdout, stderr, and error
func (r *commandRunner) Run(ctx context.Context, name string, args ...string) (string, string, error) {
	return r.RunInDir(ctx, "", name, args...)
}

// RunInDir executes a command in a specific directory
func (r *commandRunner) RunInDir(ctx context.Context, dir string, name string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}
