package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sunvim/evm_rpc/pkg/api"
	"github.com/sunvim/evm_rpc/pkg/config"
	"github.com/sunvim/evm_rpc/pkg/logger"
	"github.com/sunvim/evm_rpc/pkg/metrics"
)

// WebSocketServer represents a WebSocket JSON-RPC server
type WebSocketServer struct {
	server              *http.Server
	handler             *JSONRPCHandler
	subscriptionManager *SubscriptionManager
	config              config.WSConfig
	upgrader            websocket.Upgrader
	connections         map[*WebSocketConnection]bool
	connMutex           sync.RWMutex
	maxConnections      int
}

// WebSocketConnection represents a WebSocket connection
type WebSocketConnection struct {
	conn      *websocket.Conn
	writeMux  sync.Mutex
	sendChan  chan interface{}
	closeChan chan struct{}
	closed    bool
	clientIP  string
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(
	cfg config.WSConfig,
	handler *JSONRPCHandler,
	subscriptionManager *SubscriptionManager,
) *WebSocketServer {
	ws := &WebSocketServer{
		handler:             handler,
		subscriptionManager: subscriptionManager,
		config:              cfg,
		connections:         make(map[*WebSocketConnection]bool),
		maxConnections:      cfg.MaxConnections,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  cfg.ReadBufferSize,
			WriteBufferSize: cfg.WriteBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins; configure as needed
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ws.handleWebSocket)

	ws.server = &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: mux,
	}

	return ws
}

// Start starts the WebSocket server
func (s *WebSocketServer) Start() error {
	logger.Infof("Starting WebSocket server on %s", s.config.ListenAddr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("WebSocket server failed: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the WebSocket server
func (s *WebSocketServer) Stop(ctx context.Context) error {
	logger.Info("Stopping WebSocket server...")
	
	// Close all connections
	s.connMutex.Lock()
	for conn := range s.connections {
		conn.Close()
	}
	s.connMutex.Unlock()

	return s.server.Shutdown(ctx)
}

// handleWebSocket handles WebSocket upgrade and communication
func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check connection limit
	s.connMutex.RLock()
	connCount := len(s.connections)
	s.connMutex.RUnlock()

	if s.maxConnections > 0 && connCount >= s.maxConnections {
		http.Error(w, "max connections reached", http.StatusServiceUnavailable)
		return
	}

	// Upgrade connection
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Errorf("Failed to upgrade WebSocket connection: %v", err)
		return
	}

	// Create WebSocket connection
	wsConn := &WebSocketConnection{
		conn:      conn,
		sendChan:  make(chan interface{}, 256),
		closeChan: make(chan struct{}),
		clientIP:  extractIP(r),
	}

	// Register connection
	s.connMutex.Lock()
	s.connections[wsConn] = true
	s.connMutex.Unlock()

	// Update metrics
	metrics.RecordWebSocketConnection(1)

	logger.Infof("WebSocket connection established: %s", wsConn.clientIP)

	// Start goroutines for reading and writing
	go wsConn.writePump()
	go s.handleConnection(wsConn)
}

// handleConnection handles messages from a WebSocket connection
func (s *WebSocketServer) handleConnection(wsConn *WebSocketConnection) {
	defer func() {
		// Cleanup on disconnect
		s.connMutex.Lock()
		delete(s.connections, wsConn)
		s.connMutex.Unlock()

		// Unsubscribe all subscriptions
		s.subscriptionManager.UnsubscribeAll(wsConn)

		// Update metrics
		metrics.RecordWebSocketConnection(-1)

		wsConn.Close()
		logger.Infof("WebSocket connection closed: %s", wsConn.clientIP)
	}()

	// Set read deadline
	wsConn.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	wsConn.conn.SetPongHandler(func(string) error {
		wsConn.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		// Read message
		_, message, err := wsConn.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Errorf("WebSocket read error: %v", err)
			}
			return
		}

		// Parse request
		req, err := ParseRequest(message)
		if err != nil {
			wsConn.SendError(nil, api.ErrCodeParse, err.Error())
			continue
		}

		// Handle request based on type
		ctx := context.Background()

		switch v := req.(type) {
		case *JSONRPCRequest:
			// Check for subscription methods
			if v.Method == "eth_subscribe" {
				s.handleSubscribe(wsConn, v)
			} else if v.Method == "eth_unsubscribe" {
				s.handleUnsubscribe(wsConn, v)
			} else {
				// Regular JSON-RPC request
				response := s.handler.HandleRequest(ctx, v, wsConn.clientIP)
				wsConn.Send(response)
			}
		case []*JSONRPCRequest:
			// Batch request
			responses := s.handler.HandleBatch(ctx, v, wsConn.clientIP)
			wsConn.Send(responses)
		}
	}
}

