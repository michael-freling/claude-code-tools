package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxy_ServeHTTP_ReadFromAnyRepo(t *testing.T) {
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer ghServer.Close()

	proxy := newTestProxy(t, ghServer.URL)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "info/refs git-upload-pack from project repo",
			method: http.MethodGet,
			path:   "/github.com/my-owner/my-repo.git/info/refs?service=git-upload-pack",
		},
		{
			name:   "info/refs git-upload-pack from other repo",
			method: http.MethodGet,
			path:   "/github.com/other-owner/other-repo.git/info/refs?service=git-upload-pack",
		},
		{
			name:   "git-upload-pack POST from project repo",
			method: http.MethodPost,
			path:   "/github.com/my-owner/my-repo.git/git-upload-pack",
		},
		{
			name:   "git-upload-pack POST from other repo",
			method: http.MethodPost,
			path:   "/github.com/other-owner/other-repo.git/git-upload-pack",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			proxy.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestProxy_ServeHTTP_PushToProjectRepoAllowed(t *testing.T) {
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("push-ok"))
	}))
	defer ghServer.Close()

	proxy := newTestProxy(t, ghServer.URL)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "info/refs git-receive-pack for project repo",
			method: http.MethodGet,
			path:   "/github.com/my-owner/my-repo.git/info/refs?service=git-receive-pack",
		},
		{
			name:   "git-receive-pack POST for project repo",
			method: http.MethodPost,
			path:   "/github.com/my-owner/my-repo.git/git-receive-pack",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			proxy.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestProxy_ServeHTTP_PushToOtherRepoBlocked(t *testing.T) {
	proxy := NewProxy(
		ProxyConfig{AllowedOwner: "my-owner", AllowedRepo: "my-repo"},
		NewGitHubAuthFromToken("test-token"),
	)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "info/refs git-receive-pack for other repo",
			method: http.MethodGet,
			path:   "/github.com/other-owner/other-repo.git/info/refs?service=git-receive-pack",
		},
		{
			name:   "git-receive-pack POST for other repo",
			method: http.MethodPost,
			path:   "/github.com/other-owner/other-repo.git/git-receive-pack",
		},
		{
			name:   "info/refs git-receive-pack for same owner different repo",
			method: http.MethodGet,
			path:   "/github.com/my-owner/other-repo.git/info/refs?service=git-receive-pack",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			proxy.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)
			assert.Contains(t, w.Body.String(), "forbidden")
		})
	}
}

func TestProxy_ServeHTTP_UnknownOperationDenied(t *testing.T) {
	proxy := NewProxy(
		ProxyConfig{AllowedOwner: "my-owner", AllowedRepo: "my-repo"},
		NewGitHubAuthFromToken("test-token"),
	)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "arbitrary path",
			method: http.MethodGet,
			path:   "/github.com/my-owner/my-repo.git/objects/info/packs",
		},
		{
			name:   "info/refs with unknown service",
			method: http.MethodGet,
			path:   "/github.com/my-owner/my-repo.git/info/refs?service=unknown",
		},
		{
			name:   "info/refs with no service",
			method: http.MethodGet,
			path:   "/github.com/my-owner/my-repo.git/info/refs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			proxy.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

func TestProxy_ServeHTTP_InvalidPath(t *testing.T) {
	proxy := NewProxy(
		ProxyConfig{AllowedOwner: "my-owner", AllowedRepo: "my-repo"},
		NewGitHubAuthFromToken("test-token"),
	)

	tests := []struct {
		name string
		path string
	}{
		{
			name: "no github.com prefix",
			path: "/example.com/owner/repo.git/info/refs",
		},
		{
			name: "missing repo",
			path: "/github.com/owner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			proxy.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestProxy_ServeHTTP_ForwardsAuthHeader(t *testing.T) {
	var capturedAuth string
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer ghServer.Close()

	proxy := newTestProxy(t, ghServer.URL)

	req := httptest.NewRequest(http.MethodGet, "/github.com/any-owner/any-repo.git/info/refs?service=git-upload-pack", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, capturedAuth, "Basic ")
}

func TestProxy_ServeHTTP_ForwardsResponseBody(t *testing.T) {
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("git-response-body"))
	}))
	defer ghServer.Close()

	proxy := newTestProxy(t, ghServer.URL)

	req := httptest.NewRequest(http.MethodGet, "/github.com/owner/repo.git/info/refs?service=git-upload-pack", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "git-response-body", w.Body.String())
	assert.Equal(t, "application/x-git-upload-pack-advertisement", w.Header().Get("Content-Type"))
}

func TestProxy_ServeHTTP_ForwardsRequestBody(t *testing.T) {
	var capturedBody string
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ghServer.Close()

	proxy := newTestProxy(t, ghServer.URL)

	req := httptest.NewRequest(http.MethodPost, "/github.com/owner/repo.git/git-upload-pack", strings.NewReader("pack-data"))
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pack-data", capturedBody)
}

