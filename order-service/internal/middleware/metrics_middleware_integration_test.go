//internal/middleware/metrics_middleware_integration_test.go
//go:build integration

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"order-service/internal/metrics"

	"github.com/gorilla/mux"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestMetricsMiddleware_Integration(t *testing.T) {

	metrics.HTTPRequests.Reset()
	metrics.HTTPResponseTime.Reset()

	r := mux.NewRouter()
	r.Handle("/test", MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("ok"))
	})))


	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Result().StatusCode)
	assert.Equal(t, "ok", rec.Body.String())

	mf, err := metrics.HTTPRequests.MetricVec.GetMetricWithLabelValues("GET", "/test", "Created")
	assert.NoError(t, err)
	assert.NotNil(t, mf)

	pb := &dto.Metric{}
	err = mf.Write(pb)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, pb.GetCounter().GetValue())

	mfTime, err := metrics.HTTPResponseTime.MetricVec.GetMetricWithLabelValues("GET", "/test")
	assert.NoError(t, err)
	assert.NotNil(t, mfTime)
}
