package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sunvim/evm_rpc/pkg/logger"
)

// Server represents a metrics HTTP server
type Server struct {
	server *http.Server
	addr   string
}

// NewServer creates a new metrics server
func NewServer(addr string) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	return &Server{
		server: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		addr: addr,
	}
}

// Start starts the metrics server
func (s *Server) Start() error {
	logger.Infof("Starting metrics server on %s", s.addr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("metrics server failed: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the metrics server
func (s *Server) Stop(ctx context.Context) error {
	logger.Info("Stopping metrics server...")
	return s.server.Shutdown(ctx)
}
