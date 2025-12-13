package command

import (
	"context"
	"fmt"
	"strings"
)

// Commit represents a git commit with its hash and subject
type Commit struct {
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
}

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
	// GetCommits returns list of commits from base branch to HEAD
	GetCommits(ctx context.Context, dir string, base string) ([]Commit, error)
	// CherryPick cherry-picks a specific commit
	CherryPick(ctx context.Context, dir string, commitHash string) error
	// CreateBranch creates a new branch from a base branch
	CreateBranch(ctx context.Context, dir string, branchName string, baseBranch string) error
	// CheckoutBranch checks out an existing branch
	CheckoutBranch(ctx context.Context, dir string, branchName string) error
	// DeleteBranch deletes a local branch
	DeleteBranch(ctx context.Context, dir string, branchName string, force bool) error
	// DeleteRemoteBranch deletes a remote branch
	DeleteRemoteBranch(ctx context.Context, dir string, branchName string) error
	// CommitEmpty creates an empty commit
	CommitEmpty(ctx context.Context, dir string, message string) error
	// CheckoutFiles checks out specific files from a source branch
	CheckoutFiles(ctx context.Context, dir string, sourceBranch string, files []string) error
	// CommitAll stages all changes and creates a commit
	CommitAll(ctx context.Context, dir string, message string) error
	// GetDiffStat returns the diff stat output for the given base branch
	GetDiffStat(ctx context.Context, dir string, base string) (string, error)
}

type gitRunner struct {
	runner Runner
}

// NewGitRunner creates a new GitRunner instance
func NewGitRunner(runner Runner) GitRunner {
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

// GetCommits returns list of commits from base branch to HEAD
func (g *gitRunner) GetCommits(ctx context.Context, dir string, base string) ([]Commit, error) {
	if base == "" {
		return nil, fmt.Errorf("base branch cannot be empty")
	}

	stdout, stderr, err := g.runner.RunInDir(ctx, dir, "git", "log", fmt.Sprintf("%s..HEAD", base), "--format=%H|%s", "--reverse")
	if err != nil {
		return nil, fmt.Errorf("failed to get commits from %s: %w (stderr: %s)", base, err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []Commit{}, nil
	}

	commits := make([]Commit, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid commit format: %s", line)
		}
		commits = append(commits, Commit{
			Hash:    parts[0],
			Subject: parts[1],
		})
	}

	return commits, nil
}

// CherryPick cherry-picks a specific commit
func (g *gitRunner) CherryPick(ctx context.Context, dir string, commitHash string) error {
	if commitHash == "" {
		return fmt.Errorf("commit hash cannot be empty")
	}

	_, stderr, err := g.runner.RunInDir(ctx, dir, "git", "cherry-pick", commitHash)
	if err != nil {
		return fmt.Errorf("failed to cherry-pick commit %s: %w (stderr: %s)", commitHash, err, stderr)
	}

	return nil
}

// CreateBranch creates a new branch from a base branch
func (g *gitRunner) CreateBranch(ctx context.Context, dir string, branchName string, baseBranch string) error {
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}
	if baseBranch == "" {
		return fmt.Errorf("base branch cannot be empty")
	}

	_, stderr, err := g.runner.RunInDir(ctx, dir, "git", "checkout", "-b", branchName, baseBranch)
	if err != nil {
		return fmt.Errorf("failed to create branch %s from %s: %w (stderr: %s)", branchName, baseBranch, err, stderr)
	}

	return nil
}

// CheckoutBranch checks out an existing branch
func (g *gitRunner) CheckoutBranch(ctx context.Context, dir string, branchName string) error {
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	_, stderr, err := g.runner.RunInDir(ctx, dir, "git", "checkout", branchName)
	if err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w (stderr: %s)", branchName, err, stderr)
	}

	return nil
}

// DeleteBranch deletes a local branch
func (g *gitRunner) DeleteBranch(ctx context.Context, dir string, branchName string, force bool) error {
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	flag := "-d"
	if force {
		flag = "-D"
	}

	_, stderr, err := g.runner.RunInDir(ctx, dir, "git", "branch", flag, branchName)
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %w (stderr: %s)", branchName, err, stderr)
	}

	return nil
}

// DeleteRemoteBranch deletes a remote branch
func (g *gitRunner) DeleteRemoteBranch(ctx context.Context, dir string, branchName string) error {
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	_, stderr, err := g.runner.RunInDir(ctx, dir, "git", "push", "origin", "--delete", branchName)
	if err != nil {
		return fmt.Errorf("failed to delete remote branch %s: %w (stderr: %s)", branchName, err, stderr)
	}

	return nil
}

// CommitEmpty creates an empty commit
func (g *gitRunner) CommitEmpty(ctx context.Context, dir string, message string) error {
	if message == "" {
		return fmt.Errorf("commit message cannot be empty")
	}

	_, stderr, err := g.runner.RunInDir(ctx, dir, "git", "commit", "--allow-empty", "-m", message)
	if err != nil {
		return fmt.Errorf("failed to create empty commit: %w (stderr: %s)", err, stderr)
	}

	return nil
}

// CheckoutFiles checks out specific files from a source branch
func (g *gitRunner) CheckoutFiles(ctx context.Context, dir string, sourceBranch string, files []string) error {
	if sourceBranch == "" {
		return fmt.Errorf("source branch cannot be empty")
	}
	if len(files) == 0 {
		return fmt.Errorf("files list cannot be empty")
	}

	args := []string{"checkout", sourceBranch, "--"}
	args = append(args, files...)

	_, stderr, err := g.runner.RunInDir(ctx, dir, "git", args...)
	if err != nil {
		return fmt.Errorf("failed to checkout files from %s: %w (stderr: %s)", sourceBranch, err, stderr)
	}

	return nil
}

// CommitAll stages all changes and creates a commit
func (g *gitRunner) CommitAll(ctx context.Context, dir string, message string) error {
	if message == "" {
		return fmt.Errorf("commit message cannot be empty")
	}

	_, stderr, err := g.runner.RunInDir(ctx, dir, "git", "add", "-A")
	if err != nil {
		return fmt.Errorf("failed to stage changes: %w (stderr: %s)", err, stderr)
	}

	_, stderr, err = g.runner.RunInDir(ctx, dir, "git", "commit", "-m", message)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w (stderr: %s)", err, stderr)
	}

	return nil
}

// GetDiffStat returns the diff stat output for the given base branch
func (g *gitRunner) GetDiffStat(ctx context.Context, dir string, base string) (string, error) {
	if base == "" {
		return "", fmt.Errorf("base branch cannot be empty")
	}

	stdout, stderr, err := g.runner.RunInDir(ctx, dir, "git", "diff", "--stat", base)
	if err != nil {
		return "", fmt.Errorf("failed to get diff stat from %s: %w (stderr: %s)", base, err, stderr)
	}

	return stdout, nil
}
