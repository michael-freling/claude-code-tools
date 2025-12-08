package workflow

import (
	"context"
	"fmt"
	"strings"
)

// GitRunner abstracts git command execution
type GitRunner interface {
	// GetCurrentBranch returns the current git branch name
	GetCurrentBranch(ctx context.Context, dir string) (string, error)
	// Push pushes a branch to origin with upstream tracking
	Push(ctx context.Context, dir string, branch string) error
	// WorktreeAdd creates a new git worktree
	WorktreeAdd(ctx context.Context, dir string, path string, branch string) error
	// WorktreeRemove removes a git worktree
	WorktreeRemove(ctx context.Context, dir string, path string) error
}

type gitRunner struct {
	runner CommandRunner
}

// NewGitRunner creates a new GitRunner instance
func NewGitRunner(runner CommandRunner) GitRunner {
	return &gitRunner{
		runner: runner,
	}
}

// GetCurrentBranch returns the current git branch name
func (g *gitRunner) GetCurrentBranch(ctx context.Context, dir string) (string, error) {
	stdout, _, err := g.runner.RunInDir(ctx, dir, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(stdout), nil
}

// Push pushes a branch to origin with upstream tracking
func (g *gitRunner) Push(ctx context.Context, dir string, branch string) error {
	if branch == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	_, stderr, err := g.runner.RunInDir(ctx, dir, "git", "push", "-u", "origin", branch)
	if err != nil {
		return fmt.Errorf("failed to push branch %s: %w (stderr: %s)", branch, err, stderr)
	}

	return nil
}

// WorktreeAdd creates a new git worktree
func (g *gitRunner) WorktreeAdd(ctx context.Context, dir string, path string, branch string) error {
	if path == "" {
		return fmt.Errorf("worktree path cannot be empty")
	}
	if branch == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	_, stderr, err := g.runner.RunInDir(ctx, dir, "git", "worktree", "add", path, "-b", branch)
	if err != nil {
		if strings.Contains(stderr, "already exists") {
			return fmt.Errorf("branch %s already exists", branch)
		}
		return fmt.Errorf("failed to create worktree at %s: %w (stderr: %s)", path, err, stderr)
	}

	return nil
}

// WorktreeRemove removes a git worktree
func (g *gitRunner) WorktreeRemove(ctx context.Context, dir string, path string) error {
	if path == "" {
		return fmt.Errorf("worktree path cannot be empty")
	}

	_, stderr, err := g.runner.RunInDir(ctx, dir, "git", "worktree", "remove", path)
	if err != nil {
		return fmt.Errorf("failed to remove worktree at %s: %w (stderr: %s)", path, err, stderr)
	}

	return nil
}
