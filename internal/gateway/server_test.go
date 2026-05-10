package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServerWithAuth(t *testing.T) {
	config := ProxyConfig{
		AllowedOwner: "test-owner",
		AllowedRepo:  "test-repo",
	}
	ghAuth := NewGitHubAuthFromToken("test-token")

	server := NewServerWithAuth(config, ghAuth)

	require.NotNil(t, server)
	assert.NotNil(t, server.proxy)
	assert.NotNil(t, server.apiServer)
}

func TestNewServer_NoAuth(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("HOME", t.TempDir())

	config := ProxyConfig{
		AllowedOwner: "test-owner",
		AllowedRepo:  "test-repo",
	}

	_, err := NewServer(config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize GitHub auth")
}

func TestNewServer_WithEnvToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_test_server_token")

	config := ProxyConfig{
		AllowedOwner: "test-owner",
		AllowedRepo:  "test-repo",
	}

	server, err := NewServer(config)

	require.NoError(t, err)
	require.NotNil(t, server)
	assert.NotNil(t, server.proxy)
	assert.NotNil(t, server.apiServer)
}
