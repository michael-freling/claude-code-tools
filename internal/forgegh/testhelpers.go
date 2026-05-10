package forgegh

import (
	"io"
	"net/http"
	"strings"
)

// NewTestClient creates a Client for testing with custom stdout and stderr writers.
func NewTestClient(gatewayURL string, stdout, stderr io.Writer) *Client {
	return &Client{
		gatewayURL: strings.TrimRight(gatewayURL, "/"),
		httpClient: http.DefaultClient,
		stdout:     stdout,
		stderr:     stderr,
	}
}
