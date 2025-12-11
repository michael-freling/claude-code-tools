package hooks

import (
	"context"

	"github.com/michael-freling/claude-code-tools/internal/command"
)

// GitHelper provides methods to interact with git commands.
type GitHelper interface {
	// GetCurrentBranch returns the name of the current git branch.
	GetCurrentBranch() (string, error)
}

// realGitHelper implements GitHelper as an adapter over command.GitRunner.
type realGitHelper struct {
	runner command.GitRunner
}

// NewGitHelper creates a new GitHelper instance using command.GitRunner.
func NewGitHelper() GitHelper {
	return NewGitHelperWithRunner(command.NewGitRunner(command.NewRunner()))
}

// NewGitHelperWithRunner creates a new GitHelper with a custom runner for testing.
func NewGitHelperWithRunner(runner command.GitRunner) GitHelper {
	return &realGitHelper{
		runner: runner,
	}
}

// GetCurrentBranch returns the current git branch name.
func (g *realGitHelper) GetCurrentBranch() (string, error) {
	return g.runner.GetCurrentBranch(context.Background(), "")
}
