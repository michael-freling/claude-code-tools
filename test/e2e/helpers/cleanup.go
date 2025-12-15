package helpers

import (
	"os"
	"testing"
)

// CleanupDir safely removes a directory
func CleanupDir(t *testing.T, dir string) {
	t.Helper()

	if err := os.RemoveAll(dir); err != nil {
		t.Errorf("failed to cleanup directory %s: %v", dir, err)
	}
}

// CleanupOnFailure runs cleanup only if the test failed.
// This is useful when you want to preserve test artifacts for debugging
// when a test fails, but clean them up when tests pass successfully.
func CleanupOnFailure(t *testing.T, cleanup func()) {
	t.Helper()

	t.Cleanup(func() {
		if t.Failed() {
			cleanup()
		}
	})
}
