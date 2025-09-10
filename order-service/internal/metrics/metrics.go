// internal/metrics/metrics.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	OrdersProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orders_processed_total",
			Help: "Total number of processed orders",
		},
		[]string{"source", "status"}, // source: kafka, api; status: success, error, validation_error
	)

	OrderProcessingTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "order_processing_seconds",
			Help:    "Time spent processing orders",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
		},
		[]string{"source", "operation"},
	)

	CacheOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_operations_total",
			Help: "Total cache operations",
		},
		[]string{"type", "result"}, // type: get, set; result: hit, miss
	)

	DBOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_operations_total",
			Help: "Total database operations",
		},
		[]string{"operation", "status"}, // operation: save, get; status: success, error
	)

	HTTPRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPResponseTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_time_seconds",
			Help:    "HTTP response time",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
		},
		[]string{"method", "path"},
	)
)

func InitMetrics() {
	// Метрики автоматически регистрируются при импорте пакета
}
