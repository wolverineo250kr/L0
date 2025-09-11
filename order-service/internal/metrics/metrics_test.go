// internal/metrics/metrics_test.go
package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestOrdersProcessedCounter(t *testing.T) {
	require.Equal(t, float64(0), testutil.ToFloat64(OrdersProcessed.WithLabelValues("api", "success")))

	OrdersProcessed.WithLabelValues("api", "success").Inc()

	require.Equal(t, float64(1), testutil.ToFloat64(OrdersProcessed.WithLabelValues("api", "success")))
}

func TestOrderProcessingTimeHistogram(t *testing.T) {
	require.Equal(t, 0, testutil.CollectAndCount(OrderProcessingTime))

	OrderProcessingTime.WithLabelValues("kafka", "save").Observe(0.123)

	require.Equal(t, 1, testutil.CollectAndCount(OrderProcessingTime))
}

func TestCacheOperationsCounter(t *testing.T) {
	require.Equal(t, float64(0), testutil.ToFloat64(CacheOperations.WithLabelValues("get", "hit")))

	CacheOperations.WithLabelValues("get", "hit").Inc()

	require.Equal(t, float64(1), testutil.ToFloat64(CacheOperations.WithLabelValues("get", "hit")))
}

func TestDBOperationsCounter(t *testing.T) {
	require.Equal(t, float64(0), testutil.ToFloat64(DBOperations.WithLabelValues("save", "success")))
	DBOperations.WithLabelValues("save", "success").Inc()
	require.Equal(t, float64(1), testutil.ToFloat64(DBOperations.WithLabelValues("save", "success")))
}

func TestHTTPRequestsCounter(t *testing.T) {
	require.Equal(t, float64(0), testutil.ToFloat64(HTTPRequests.WithLabelValues("GET", "/api/test", "200")))
	HTTPRequests.WithLabelValues("GET", "/api/test", "200").Inc()
	require.Equal(t, float64(1), testutil.ToFloat64(HTTPRequests.WithLabelValues("GET", "/api/test", "200")))
}

func TestHTTPResponseTimeHistogram(t *testing.T) {
	require.Equal(t, 0, testutil.CollectAndCount(HTTPResponseTime))

	HTTPResponseTime.WithLabelValues("POST", "/api/test").Observe(0.456)

	require.Equal(t, 1, testutil.CollectAndCount(HTTPResponseTime))
}
