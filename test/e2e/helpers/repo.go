package helpers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TempRepo represents a temporary Git repository for testing
type TempRepo struct {
	Dir string
	t   *testing.T
}

// NewTempRepo creates a new temporary Git repository for testing
func NewTempRepo(t *testing.T) *TempRepo {
	t.Helper()

	dir, err := os.MkdirTemp("", "e2e-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	repo := &TempRepo{
		Dir: dir,
		t:   t,
	}

	if _, err := repo.RunGit("init"); err != nil {
		_ = os.RemoveAll(dir) // Ignore cleanup error, already failing
		t.Fatalf("failed to initialize git repo: %v", err)
	}

	if _, err := repo.RunGit("config", "user.email", "test@test.com"); err != nil {
		_ = os.RemoveAll(dir) // Ignore cleanup error, already failing
		t.Fatalf("failed to configure git user.email: %v", err)
	}

	if _, err := repo.RunGit("config", "user.name", "Test User"); err != nil {
		_ = os.RemoveAll(dir) // Ignore cleanup error, already failing
		t.Fatalf("failed to configure git user.name: %v", err)
	}

	t.Cleanup(func() {
		repo.Cleanup()
	})

	return repo
}

// Cleanup removes the temporary directory
func (r *TempRepo) Cleanup() {
	r.t.Helper()

	if err := os.RemoveAll(r.Dir); err != nil {
		r.t.Errorf("failed to cleanup temp repo: %v", err)
	}
}

// CreateFile creates a file with the given content
func (r *TempRepo) CreateFile(path, content string) error {
	r.t.Helper()

	fullPath := filepath.Join(r.Dir, path)

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}

// Commit stages all files and creates a commit
func (r *TempRepo) Commit(message string) error {
	r.t.Helper()

	if _, err := r.RunGit("add", "-A"); err != nil {
		return fmt.Errorf("failed to stage files: %w", err)
	}

	if _, err := r.RunGit("commit", "-m", message); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// CreateBranch creates and checks out a new branch
func (r *TempRepo) CreateBranch(name string) error {
	r.t.Helper()

	if _, err := r.RunGit("checkout", "-b", name); err != nil {
		return fmt.Errorf("failed to create branch %s: %w", name, err)
	}

	return nil
}

// RunGit runs a git command in the repository
func (r *TempRepo) RunGit(args ...string) (string, error) {
	r.t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed: %w: %s", err, string(output))
	}

	return string(output), nil
}

// CloneRepo clones a git repository to a temporary directory
func CloneRepo(t *testing.T, url string) *TempRepo {
	t.Helper()

	dir, err := os.MkdirTemp("", "e2e-clone-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	repo := &TempRepo{
		Dir: dir,
		t:   t,
	}

	cmd := exec.Command("git", "clone", url, dir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(dir)
		t.Fatalf("failed to clone repo: %v: %s", err, string(output))
	}

	if _, err := repo.RunGit("config", "user.email", "test@test.com"); err != nil {
		_ = os.RemoveAll(dir)
		t.Fatalf("failed to configure git user.email: %v", err)
	}

	if _, err := repo.RunGit("config", "user.name", "Test User"); err != nil {
		_ = os.RemoveAll(dir)
		t.Fatalf("failed to configure git user.name: %v", err)
	}

	t.Cleanup(func() {
		repo.Cleanup()
	})

	return repo
}
