package server

import (
	"context"
	"net"
	"net/http"

	"github.com/arshchimni/tekton-monorepo-interceptor/diff"
	"go.uber.org/zap"
)

type Server struct {
	logger *zap.Logger

	mux    *http.ServeMux
	server *http.Server
	diff   diff.Diff
}

// New creates new HTTP server. The handlers are registered inside
// this function. gRPCPort is used to check gRPC server health check.
func New(logger *zap.Logger, diff diff.Diff) *Server {
	server := &Server{
		logger: logger,

		mux:  http.NewServeMux(),
		diff: diff,
	}
	server.registerHandlers()

	return server
}

// Serve starts accept requests from the given listener. If any returns error.
func (s *Server) Serve(ln net.Listener) error {
	s.logger.Info("starting HTTP server")

	server := &http.Server{
		Handler: s.mux,
	}
	s.server = server

	// ErrServerClosed is returned by the Server's Serve
	// after a call to Shutdown or Close, we can ignore it.
	if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// GracefulStop gracefully shuts down the server without interrupting any
// active connections. If any returns error.
func (s *Server) GracefulStop(ctx context.Context) error {
	s.logger.Info("shutting down monorepo interceptor HTTP server")
	return s.server.Shutdown(ctx)
}

// registerHandlers registers handler to the default server mux.
func (s *Server) registerHandlers() {
	s.mux.Handle("/monorepo", s.InterceptGitPayload())
}
