package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RPCRequestsTotal tracks the total number of RPC requests
	RPCRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_requests_total",
			Help: "Total number of RPC requests",
		},
		[]string{"method", "status"},
	)

	// RPCRequestDuration tracks the duration of RPC requests
	RPCRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rpc_request_duration_seconds",
			Help:    "Duration of RPC requests in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method"},
	)

	// RPCRequestsInFlight tracks the number of in-flight RPC requests
	RPCRequestsInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rpc_requests_in_flight",
			Help: "Number of in-flight RPC requests",
		},
		[]string{"method"},
	)

	// RPCRateLimitRejections tracks the number of rate limit rejections
	RPCRateLimitRejections = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_ratelimit_rejections_total",
			Help: "Total number of rate limit rejections",
		},
		[]string{"type"}, // type: global, ip, method
	)

	// RPCWebSocketConnections tracks the number of active WebSocket connections
	RPCWebSocketConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "rpc_websocket_connections",
			Help: "Number of active WebSocket connections",
		},
	)

	// RPCBatchRequestsTotal tracks the total number of batch requests
	RPCBatchRequestsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "rpc_batch_requests_total",
			Help: "Total number of batch RPC requests",
		},
	)

	// RPCBatchRequestSize tracks the size of batch requests
	RPCBatchRequestSize = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "rpc_batch_request_size",
			Help:    "Size of batch RPC requests",
			Buckets: []float64{1, 2, 5, 10, 20, 50, 100},
		},
	)

	// RPCSubscriptionsActive tracks the number of active subscriptions
	RPCSubscriptionsActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rpc_subscriptions_active",
			Help: "Number of active RPC subscriptions",
		},
		[]string{"type"}, // type: newHeads, logs, newPendingTransactions
	)

	// RPCSubscriptionNotifications tracks the number of subscription notifications sent
	RPCSubscriptionNotifications = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_subscription_notifications_total",
			Help: "Total number of subscription notifications sent",
		},
		[]string{"type"}, // type: newHeads, logs, newPendingTransactions
	)
)

// RecordRequest records an RPC request with status
func RecordRequest(method, status string, duration float64) {
	RPCRequestsTotal.WithLabelValues(method, status).Inc()
	RPCRequestDuration.WithLabelValues(method).Observe(duration)
}

// RecordInFlight records an in-flight RPC request
func RecordInFlight(method string, delta float64) {
	RPCRequestsInFlight.WithLabelValues(method).Add(delta)
}

// RecordRateLimit records a rate limit rejection
func RecordRateLimit(limitType string) {
	RPCRateLimitRejections.WithLabelValues(limitType).Inc()
}

// RecordWebSocketConnection records a WebSocket connection change
func RecordWebSocketConnection(delta float64) {
	RPCWebSocketConnections.Add(delta)
}

// RecordBatchRequest records a batch request
func RecordBatchRequest(size int) {
	RPCBatchRequestsTotal.Inc()
	RPCBatchRequestSize.Observe(float64(size))
}

// RecordSubscription records subscription changes
func RecordSubscription(subType string, delta float64) {
	RPCSubscriptionsActive.WithLabelValues(subType).Add(delta)
}

// RecordNotification records a subscription notification
func RecordNotification(subType string) {
	RPCSubscriptionNotifications.WithLabelValues(subType).Inc()
}
