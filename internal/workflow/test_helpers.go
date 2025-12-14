//go:build e2e

package workflow

import (
	"fmt"
	"time"

	"github.com/michael-freling/claude-code-tools/internal/command"
)

// NewTestOrchestrator creates an orchestrator for E2E testing with a custom CI checker factory.
// This allows E2E tests to provide a mock CI checker while using real Claude CLI and other components.
func NewTestOrchestrator(config *Config, ciCheckerFactory func(workingDir string, checkInterval time.Duration, commandTimeout time.Duration) CIChecker) (*Orchestrator, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.BaseDir == "" {
		return nil, fmt.Errorf("baseDir cannot be empty")
	}

	promptGen, err := NewPromptGenerator()
	if err != nil {
		return nil, err
	}

	logger := NewLogger(config.LogLevel)
	executor := NewClaudeExecutorWithPath(config.ClaudePath, logger)
	stateManager := NewStateManager(config.BaseDir)
	parser := NewOutputParser()
	worktreeManager := NewWorktreeManager(config.BaseDir)
	cmdRunner := command.NewRunner()
	ghRunner := command.NewGhRunner(cmdRunner)
	gitRunner := command.NewGitRunner(cmdRunner)
	splitManager := NewPRSplitManager(gitRunner, ghRunner)

	return &Orchestrator{
		stateManager:     stateManager,
		executor:         executor,
		promptGenerator:  promptGen,
		parser:           parser,
		config:           config,
		confirmFunc:      defaultConfirmFunc,
		worktreeManager:  worktreeManager,
		logger:           logger,
		ghRunner:         ghRunner,
		gitRunner:        gitRunner,
		splitManager:     splitManager,
		ciCheckerFactory: ciCheckerFactory,
	}, nil
}
