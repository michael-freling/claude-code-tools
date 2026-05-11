package gateway

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ProxyConfig holds configuration for the gateway proxy.
type ProxyConfig struct {
	AllowedOwner string
	AllowedRepo  string
}

// defaultGitHubBaseURL is the default upstream base URL for git operations.
const defaultGitHubBaseURL = "https://github.com"

// Proxy is an HTTP reverse proxy that forwards git operations to github.com.
// The agent's gitconfig rewrites github.com URLs to go through this proxy:
//
//	[url "http://gateway:8080/github.com/"]
//	    insteadOf = https://github.com/
//
// This means requests arrive as plain HTTP: /github.com/{owner}/{repo}.git/{operation}
type Proxy struct {
	config      ProxyConfig
	ghAuth      *GitHubAuth
	upstreamURL string // base URL for upstream git server, defaults to https://github.com
	httpClient  *http.Client
}

// NewProxy creates a new git proxy with the given config and auth.
func NewProxy(config ProxyConfig, ghAuth *GitHubAuth) *Proxy {
	return &Proxy{
		config:      config,
		ghAuth:      ghAuth,
		upstreamURL: defaultGitHubBaseURL,
		httpClient:  http.DefaultClient,
	}
}

// gitRequest represents a parsed git HTTP request.
type gitRequest struct {
	Owner     string
	Repo      string
	Operation string
	Service   string // query param service value for info/refs
}

// ServeHTTP handles requests matching /github.com/{owner}/{repo}.git/{operation}.
// It enforces access control:
//   - Read operations (git-upload-pack) are allowed for any repo
//   - Write operations (git-receive-pack) are allowed only for the configured project
//   - All other requests are denied
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	gr, err := parseGitRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !p.isAllowed(gr, r.Method) {
		http.Error(w, "forbidden: write access denied for this repository", http.StatusForbidden)
		return
	}

	p.forwardToGitHub(w, r, gr)
}

// parseGitRequest extracts owner, repo, and operation from the request path.
// Expected path format: /github.com/{owner}/{repo}.git/{operation...}
func parseGitRequest(r *http.Request) (*gitRequest, error) {
	path := strings.TrimPrefix(r.URL.Path, "/github.com/")
	if path == r.URL.Path {
		return nil, fmt.Errorf("path must start with /github.com/")
	}

	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid path: expected /github.com/{owner}/{repo}.git/{operation}")
	}

	owner := parts[0]
	repoRaw := parts[1]

	// The repo part may or may not end with ".git"; strip the suffix.
	repo := strings.TrimSuffix(repoRaw, ".git")

	// The third part (if present) is the git operation path.
	operation := ""
	if len(parts) == 3 {
		operation = parts[2]
	}

	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid path: owner and repo must not be empty")
	}

	gr := &gitRequest{
		Owner:     owner,
		Repo:      repo,
		Operation: operation,
	}

	// Extract service query param for info/refs
	if strings.HasSuffix(operation, "info/refs") {
		gr.Service = r.URL.Query().Get("service")
	}

	return gr, nil
}

// isAllowed checks whether the request is permitted based on the operation type.
func (p *Proxy) isAllowed(gr *gitRequest, method string) bool {
	op := gr.Operation

	// info/refs endpoint
	if strings.HasSuffix(op, "info/refs") {
		switch gr.Service {
		case "git-upload-pack":
			// Read: allowed for any repo
			return true
		case "git-receive-pack":
			// Write: only allowed for the configured project
			return p.isProjectRepo(gr)
		default:
			// Unknown service
			return false
		}
	}

	// git-upload-pack POST: read operation, allowed for any repo
	if strings.HasSuffix(op, "git-upload-pack") && method == http.MethodPost {
		return true
	}

	// git-receive-pack POST: write operation, only allowed for project repo
	if strings.HasSuffix(op, "git-receive-pack") && method == http.MethodPost {
		return p.isProjectRepo(gr)
	}

	// Everything else is denied
	return false
}

// isProjectRepo checks if the request targets the allowed project repository.
func (p *Proxy) isProjectRepo(gr *gitRequest) bool {
	return strings.EqualFold(gr.Owner, p.config.AllowedOwner) &&
		strings.EqualFold(gr.Repo, p.config.AllowedRepo)
}

// forwardToGitHub forwards the request to the actual GitHub server.
func (p *Proxy) forwardToGitHub(w http.ResponseWriter, r *http.Request, gr *gitRequest) {
	// Build the upstream GitHub URL
	targetURL := fmt.Sprintf("%s/%s/%s.git/%s", p.upstreamURL, gr.Owner, gr.Repo, gr.Operation)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// Create the upstream request
	upstreamReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create upstream request: %v", err), http.StatusInternalServerError)
		return
	}

	// Copy relevant headers
	for key, values := range r.Header {
		for _, value := range values {
			upstreamReq.Header.Add(key, value)
		}
	}

	// Add GitHub authentication
	upstreamReq.Header.Set("Authorization", "Basic "+basicAuth("x-access-token", p.ghAuth.Token()))

	// Execute the upstream request
	resp, err := p.httpClient.Do(upstreamReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to contact GitHub: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy the response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Copy the response body
	io.Copy(w, resp.Body)
}

// basicAuth encodes user:pass for HTTP Basic Auth.
func basicAuth(user, pass string) string {
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
}
