package workflow

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var prNumberRegex = regexp.MustCompile(`/pull/(\d+)`)

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
	gitRunner  GitRunner
	ghRunner   GhRunner
}

// NewPRManager creates a new PR manager
func NewPRManager(workingDir string) PRManager {
	cmdRunner := NewCommandRunner()
	return &prManager{
		workingDir: workingDir,
		gitRunner:  NewGitRunner(cmdRunner),
		ghRunner:   NewGhRunner(cmdRunner),
	}
}

// NewPRManagerWithRunners creates a new PR manager with injected runners for testing
func NewPRManagerWithRunners(workingDir string, gitRunner GitRunner, ghRunner GhRunner) PRManager {
	return &prManager{
		workingDir: workingDir,
		gitRunner:  gitRunner,
		ghRunner:   ghRunner,
	}
}

// CreatePR creates a new PR for the current branch
func (p *prManager) CreatePR(ctx context.Context, title, body string) (int, error) {
	branchName, err := p.getCurrentBranch(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get current branch: %w", err)
	}

	prURL, err := p.ghRunner.PRCreate(ctx, p.workingDir, title, body, branchName)
	if err != nil {
		return 0, err
	}

	prNumber, err := extractPRNumberFromURL(strings.TrimSpace(prURL))
	if err != nil {
		return 0, fmt.Errorf("failed to extract PR number from URL %q: %w", prURL, err)
	}

	return prNumber, nil
}

// getCurrentBranch returns the current branch name
func (p *prManager) getCurrentBranch(ctx context.Context) (string, error) {
	return p.gitRunner.GetCurrentBranch(ctx, p.workingDir)
}

// GetCurrentBranchPR returns the PR number for the current branch
func (p *prManager) GetCurrentBranchPR(ctx context.Context) (int, error) {
	prNumberStr, err := p.ghRunner.PRView(ctx, p.workingDir, "number", ".number")
	if err != nil {
		if strings.Contains(err.Error(), "no pull requests found") {
			return 0, nil
		}
		return 0, err
	}

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

	return p.gitRunner.Push(ctx, p.workingDir, branchName)
}

// extractPRNumberFromURL extracts PR number from a GitHub PR URL
func extractPRNumberFromURL(url string) (int, error) {
	// Match URLs like https://github.com/owner/repo/pull/123
	matches := prNumberRegex.FindStringSubmatch(url)
	if len(matches) < 2 {
		return 0, fmt.Errorf("URL does not contain PR number")
	}

	return strconv.Atoi(matches[1])
}
