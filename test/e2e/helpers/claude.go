package helpers

import (
	"os/exec"
	"testing"
)

// RequireClaude skips the test if claude is not available in PATH
func RequireClaude(t *testing.T) {
	t.Helper()

	if !IsCLIAvailable() {
		t.Skip("claude not found in PATH")
	}
}

// ClaudeVersion returns the claude version
func ClaudeVersion(t *testing.T) string {
	t.Helper()

	cmd := exec.Command("claude", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to get claude version: %v", err)
	}

	return string(output)
}

// IsCLIAvailable checks if claude is available without skipping
func IsCLIAvailable() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}
