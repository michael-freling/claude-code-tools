package command

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
)

// Runner abstracts command execution for testability
type Runner interface {
	// Run executes a command and returns stdout, stderr, and error
	Run(ctx context.Context, name string, args ...string) (stdout string, stderr string, err error)
	// RunInDir executes a command in a specific directory
	RunInDir(ctx context.Context, dir string, name string, args ...string) (stdout string, stderr string, err error)
}

// realRunner implements Runner interface
type realRunner struct{}

// NewRunner creates a new command runner
func NewRunner() Runner {
	return &realRunner{}
}

// Run executes a command and returns stdout, stderr, and error
func (r *realRunner) Run(ctx context.Context, name string, args ...string) (string, string, error) {
	return r.RunInDir(ctx, "", name, args...)
}

// RunInDir executes a command in a specific directory
func (r *realRunner) RunInDir(ctx context.Context, dir string, name string, args ...string) (string, string, error) {
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
