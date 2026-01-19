package server

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/sunvim/evm_rpc/pkg/api"
	"github.com/sunvim/evm_rpc/pkg/logger"
	"github.com/sunvim/evm_rpc/pkg/metrics"
	"github.com/sunvim/evm_rpc/pkg/middleware"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *api.RPCError `json:"error,omitempty"`
}

// JSONRPCHandler handles JSON-RPC 2.0 requests
type JSONRPCHandler struct {
	methods           map[string]*methodHandler
	rateLimiter       *middleware.RateLimiter
	slowQueryThreshold time.Duration
}

// methodHandler holds information about a registered method
type methodHandler struct {
	receiver reflect.Value
	method   reflect.Method
	argTypes []reflect.Type
}

// NewJSONRPCHandler creates a new JSON-RPC handler
func NewJSONRPCHandler(rateLimiter *middleware.RateLimiter, slowQueryThreshold time.Duration) *JSONRPCHandler {
	return &JSONRPCHandler{
		methods:           make(map[string]*methodHandler),
		rateLimiter:       rateLimiter,
		slowQueryThreshold: slowQueryThreshold,
	}
}

// RegisterService registers all methods of a service
func (h *JSONRPCHandler) RegisterService(namespace string, service interface{}) error {
	serviceType := reflect.TypeOf(service)
	serviceValue := reflect.ValueOf(service)

	for i := 0; i < serviceType.NumMethod(); i++ {
		method := serviceType.Method(i)
		methodName := fmt.Sprintf("%s_%s", namespace, method.Name)

		// Validate method signature
		if !isValidMethod(method) {
			logger.Warnf("Skipping invalid method: %s", methodName)
			continue
		}

		// Extract argument types
		argTypes := make([]reflect.Type, method.Type.NumIn()-1) // -1 to skip receiver
		for j := 1; j < method.Type.NumIn(); j++ {
			argTypes[j-1] = method.Type.In(j)
		}

		h.methods[methodName] = &methodHandler{
			receiver: serviceValue,
			method:   method,
			argTypes: argTypes,
		}

		logger.Debugf("Registered RPC method: %s", methodName)
	}

	return nil
}

// isValidMethod checks if a method has a valid signature for RPC
// Valid signature: func(ctx context.Context, args...) (result, error)
func isValidMethod(method reflect.Method) bool {
	mType := method.Type

	// Must have at least 2 inputs (receiver + context)
	if mType.NumIn() < 2 {
		return false
	}

	// First argument (after receiver) must be context.Context
	if mType.In(1) != reflect.TypeOf((*context.Context)(nil)).Elem() {
		return false
	}

	// Must have exactly 2 outputs
	if mType.NumOut() != 2 {
		return false
	}

	// Last output must be error
	if !mType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return false
	}

	return true
}

// HandleRequest handles a single JSON-RPC request
func (h *JSONRPCHandler) HandleRequest(ctx context.Context, req *JSONRPCRequest, clientIP string) *JSONRPCResponse {
	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   api.NewRPCError(api.ErrCodeInvalidRequest, "invalid jsonrpc version"),
		}
	}

	// Check rate limit
	if h.rateLimiter != nil {
		allowed, limitType := h.rateLimiter.Allow(clientIP, req.Method)
		if !allowed {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   api.NewRPCError(api.ErrCodeLimitExceeded, fmt.Sprintf("rate limit exceeded: %s", limitType)),
			}
		}
	}

	// Find method handler
	handler, exists := h.methods[req.Method]
	if !exists {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   api.NewRPCError(api.ErrCodeMethodNotFound, fmt.Sprintf("method not found: %s", req.Method)),
		}
	}

	// Track in-flight requests
	metrics.RecordInFlight(req.Method, 1)
	defer metrics.RecordInFlight(req.Method, -1)

	// Execute method
	start := time.Now()
	result, err := h.executeMethod(ctx, handler, req.Params)
	duration := time.Since(start)

	// Log request
	middleware.LogRPCRequest(req.Method, req.Params)
	middleware.LogRPCResponse(req.Method, duration, err)
	middleware.LogSlowRPCRequest(req.Method, duration, h.slowQueryThreshold)
	middleware.RecordRPCMetrics(req.Method, duration, err)

	// Build response
	resp := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	if err != nil {
		// Check if error is already an RPCError
		if rpcErr, ok := err.(*api.RPCError); ok {
			resp.Error = rpcErr
		} else {
			resp.Error = &api.RPCError{
				Code:    api.ErrCodeInternal,
				Message: err.Error(),
			}
		}
	} else {
		resp.Result = result
	}

	return resp
}

// executeMethod executes a method with the given parameters
func (h *JSONRPCHandler) executeMethod(ctx context.Context, handler *methodHandler, params json.RawMessage) (interface{}, error) {
	// Parse parameters
	args := make([]reflect.Value, len(handler.argTypes))
	
	// First argument is always context
	args[0] = reflect.ValueOf(ctx)

	// Parse remaining arguments
	if len(handler.argTypes) > 1 {
		// Unmarshal params into slice or struct
		var paramList []json.RawMessage
		
		// Try to unmarshal as array first
		if err := json.Unmarshal(params, &paramList); err != nil {
			// If that fails, wrap it in an array
			paramList = []json.RawMessage{params}
		}

		// Create arguments from param list
		for i := 1; i < len(handler.argTypes); i++ {
			argType := handler.argTypes[i]
			arg := reflect.New(argType)

			if i-1 < len(paramList) {
				if err := json.Unmarshal(paramList[i-1], arg.Interface()); err != nil {
					return nil, api.NewRPCError(api.ErrCodeInvalidParams, fmt.Sprintf("invalid param %d: %v", i, err))
				}
			}

			args[i] = arg.Elem()
		}
	}

	// Call method
	results := handler.method.Func.Call(append([]reflect.Value{handler.receiver}, args...))

	// Extract results
	var result interface{}
	var err error

	if !results[0].IsNil() {
		result = results[0].Interface()
	}

	if !results[1].IsNil() {
		err = results[1].Interface().(error)
	}

	return result, err
}

// HandleBatch handles a batch of JSON-RPC requests
func (h *JSONRPCHandler) HandleBatch(ctx context.Context, requests []*JSONRPCRequest, clientIP string) []*JSONRPCResponse {
	metrics.RecordBatchRequest(len(requests))

	responses := make([]*JSONRPCResponse, len(requests))
	for i, req := range requests {
		responses[i] = h.HandleRequest(ctx, req, clientIP)
	}

	return responses
}

// ParseRequest parses a JSON-RPC request from raw bytes
func ParseRequest(data []byte) (interface{}, error) {
	// Try to parse as single request first
	var single JSONRPCRequest
	if err := json.Unmarshal(data, &single); err == nil && single.JSONRPC != "" {
		return &single, nil
	}

	// Try to parse as batch request
	var batch []*JSONRPCRequest
	if err := json.Unmarshal(data, &batch); err != nil {
		return nil, api.NewRPCError(api.ErrCodeParse, "failed to parse request")
	}

	if len(batch) == 0 {
		return nil, api.NewRPCError(api.ErrCodeInvalidRequest, "empty batch request")
	}

	return batch, nil
}
