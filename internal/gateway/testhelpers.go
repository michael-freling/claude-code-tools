package gateway

import "net/http"

// NewTestProxy creates a Proxy with a custom upstream URL for testing.
func NewTestProxy(config ProxyConfig, ghAuth *GitHubAuth, upstreamURL string) *Proxy {
	return &Proxy{
		config:      config,
		ghAuth:      ghAuth,
		upstreamURL: upstreamURL,
		httpClient:  http.DefaultClient,
	}
}

// NewTestAPIServer creates an APIServer with a custom upstream URL for testing.
func NewTestAPIServer(config ProxyConfig, ghAuth *GitHubAuth, upstreamURL string) *APIServer {
	return &APIServer{
		config:      config,
		ghAuth:      ghAuth,
		upstreamURL: upstreamURL,
		httpClient:  http.DefaultClient,
	}
}
