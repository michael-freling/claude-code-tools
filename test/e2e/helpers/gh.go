package helpers

import (
	"os/exec"
	"strings"
	"testing"
)

// RequireGH skips the test if gh is not available in PATH
func RequireGH(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("gh not found in PATH")
	}
}

// RequireGHAuth skips the test if not authenticated with gh
func RequireGHAuth(t *testing.T) {
	t.Helper()

	RequireGH(t)

	cmd := exec.Command("gh", "auth", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("gh not authenticated: %v: %s", err, string(output))
	}

	if !strings.Contains(string(output), "Logged in") {
		t.Skipf("gh not authenticated: %s", string(output))
	}
}

// GHVersion returns the gh version
func GHVersion(t *testing.T) string {
	t.Helper()

	cmd := exec.Command("gh", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to get gh version: %v", err)
	}

	return string(output)
}
