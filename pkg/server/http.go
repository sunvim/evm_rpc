package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sunvim/evm_rpc/pkg/api"
	"github.com/sunvim/evm_rpc/pkg/config"
	"github.com/sunvim/evm_rpc/pkg/logger"
	"github.com/sunvim/evm_rpc/pkg/middleware"
	"github.com/sunvim/evm_rpc/pkg/storage"
)

// HTTPServer represents an HTTP JSON-RPC server
type HTTPServer struct {
	server      *http.Server
	handler     *JSONRPCHandler
	blockReader *storage.BlockReader
	config      config.HTTPConfig
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(
	cfg config.HTTPConfig,
	handler *JSONRPCHandler,
	blockReader *storage.BlockReader,
	rateLimiter *middleware.RateLimiter,
	loggingMiddleware *middleware.LoggingMiddleware,
	corsMiddleware *cors.Cors,
) *HTTPServer {
	router := mux.NewRouter()

	httpServer := &HTTPServer{
		handler:     handler,
		blockReader: blockReader,
		config:      cfg,
	}

	// Health check endpoint
	router.HandleFunc("/health", httpServer.handleHealth).Methods("GET")

	// JSON-RPC endpoint
	router.HandleFunc("/", httpServer.handleRPC).Methods("POST")

	// Apply middleware
	var h http.Handler = router

	// CORS middleware (outermost)
	if corsMiddleware != nil {
		h = corsMiddleware.Handler(h)
	}

	// Rate limiting middleware
	if rateLimiter != nil {
		h = rateLimiter.Middleware()(h)
	}

	// Logging middleware (innermost)
	if loggingMiddleware != nil {
		h = loggingMiddleware.Middleware()(h)
	}

	httpServer.server = &http.Server{
		Addr:           cfg.ListenAddr,
		Handler:        h,
		ReadTimeout:    cfg.ReadTimeout,
		WriteTimeout:   cfg.WriteTimeout,
		IdleTimeout:    cfg.IdleTimeout,
		MaxHeaderBytes: cfg.MaxHeaderBytes,
	}

	return httpServer
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	logger.Infof("Starting HTTP server on %s", s.config.ListenAddr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server failed: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the HTTP server
func (s *HTTPServer) Stop(ctx context.Context) error {
	logger.Info("Stopping HTTP server...")
	return s.server.Shutdown(ctx)
}

// handleHealth handles health check requests
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get latest block number to check if we're synced
	latestBlock, err := s.blockReader.GetLatestBlockNumber(ctx)
	
	health := map[string]interface{}{
		"status": "ok",
		"syncing": false,
	}

	if err != nil {
		health["status"] = "degraded"
		health["error"] = err.Error()
	} else {
		health["latestBlock"] = latestBlock
		
		// Check if we're significantly behind (this is a simple heuristic)
		// In production, you'd compare with actual network block height
		timeSinceUpdate := time.Since(time.Now()) // Placeholder
		if timeSinceUpdate > 5*time.Minute {
			health["syncing"] = true
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

// handleRPC handles JSON-RPC requests
func (s *HTTPServer) handleRPC(w http.ResponseWriter, r *http.Request) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendJSONRPCError(w, nil, -32700, "failed to read request body")
		return
	}
	defer r.Body.Close()

	// Parse request
	req, err := ParseRequest(body)
	if err != nil {
		sendJSONRPCError(w, nil, -32700, err.Error())
		return
	}

	// Extract client IP
	clientIP := extractIP(r)

	// Handle request based on type
	var response interface{}
	ctx := r.Context()

	switch v := req.(type) {
	case *JSONRPCRequest:
		// Single request
		response = s.handler.HandleRequest(ctx, v, clientIP)
	case []*JSONRPCRequest:
		// Batch request
		response = s.handler.HandleBatch(ctx, v, clientIP)
	default:
		sendJSONRPCError(w, nil, -32600, "invalid request")
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("Failed to encode response: %v", err)
	}
}

// sendJSONRPCError sends a JSON-RPC error response
func sendJSONRPCError(w http.ResponseWriter, id interface{}, code int, message string) {
	response := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &api.RPCError{
			Code:    code,
			Message: message,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC always returns 200
	json.NewEncoder(w).Encode(response)
}

// extractIP extracts the client IP address from the request
func extractIP(r *http.Request) string {
	// Try X-Forwarded-For header first
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		return ip
	}

	// Try X-Real-IP header
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