// handleSubscribe handles eth_subscribe requests
func (s *WebSocketServer) handleSubscribe(wsConn *WebSocketConnection, req *JSONRPCRequest) {
	// Parse params
	var params []json.RawMessage
	if err := json.Unmarshal(req.Params, &params); err != nil {
		wsConn.SendError(req.ID, api.ErrCodeInvalidParams, "invalid params")
		return
	}

	if len(params) == 0 {
		wsConn.SendError(req.ID, api.ErrCodeInvalidParams, "missing subscription type")
		return
	}

	// Parse subscription type
	var subType string
	if err := json.Unmarshal(params[0], &subType); err != nil {
		wsConn.SendError(req.ID, api.ErrCodeInvalidParams, "invalid subscription type")
		return
	}

	// Parse filter criteria for logs subscription
	var filter *FilterCriteria
	if subType == "logs" && len(params) > 1 {
		filter = &FilterCriteria{}
		if err := json.Unmarshal(params[1], filter); err != nil {
			wsConn.SendError(req.ID, api.ErrCodeInvalidParams, "invalid filter criteria")
			return
		}
	}

	// Create subscription
	subID, err := s.subscriptionManager.Subscribe(wsConn, SubscriptionType(subType), filter)
	if err != nil {
		wsConn.SendError(req.ID, api.ErrCodeInternal, err.Error())
		return
	}

	// Send response
	response := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  subID,
	}
	wsConn.Send(response)
}

// handleUnsubscribe handles eth_unsubscribe requests
func (s *WebSocketServer) handleUnsubscribe(wsConn *WebSocketConnection, req *JSONRPCRequest) {
	// Parse params
	var params []string
	if err := json.Unmarshal(req.Params, &params); err != nil {
		wsConn.SendError(req.ID, api.ErrCodeInvalidParams, "invalid params")
		return
	}

	if len(params) == 0 {
		wsConn.SendError(req.ID, api.ErrCodeInvalidParams, "missing subscription ID")
		return
	}

	subID := params[0]

	// Unsubscribe
	if err := s.subscriptionManager.Unsubscribe(subID); err != nil {
		wsConn.SendError(req.ID, api.ErrCodeInternal, err.Error())
		return
	}

	// Send response
	response := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  true,
	}
	wsConn.Send(response)
}

// Send sends a message to the WebSocket connection
func (c *WebSocketConnection) Send(msg interface{}) {
	if c.closed {
		return
	}
	select {
	case c.sendChan <- msg:
	default:
		logger.Warn("WebSocket send channel full, dropping message")
	}
}

// SendNotification sends a subscription notification
func (c *WebSocketConnection) SendNotification(notification interface{}) error {
	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_subscription",
		"params":  notification,
	}
	c.Send(msg)
	return nil
}

// SendError sends an error response
func (c *WebSocketConnection) SendError(id interface{}, code int, message string) {
	response := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &api.RPCError{
			Code:    code,
			Message: message,
		},
	}
	c.Send(response)
}

// writePump pumps messages from the send channel to the WebSocket connection
func (c *WebSocketConnection) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.sendChan:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.writeMux.Lock()
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteJSON(message); err != nil {
				c.writeMux.Unlock()
				logger.Errorf("WebSocket write error: %v", err)
				return
			}
			c.writeMux.Unlock()

		case <-ticker.C:
			c.writeMux.Lock()
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.writeMux.Unlock()
				return
			}
			c.writeMux.Unlock()

		case <-c.closeChan:
			return
		}
	}
}

// Close closes the WebSocket connection
func (c *WebSocketConnection) Close() {
	if c.closed {
		return
	}
	c.closed = true
	close(c.closeChan)
	close(c.sendChan)
	c.conn.Close()
}
