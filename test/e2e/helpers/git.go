package helpers

import (
	"os/exec"
	"testing"
)

// RequireGit skips the test if git is not available in PATH
func RequireGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
}

// GitVersion returns the git version
func GitVersion(t *testing.T) string {
	t.Helper()

	cmd := exec.Command("git", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to get git version: %v", err)
	}

	return string(output)
}
