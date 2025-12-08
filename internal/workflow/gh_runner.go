package workflow

import (
	"context"
	"fmt"
)

// GhRunner abstracts gh CLI command execution for testing
type GhRunner interface {
	// PRCreate creates a new PR and returns the PR URL
	PRCreate(ctx context.Context, dir string, title, body, head string) (prURL string, err error)
	// PRView returns PR info as JSON
	PRView(ctx context.Context, dir string, jsonFields string, jqQuery string) (output string, err error)
	// PRChecks returns CI check status as JSON
	PRChecks(ctx context.Context, dir string, prNumber int, jsonFields string) (output string, err error)
}

// ghRunner implements GhRunner interface
type ghRunner struct {
	runner CommandRunner
}

// NewGhRunner creates a new gh runner
func NewGhRunner(runner CommandRunner) GhRunner {
	return &ghRunner{
		runner: runner,
	}
}

// PRCreate creates a new PR and returns the PR URL
func (g *ghRunner) PRCreate(ctx context.Context, dir string, title, body, head string) (string, error) {
	args := []string{"pr", "create", "--title", title, "--body", body, "--head", head}

	stdout, stderr, err := g.runner.RunInDir(ctx, dir, "gh", args...)
	if err != nil {
		return "", fmt.Errorf("failed to create PR: %w (stderr: %s)", err, stderr)
	}

	return stdout, nil
}

// PRView returns PR info as JSON
func (g *ghRunner) PRView(ctx context.Context, dir string, jsonFields string, jqQuery string) (string, error) {
	args := []string{"pr", "view", "--json", jsonFields, "-q", jqQuery}

	stdout, stderr, err := g.runner.RunInDir(ctx, dir, "gh", args...)
	if err != nil {
		return "", fmt.Errorf("failed to view PR: %w (stderr: %s)", err, stderr)
	}

	return stdout, nil
}

// PRChecks returns CI check status as JSON
func (g *ghRunner) PRChecks(ctx context.Context, dir string, prNumber int, jsonFields string) (string, error) {
	var args []string
	if prNumber > 0 {
		args = []string{"pr", "checks", fmt.Sprintf("%d", prNumber), "--json", jsonFields}
	} else {
		args = []string{"pr", "checks", "--json", jsonFields}
	}

	stdout, stderr, err := g.runner.RunInDir(ctx, dir, "gh", args...)
	if err != nil {
		return "", fmt.Errorf("failed to check PR status: %w (stderr: %s)", err, stderr)
	}

	return stdout, nil
}
