//go:build forge_e2e

package forge_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/michael-freling/claude-code-tools/internal/gateway"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startMockGitServer creates a mock git server that accepts all requests.
func startMockGitServer(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("mock-git-response"))
	}))
	t.Cleanup(server.Close)
	return server
}

// startProxyServer creates a gateway Proxy backed by the given mock git URL.
func startProxyServer(t *testing.T, mockGitURL string) string {
	t.Helper()

	config := gateway.ProxyConfig{
		AllowedOwner: "test-owner",
		AllowedRepo:  "test-repo",
	}
	ghAuth := gateway.NewGitHubAuthFromToken("test-github-token")
	proxy := gateway.NewTestProxy(config, ghAuth, mockGitURL)

	server := httptest.NewServer(proxy)
	t.Cleanup(server.Close)
	return server.URL
}

func TestProxy_ReadAccess_AnyRepo(t *testing.T) {
	mockGit := startMockGitServer(t)
	proxyURL := startProxyServer(t, mockGit.URL)

	resp, err := http.Get(proxyURL + "/github.com/any-owner/any-repo.git/info/refs?service=git-upload-pack")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProxy_WriteAccess_DeniedForWrongRepo(t *testing.T) {
	mockGit := startMockGitServer(t)
	proxyURL := startProxyServer(t, mockGit.URL)

	resp, err := http.Post(proxyURL+"/github.com/other-owner/other-repo.git/git-receive-pack", "", strings.NewReader(""))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestProxy_WriteAccess_AllowedForProjectRepo(t *testing.T) {
	mockGit := startMockGitServer(t)
	proxyURL := startProxyServer(t, mockGit.URL)

	resp, err := http.Post(proxyURL+"/github.com/test-owner/test-repo.git/git-receive-pack", "", strings.NewReader(""))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProxy_WriteInfoRefs_DeniedForWrongRepo(t *testing.T) {
	mockGit := startMockGitServer(t)
	proxyURL := startProxyServer(t, mockGit.URL)

	resp, err := http.Get(proxyURL + "/github.com/other-owner/other-repo.git/info/refs?service=git-receive-pack")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestProxy_WriteInfoRefs_AllowedForProjectRepo(t *testing.T) {
	mockGit := startMockGitServer(t)
	proxyURL := startProxyServer(t, mockGit.URL)

	resp, err := http.Get(proxyURL + "/github.com/test-owner/test-repo.git/info/refs?service=git-receive-pack")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProxy_AuthHeaderForwarded(t *testing.T) {
	var capturedAuth string
	mockGit := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(mockGit.Close)

	proxyURL := startProxyServer(t, mockGit.URL)

	resp, err := http.Get(proxyURL + "/github.com/any-owner/any-repo.git/info/refs?service=git-upload-pack")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	_ = body

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, capturedAuth, "Authorization header should be forwarded")
	assert.True(t, strings.HasPrefix(capturedAuth, "Basic "), "Authorization header should start with 'Basic '")
}
