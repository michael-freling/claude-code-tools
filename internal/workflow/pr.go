package workflow

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// PRManager handles PR creation and management
type PRManager interface {
	// CreatePR creates a new PR for the current branch and returns the PR number
	CreatePR(ctx context.Context, title, body string) (int, error)
	// GetCurrentBranchPR returns the PR number for the current branch, or 0 if none exists
	GetCurrentBranchPR(ctx context.Context) (int, error)
	// EnsurePR ensures a PR exists for the current branch, creating one if needed
	EnsurePR(ctx context.Context, title, body string) (int, error)
	// PushBranch pushes the current branch to origin
	PushBranch(ctx context.Context) error
}

// prManager implements PRManager interface
type prManager struct {
	workingDir string
}

// NewPRManager creates a new PR manager
func NewPRManager(workingDir string) PRManager {
	return &prManager{
		workingDir: workingDir,
	}
}

// CreatePR creates a new PR for the current branch
func (p *prManager) CreatePR(ctx context.Context, title, body string) (int, error) {
	// Get current branch name to use with --head flag
	branchName, err := p.getCurrentBranch(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get current branch: %w", err)
	}

	cmd := exec.CommandContext(ctx, "gh", "pr", "create", "--title", title, "--body", body, "--head", branchName)
	if p.workingDir != "" {
		cmd.Dir = p.workingDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return 0, fmt.Errorf("failed to create PR: %w (stderr: %s)", err, stderr.String())
	}

	// Parse PR URL from output to extract PR number
	// Output format: https://github.com/owner/repo/pull/123
	prURL := strings.TrimSpace(stdout.String())
	prNumber, err := extractPRNumberFromURL(prURL)
	if err != nil {
		return 0, fmt.Errorf("failed to extract PR number from URL %q: %w", prURL, err)
	}

	return prNumber, nil
}

// getCurrentBranch returns the current branch name
func (p *prManager) getCurrentBranch(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if p.workingDir != "" {
		cmd.Dir = p.workingDir
	}

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetCurrentBranchPR returns the PR number for the current branch
func (p *prManager) GetCurrentBranchPR(ctx context.Context) (int, error) {
	cmd := exec.CommandContext(ctx, "gh", "pr", "view", "--json", "number", "-q", ".number")
	if p.workingDir != "" {
		cmd.Dir = p.workingDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if the error is "no pull requests found"
		if strings.Contains(stderr.String(), "no pull requests found") {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get PR for current branch: %w (stderr: %s)", err, stderr.String())
	}

	prNumberStr := strings.TrimSpace(stdout.String())
	if prNumberStr == "" {
		return 0, nil
	}

	prNumber, err := strconv.Atoi(prNumberStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse PR number %q: %w", prNumberStr, err)
	}

	return prNumber, nil
}

// EnsurePR ensures a PR exists for the current branch, creating one if needed
func (p *prManager) EnsurePR(ctx context.Context, title, body string) (int, error) {
	// First check if a PR already exists
	prNumber, err := p.GetCurrentBranchPR(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to check for existing PR: %w", err)
	}

	if prNumber > 0 {
		return prNumber, nil
	}

	// No PR exists, create one
	return p.CreatePR(ctx, title, body)
}

// PushBranch pushes the current branch to origin with upstream tracking
func (p *prManager) PushBranch(ctx context.Context) error {
	branchName, err := p.getCurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Push with upstream tracking and force to ensure it's up to date
	pushCmd := exec.CommandContext(ctx, "git", "push", "-u", "origin", branchName)
	if p.workingDir != "" {
		pushCmd.Dir = p.workingDir
	}

	var stderr bytes.Buffer
	pushCmd.Stderr = &stderr

	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("failed to push branch: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// extractPRNumberFromURL extracts PR number from a GitHub PR URL
func extractPRNumberFromURL(url string) (int, error) {
	// Match URLs like https://github.com/owner/repo/pull/123
	re := regexp.MustCompile(`/pull/(\d+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) < 2 {
		return 0, fmt.Errorf("URL does not contain PR number")
	}

	return strconv.Atoi(matches[1])
}
