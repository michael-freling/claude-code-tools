//go:build forge_e2e

package forge_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/michael-freling/claude-code-tools/internal/forgegh"
	"github.com/michael-freling/claude-code-tools/internal/gateway"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startMockGitHubAPI creates a mock GitHub API server with canned responses.
func startMockGitHubAPI(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /repos/{owner}/{repo}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"full_name":   "test-owner/test-repo",
			"description": "Test repo",
			"private":     false,
		})
	})

	mux.HandleFunc("GET /repos/{owner}/{repo}/pulls", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{"number": 1, "title": "Test PR", "state": "open"},
		})
	})

	mux.HandleFunc("POST /repos/{owner}/{repo}/issues", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"number": 42,
			"title":  body["title"],
			"state":  "open",
		})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// startGatewayAPIServer creates a gateway API server backed by the given mock GitHub URL.
func startGatewayAPIServer(t *testing.T, mockGHURL string) string {
	t.Helper()

	config := gateway.ProxyConfig{
		AllowedOwner: "test-owner",
		AllowedRepo:  "test-repo",
	}
	ghAuth := gateway.NewGitHubAuthFromToken("test-github-token")
	apiServer := gateway.NewTestAPIServer(config, ghAuth, mockGHURL)

	server := httptest.NewServer(apiServer)
	t.Cleanup(server.Close)
	return server.URL
}

func TestGateway_SchemaDiscovery(t *testing.T) {
	mockGH := startMockGitHubAPI(t)
	apiURL := startGatewayAPIServer(t, mockGH.URL)

	resp, err := http.Get(apiURL + "/api/schema")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var schema gateway.SchemaResponse
	err = json.NewDecoder(resp.Body).Decode(&schema)
	require.NoError(t, err)

	opNames := make(map[string]bool)
	for _, op := range schema.Operations {
		opNames[op.Name] = true
	}

	assert.True(t, opNames["list-prs"], "should have list-prs operation")
	assert.True(t, opNames["get-repo"], "should have get-repo operation")
	assert.True(t, opNames["create-pr"], "should have create-pr operation")
	assert.True(t, opNames["create-issue"], "should have create-issue operation")
}

func TestGateway_ForgeGH_RepoView(t *testing.T) {
	mockGH := startMockGitHubAPI(t)
	apiURL := startGatewayAPIServer(t, mockGH.URL)

	var stdout bytes.Buffer
	client := forgegh.NewTestClient(apiURL, &stdout, io.Discard)

	err := client.Run([]string{"repo", "view", "--repo", "test-owner/test-repo"})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "test-owner/test-repo")
}

func TestGateway_ForgeGH_PRList(t *testing.T) {
	mockGH := startMockGitHubAPI(t)
	apiURL := startGatewayAPIServer(t, mockGH.URL)

	var stdout bytes.Buffer
	client := forgegh.NewTestClient(apiURL, &stdout, io.Discard)

	err := client.Run([]string{"pr", "list", "--repo", "test-owner/test-repo"})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Test PR")
}

func TestGateway_ForgeGH_IssueCreate_AllowedRepo(t *testing.T) {
	mockGH := startMockGitHubAPI(t)
	apiURL := startGatewayAPIServer(t, mockGH.URL)

	var stdout bytes.Buffer
	client := forgegh.NewTestClient(apiURL, &stdout, io.Discard)

	err := client.Run([]string{"issue", "create", "--repo", "test-owner/test-repo", "--title", "Test Issue", "--body", "Body text"})
	require.NoError(t, err)
}

func TestGateway_ForgeGH_IssueCreate_DeniedRepo(t *testing.T) {
	mockGH := startMockGitHubAPI(t)
	apiURL := startGatewayAPIServer(t, mockGH.URL)

	var stdout bytes.Buffer
	client := forgegh.NewTestClient(apiURL, &stdout, io.Discard)

	err := client.Run([]string{"issue", "create", "--repo", "other-owner/other-repo", "--title", "Test Issue"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestGateway_ForgeGH_RepoView_WithEnvVars(t *testing.T) {
	mockGH := startMockGitHubAPI(t)
	apiURL := startGatewayAPIServer(t, mockGH.URL)

	t.Setenv("FORGE_PROJECT_OWNER", "test-owner")
	t.Setenv("FORGE_PROJECT_REPO", "test-repo")

	var stdout bytes.Buffer
	client := forgegh.NewTestClient(apiURL, &stdout, io.Discard)

	err := client.Run([]string{"repo", "view"})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "test-owner/test-repo")
}
