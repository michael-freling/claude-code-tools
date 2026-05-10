//go:build forge_e2e

package forge_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/michael-freling/claude-code-tools/internal/forge/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requireDockerAndToken skips the test if Docker or GITHUB_TOKEN is not available.
func requireDockerAndToken(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found in PATH")
	}

	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("Docker daemon not available")
	}

	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN not set -- set it for full e2e tests")
	}
}

// buildForgeImages builds the claude-forge binary and Docker images for e2e testing.
func buildForgeImages(t *testing.T) {
	t.Helper()
	projectRoot := findProjectRoot(t)

	// Build claude-forge binary for Linux amd64
	cmd := exec.Command("go", "build", "-o", filepath.Join(projectRoot, "docker/agent/claude-forge"), "./cmd/claude-forge/")
	cmd.Dir = projectRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build binary: %s", out)

	// Build gateway image
	cmd = exec.Command("docker", "build", "-t", "forge-e2e-gateway", "-f", "docker/gateway/Dockerfile", ".")
	cmd.Dir = projectRoot
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, "failed to build gateway image: %s", out)

	// Build agent image
	cmd = exec.Command("docker", "build", "-t", "forge-e2e-agent", "docker/agent/")
	cmd.Dir = projectRoot
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, "failed to build agent image: %s", out)
}

// findProjectRoot walks up from the current directory to find the go.mod file.
func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "could not find project root")
		dir = parent
	}
}

// dockerExec runs a command inside a running container and returns its output.
func dockerExec(t *testing.T, containerName string, args ...string) string {
	t.Helper()
	cmdArgs := append([]string{"exec", containerName}, args...)
	cmd := exec.Command("docker", cmdArgs...)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "docker exec %v failed: %s", args, output)
	return string(output)
}

func TestForgeStart_GatewayOperations(t *testing.T) {
	requireDockerAndToken(t)
	buildForgeImages(t)

	ctx := context.Background()
	dockerClient, err := container.NewClient()
	require.NoError(t, err)
	defer dockerClient.Close()

	networkName := "forge-e2e-start-net"
	agentName := "forge-e2e-start-agent"
	gatewayName := "forge-e2e-start-gateway"
	projectRoot := findProjectRoot(t)

	// Cache mount dir
	cacheDir := t.TempDir()
	npmCacheDir := filepath.Join(cacheDir, "npm")
	require.NoError(t, os.MkdirAll(npmCacheDir, 0o755))

	// Cleanup
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", agentName).Run()
		_ = dockerClient.StopContainer(ctx, gatewayName)
		_ = dockerClient.RemoveContainer(ctx, gatewayName)
		_ = dockerClient.RemoveNetwork(ctx, networkName)
	})

	// Create network
	_, err = dockerClient.CreateNetwork(ctx, networkName)
	require.NoError(t, err)

	// Start gateway with GITHUB_TOKEN
	_, err = dockerClient.StartGateway(ctx, container.GatewayOptions{
		Name:        gatewayName,
		Image:       "forge-e2e-gateway",
		NetworkName: networkName,
		Owner:       "michael-freling",
		Repo:        "claude-code-tools",
		Env: map[string]string{
			"GITHUB_TOKEN": os.Getenv("GITHUB_TOKEN"),
		},
	})
	require.NoError(t, err)

	// Start agent with sleep (using docker run directly to avoid "claude" prefix
	// that StartAgent prepends to the command)
	agentCmd := exec.Command("docker", "run", "-d",
		"--name", agentName,
		"--network", networkName,
		"-v", projectRoot+":/work",
		"-v", npmCacheDir+":/home/user/.npm",
		"-e", "HOME=/home/user",
		"-e", "FORGE_PROJECT_OWNER=michael-freling",
		"-e", "FORGE_PROJECT_REPO=claude-code-tools",
		"-w", "/work",
		"forge-e2e-agent",
		"sleep", "120",
	)
	output, err := agentCmd.CombinedOutput()
	require.NoError(t, err, "failed to start agent: %s", output)

	// Wait for gateway to be ready
	time.Sleep(3 * time.Second)

	// Test 1: forge-gh can reach gateway and get repo info via API
	t.Run("forge-gh repo view", func(t *testing.T) {
		out := dockerExec(t, agentName, "forge-gh", "repo", "view")
		require.NotEmpty(t, out)
		assert.Contains(t, out, "claude-code-tools")
	})

	// Test 2: git ls-remote through gateway proxy
	t.Run("git ls-remote through proxy", func(t *testing.T) {
		// Configure git to use the gateway as HTTP proxy for github.com
		out := dockerExec(t, agentName, "git",
			"-c", "http.https://github.com.proxy=http://gateway:8080",
			"ls-remote", "--heads",
			"https://github.com/michael-freling/claude-code-tools.git",
		)
		require.NotEmpty(t, out)
		assert.Contains(t, out, "refs/heads/main")
	})

	// Test 3: Cache mount persistence
	t.Run("cache mount persistence", func(t *testing.T) {
		dockerExec(t, agentName, "sh", "-c", "echo e2e-test-data > /home/user/.npm/e2e-marker.txt")

		markerPath := filepath.Join(npmCacheDir, "e2e-marker.txt")
		content, err := os.ReadFile(markerPath)
		require.NoError(t, err, "file written in container should be visible on host")
		assert.Equal(t, "e2e-test-data\n", string(content))
	})
}
