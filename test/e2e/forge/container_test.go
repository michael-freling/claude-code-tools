//go:build forge_e2e

package forge_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requireDocker skips the test if Docker is not available.
func requireDocker(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found in PATH, skipping container test")
	}

	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Skip("docker daemon not available, skipping container test")
	}
}

func TestContainer_CacheMountPersistence(t *testing.T) {
	requireDocker(t)

	tmpDir := t.TempDir()
	npmDir := filepath.Join(tmpDir, "npm")
	require.NoError(t, os.MkdirAll(npmDir, 0o755))

	cmd := exec.Command("docker", "run", "--rm",
		"-v", npmDir+":/cache",
		"ubuntu:24.04",
		"bash", "-c", "echo cached-data > /cache/test-file.txt",
	)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "docker run failed: %s", string(output))

	content, err := os.ReadFile(filepath.Join(npmDir, "test-file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "cached-data\n", string(content))
}

func TestContainer_FileOwnership(t *testing.T) {
	requireDocker(t)

	if runtime.GOOS != "linux" {
		t.Skip("file ownership verification requires Linux")
	}

	hostUID := os.Getuid()
	hostGID := os.Getgid()

	mountDir := t.TempDir()

	uidStr := strconv.Itoa(hostUID)
	gidStr := strconv.Itoa(hostGID)

	// Use a shell approach that works regardless of whether UID/GID
	// already exist in the container image (e.g. ubuntu:24.04 has UID 1000).
	// Create the file as root, then chown to the host UID/GID.
	script := "echo hello > /work/owned-file.txt && chown " + uidStr + ":" + gidStr + " /work/owned-file.txt"

	cmd := exec.Command("docker", "run", "--rm",
		"-v", mountDir+":/work",
		"ubuntu:24.04",
		"bash", "-c", script,
	)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "docker run failed: %s", string(output))

	filePath := filepath.Join(mountDir, "owned-file.txt")

	// Verify the file exists and is readable
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "hello\n", string(content))

	// Verify UID/GID match using syscall.Stat_t
	var stat syscall.Stat_t
	err = syscall.Stat(filePath, &stat)
	require.NoError(t, err)
	assert.Equal(t, uint32(hostUID), stat.Uid, "file UID should match host UID")
	assert.Equal(t, uint32(hostGID), stat.Gid, "file GID should match host GID")

	// Verify we can modify the file (proves ownership is correct)
	err = os.WriteFile(filePath, []byte("modified\n"), 0o644)
	require.NoError(t, err, "should be able to modify the file owned by host user")

	modified, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "modified\n", string(modified))
}

func TestContainer_MultipleMountsIsolation(t *testing.T) {
	requireDocker(t)

	tmpDir := t.TempDir()
	npmDir := filepath.Join(tmpDir, "npm")
	pipDir := filepath.Join(tmpDir, "pip")
	require.NoError(t, os.MkdirAll(npmDir, 0o755))
	require.NoError(t, os.MkdirAll(pipDir, 0o755))

	cmd := exec.Command("docker", "run", "--rm",
		"-v", npmDir+":/cache-npm",
		"-v", pipDir+":/cache-pip",
		"ubuntu:24.04",
		"bash", "-c", "echo npm-content > /cache-npm/package.txt && echo pip-content > /cache-pip/requirements.txt",
	)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "docker run failed: %s", string(output))

	npmContent, err := os.ReadFile(filepath.Join(npmDir, "package.txt"))
	require.NoError(t, err)
	assert.Equal(t, "npm-content\n", string(npmContent))

	pipContent, err := os.ReadFile(filepath.Join(pipDir, "requirements.txt"))
	require.NoError(t, err)
	assert.Equal(t, "pip-content\n", string(pipContent))
}
