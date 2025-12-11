package hooks

import (
	"context"

	"github.com/michael-freling/claude-code-tools/internal/command"
)

// GhHelper provides methods to interact with GitHub CLI commands.
type GhHelper interface {
	// GetPRBaseBranch returns the base branch name for a pull request.
	GetPRBaseBranch(prNumber string) (string, error)
}

// realGhHelper implements GhHelper as an adapter over command.GhRunner.
type realGhHelper struct {
	runner command.GhRunner
}

// NewGhHelper creates a new GhHelper instance using command.GhRunner.
func NewGhHelper() GhHelper {
	return NewGhHelperWithRunner(command.NewGhRunner(command.NewRunner()))
}

// NewGhHelperWithRunner creates a new GhHelper with a custom runner for testing.
func NewGhHelperWithRunner(runner command.GhRunner) GhHelper {
	return &realGhHelper{
		runner: runner,
	}
}

// GetPRBaseBranch returns the base branch name for the specified PR number.
func (g *realGhHelper) GetPRBaseBranch(prNumber string) (string, error) {
	return g.runner.GetPRBaseBranch(context.Background(), "", prNumber)
}
