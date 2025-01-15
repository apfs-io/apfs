package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"

	"github.com/apfs-io/apfs/internal/context/ctxlogger"
)

// HTTPContextWrapper global middelware of the HTTP rounter
func HTTPContextWrapper(h http.Handler, ctxWrapper func(ctx context.Context) context.Context) http.Handler {
	var (
		buckets      = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
		metricsCount = promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "http_request_count",
			Help: "Count of requests by country",
		}, []string{"method", "http_path"})
		metricTiming = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "http",
			Name:      "http_request_duration_seconds",
			Help:      "Histogram of response time for handler in seconds",
			Buckets:   buckets,
		}, []string{"method", "http_path"})
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		metricsCount.WithLabelValues(r.Method, r.URL.Path).Inc()
		newCtx := ctxWrapper(r.Context())
		ctxlogger.Get(newCtx).Info("HTTP",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path))

		h.ServeHTTP(w, r.WithContext(newCtx))

		duration := time.Since(start)
		metricTiming.WithLabelValues(r.Method, r.URL.Path).
			Observe(duration.Seconds())
	})
}
