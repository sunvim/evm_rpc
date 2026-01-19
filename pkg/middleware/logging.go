package middleware

import (
	"net/http"
	"time"

	"github.com/sunvim/evm_rpc/pkg/logger"
	"github.com/sunvim/evm_rpc/pkg/metrics"
)

// LoggingMiddleware logs HTTP requests
type LoggingMiddleware struct {
	slowQueryThreshold time.Duration
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(slowQueryThreshold time.Duration) *LoggingMiddleware {
	return &LoggingMiddleware{
		slowQueryThreshold: slowQueryThreshold,
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Middleware creates an HTTP middleware for logging
func (lm *LoggingMiddleware) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := newResponseWriter(w)

			// Log incoming request
			logger.Debugf("Incoming request: method=%s, path=%s, remote=%s",
				r.Method, r.URL.Path, r.RemoteAddr)

			// Process request
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start)

			// Log response
			logger.Infof("Request completed: method=%s, path=%s, status=%d, duration=%v",
				r.Method, r.URL.Path, wrapped.statusCode, duration)

			// Log slow queries
			if duration > lm.slowQueryThreshold {
				logger.Warnf("Slow query detected: method=%s, path=%s, duration=%v",
					r.Method, r.URL.Path, duration)
			}
		})
	}
}

// LogRPCRequest logs an RPC request with method and params
func LogRPCRequest(method string, params interface{}) {
	logger.Debugf("RPC request: method=%s, params=%v", method, params)
}

// LogRPCResponse logs an RPC response with duration
func LogRPCResponse(method string, duration time.Duration, err error) {
	if err != nil {
		logger.Warnf("RPC response: method=%s, duration=%v, error=%v", method, duration, err)
	} else {
		logger.Debugf("RPC response: method=%s, duration=%v", method, duration)
	}
}

// LogSlowRPCRequest logs a slow RPC request
func LogSlowRPCRequest(method string, duration time.Duration, threshold time.Duration) {
	if duration > threshold {
		logger.Warnf("Slow RPC request: method=%s, duration=%v, threshold=%v",
			method, duration, threshold)
	}
}

// RecordRPCMetrics records metrics for an RPC request
func RecordRPCMetrics(method string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	metrics.RecordRequest(method, status, duration.Seconds())
}
