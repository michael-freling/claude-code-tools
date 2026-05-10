package gateway

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Server is the main gateway server that runs both the proxy and API server.
type Server struct {
	proxy     *Proxy
	apiServer *APIServer
}

// NewServer creates a new gateway server with the given config.
// It resolves GitHub authentication automatically.
func NewServer(config ProxyConfig) (*Server, error) {
	ghAuth, err := NewGitHubAuth()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GitHub auth: %w", err)
	}

	return &Server{
		proxy:     NewProxy(config, ghAuth),
		apiServer: NewAPIServer(config, ghAuth),
	}, nil
}

// NewServerWithAuth creates a new gateway server with explicit auth.
func NewServerWithAuth(config ProxyConfig, ghAuth *GitHubAuth) *Server {
	return &Server{
		proxy:     NewProxy(config, ghAuth),
		apiServer: NewAPIServer(config, ghAuth),
	}
}

// Run starts both servers. Proxy listens on proxyAddr and the API server
// listens on apiAddr. It blocks until an OS interrupt signal is received,
// then shuts down both servers gracefully.
func (s *Server) Run(proxyAddr, apiAddr string) error {
	proxyServer := &http.Server{
		Addr:    proxyAddr,
		Handler: s.proxy,
	}
	apiHTTPServer := &http.Server{
		Addr:    apiAddr,
		Handler: s.apiServer,
	}

	errCh := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := proxyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("proxy server error: %w", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := apiHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("API server error: %w", err)
		}
	}()

	// Wait for interrupt or error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		fmt.Fprintf(os.Stderr, "received signal %v, shutting down\n", sig)
	case err := <-errCh:
		return err
	}

	// Graceful shutdown
	ctx := context.Background()
	proxyServer.Shutdown(ctx)
	apiHTTPServer.Shutdown(ctx)

	wg.Wait()
	return nil
}