func TestProxy_ServeHTTP_CaseInsensitiveOwnerRepo(t *testing.T) {
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ghServer.Close()

	proxy := newTestProxy(t, ghServer.URL)

	// Push with different casing should still match
	req := httptest.NewRequest(http.MethodPost, "/github.com/My-Owner/My-Repo.git/git-receive-pack", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProxy_ServeHTTP_ForwardsQueryParams(t *testing.T) {
	var capturedQuery string
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer ghServer.Close()

	proxy := newTestProxy(t, ghServer.URL)

	req := httptest.NewRequest(http.MethodGet, "/github.com/owner/repo.git/info/refs?service=git-upload-pack", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, capturedQuery, "service=git-upload-pack")
}

func TestProxy_ServeHTTP_UpstreamError(t *testing.T) {
	// Use a URL that will immediately fail
	proxy := NewProxy(
		ProxyConfig{AllowedOwner: "my-owner", AllowedRepo: "my-repo"},
		NewGitHubAuthFromToken("test-token"),
	)
	proxy.upstreamURL = "http://127.0.0.1:1" // port that should refuse connections

	req := httptest.NewRequest(http.MethodGet, "/github.com/my-owner/my-repo.git/info/refs?service=git-upload-pack", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "failed to contact GitHub")
}

func TestProxy_ServeHTTP_UpstreamPath(t *testing.T) {
	var capturedPath string
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer ghServer.Close()

	proxy := newTestProxy(t, ghServer.URL)

	req := httptest.NewRequest(http.MethodPost, "/github.com/owner/repo.git/git-upload-pack", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "/owner/repo.git/git-upload-pack", capturedPath)
}

func TestParseGitRequest(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		query     string
		wantOwner string
		wantRepo  string
		wantOp    string
		wantSvc   string
		wantErr   bool
	}{
		{
			name:      "info/refs with service",
			path:      "/github.com/owner/repo.git/info/refs",
			query:     "service=git-upload-pack",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantOp:    "info/refs",
			wantSvc:   "git-upload-pack",
		},
		{
			name:      "git-upload-pack",
			path:      "/github.com/owner/repo.git/git-upload-pack",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantOp:    "git-upload-pack",
		},
		{
			name:      "git-receive-pack",
			path:      "/github.com/owner/repo.git/git-receive-pack",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantOp:    "git-receive-pack",
		},
		{
			name:    "invalid prefix",
			path:    "/example.com/owner/repo.git/info/refs",
			wantErr: true,
		},
		{
			name:    "missing repo",
			path:    "/github.com/owner",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := tt.path
			if tt.query != "" {
				url += "?" + tt.query
			}
			r := httptest.NewRequest(http.MethodGet, url, nil)

			gr, err := parseGitRequest(r)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOwner, gr.Owner)
			assert.Equal(t, tt.wantRepo, gr.Repo)
			assert.Equal(t, tt.wantOp, gr.Operation)
			assert.Equal(t, tt.wantSvc, gr.Service)
		})
	}
}

func TestProxy_ServeHTTP_ForwardsRequestHeaders(t *testing.T) {
	var capturedHeaders http.Header
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer ghServer.Close()

	proxy := newTestProxy(t, ghServer.URL)

	req := httptest.NewRequest(http.MethodGet, "/github.com/owner/repo.git/info/refs?service=git-upload-pack", nil)
	req.Header.Set("X-Custom-Header", "custom-value")
	req.Header.Add("X-Multi-Header", "value1")
	req.Header.Add("X-Multi-Header", "value2")
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "custom-value", capturedHeaders.Get("X-Custom-Header"))
	values := capturedHeaders.Values("X-Multi-Header")
	assert.Contains(t, values, "value1")
	assert.Contains(t, values, "value2")
}

func TestProxy_ServeHTTP_EmptyOwnerRepoPath(t *testing.T) {
	proxy := NewProxy(
		ProxyConfig{AllowedOwner: "my-owner", AllowedRepo: "my-repo"},
		NewGitHubAuthFromToken("test-token"),
	)

	// Path that resolves to empty owner/repo after stripping github.com prefix
	req := httptest.NewRequest(http.MethodGet, "/github.com//.git/info/refs?service=git-upload-pack", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// newTestProxy creates a Proxy with its upstream URL pointed at the test server.
func newTestProxy(t *testing.T, testServerURL string) *Proxy {
	t.Helper()

	proxy := NewProxy(
		ProxyConfig{AllowedOwner: "my-owner", AllowedRepo: "my-repo"},
		NewGitHubAuthFromToken("test-token"),
	)
	proxy.upstreamURL = testServerURL

	return proxy
}
