package workflow

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/michael-freling/claude-code-tools/internal/command"
)

// WorktreeManager handles git worktree operations
type WorktreeManager interface {
	CreateWorktree(workflowName string) (string, error)
	WorktreeExists(path string) bool
	DeleteWorktree(path string) error
}

// worktreeManager implements WorktreeManager interface
type worktreeManager struct {
	baseDir   string
	gitRunner command.GitRunner
}

// NewWorktreeManager creates a new worktree manager
func NewWorktreeManager(baseDir string) WorktreeManager {
	cmdRunner := command.NewRunner()
	return &worktreeManager{
		baseDir:   baseDir,
		gitRunner: command.NewGitRunner(cmdRunner),
	}
}

// NewWorktreeManagerWithRunner creates a new worktree manager with a custom GitRunner
func NewWorktreeManagerWithRunner(baseDir string, gitRunner command.GitRunner) WorktreeManager {
	return &worktreeManager{
		baseDir:   baseDir,
		gitRunner: gitRunner,
	}
}

// CreateWorktree creates a new git worktree for the workflow
func (w *worktreeManager) CreateWorktree(workflowName string) (string, error) {
	if workflowName == "" {
		return "", fmt.Errorf("workflow name cannot be empty")
	}

	worktreesDir := filepath.Join(w.baseDir, "..", "worktrees")
	worktreePath := filepath.Join(worktreesDir, workflowName)

	absWorktreePath, err := filepath.Abs(worktreePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	if w.WorktreeExists(absWorktreePath) {
		return absWorktreePath, nil
	}

	if _, err := os.Stat(worktreesDir); os.IsNotExist(err) {
		if err := os.MkdirAll(worktreesDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create worktrees directory: %w", err)
		}
	}

	branchName := fmt.Sprintf("workflow/%s", workflowName)

	ctx := context.Background()
	if err := w.gitRunner.WorktreeAdd(ctx, w.baseDir, absWorktreePath, branchName); err != nil {
		return "", err
	}

	// Best effort: set up pre-commit hooks (non-fatal)
	if err := w.setupPreCommitHooks(absWorktreePath); err != nil {
		// Silently ignore errors - pre-commit setup is optional
		_ = err
	}

	return absWorktreePath, nil
}

// WorktreeExists checks if a worktree exists at the given path
func (w *worktreeManager) WorktreeExists(path string) bool {
	if path == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if !info.IsDir() {
		return false
	}

	gitDir := filepath.Join(path, ".git")
	_, err = os.Stat(gitDir)
	return err == nil
}

// DeleteWorktree removes a git worktree
func (w *worktreeManager) DeleteWorktree(path string) error {
	if path == "" {
		return fmt.Errorf("worktree path cannot be empty")
	}

	if !w.WorktreeExists(path) {
		return nil
	}

	ctx := context.Background()
	if err := w.gitRunner.WorktreeRemove(ctx, w.baseDir, path); err != nil {
		return err
	}

	return nil
}

// setupPreCommitHooks installs pre-commit hooks in the worktree if pre-commit is available
func (w *worktreeManager) setupPreCommitHooks(worktreePath string) error {
	configPath := filepath.Join(worktreePath, ".pre-commit-config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	}

	if _, err := exec.LookPath("pre-commit"); err != nil {
		return nil
	}

	ctx := context.Background()

	cmd := exec.CommandContext(ctx, "pre-commit", "install")
	cmd.Dir = worktreePath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run pre-commit install: %w", err)
	}

	cmd = exec.CommandContext(ctx, "pre-commit", "install", "--hook-type", "pre-push")
	cmd.Dir = worktreePath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run pre-commit install --hook-type pre-push: %w", err)
	}

	return nil
}
