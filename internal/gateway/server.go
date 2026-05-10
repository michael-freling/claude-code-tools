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
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	return s.RunWithContext(ctx, proxyAddr, apiAddr)
}

// RunWithContext starts both servers and blocks until the context is cancelled
// or a server error occurs, then shuts down gracefully.
func (s *Server) RunWithContext(ctx context.Context, proxyAddr, apiAddr string) error {
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

	select {
	case <-ctx.Done():
		fmt.Fprintf(os.Stderr, "shutting down\n")
	case err := <-errCh:
		return err
	}

	// Graceful shutdown
	shutdownCtx := context.Background()
	proxyServer.Shutdown(shutdownCtx)
	apiHTTPServer.Shutdown(shutdownCtx)

	wg.Wait()
	return nil
}
