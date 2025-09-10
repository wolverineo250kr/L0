package middleware

import (
	"net/http"
	"time"

	"order-service/internal/metrics"

	"github.com/gorilla/mux"
)

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// создаем для перехвата статуса
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()

		// получаем путь из роутера
		path := r.URL.Path
		if route := mux.CurrentRoute(r); route != nil {
			if tpl, err := route.GetPathTemplate(); err == nil && tpl != "" {
				path = tpl
			}
		}

		metrics.HTTPRequests.WithLabelValues(
			r.Method,
			path,
			http.StatusText(rw.statusCode),
		).Inc()

		metrics.HTTPResponseTime.WithLabelValues(
			r.Method,
			path,
		).Observe(duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
