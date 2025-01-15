package middleware

import (
	"context"
	"time"

	"google.golang.org/grpc"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricsGRPCCount *prometheus.CounterVec
	metricGRPCTiming *prometheus.HistogramVec
)

func init() {
	buckets := []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
	metricsGRPCCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "grpc_request_count",
		Help: "Count of requests by method",
	}, []string{"method", "grpc_error"})
	metricGRPCTiming = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "grpc",
		Name:      "grpc_request_duration_seconds",
		Help:      "Histogram of response time for handler in seconds",
		Buckets:   buckets,
	}, []string{"method", "grpc_error"})
}

// GRPCContextUnaryWrapper implements wrapper of unary handler with context overriding
func GRPCContextUnaryWrapper(ctxWrapper func(ctx context.Context) (context.Context, error)) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		var (
			start  = time.Now()
			newCtx context.Context
		)
		defer func() {
			metricsGRPCCount.WithLabelValues(info.FullMethod, oneStrIf(err != nil)).Inc()
			metricGRPCTiming.WithLabelValues(info.FullMethod, oneStrIf(err != nil)).
				Observe(time.Since(start).Seconds())
		}()
		if newCtx, err = ctxWrapper(ctx); err != nil {
			return nil, err
		}
		return handler(newCtx, req)
	}
}

// GRPCContextStreamWrapper implements wrapper of stream handler with context overriding
func GRPCContextStreamWrapper(ctxWrapper func(ctx context.Context) (context.Context, error)) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		start := time.Now()
		defer func() {
			metricsGRPCCount.WithLabelValues(info.FullMethod, oneStrIf(err != nil)).Inc()
			metricGRPCTiming.WithLabelValues(info.FullMethod, oneStrIf(err != nil)).
				Observe(time.Since(start).Seconds())
		}()
		newStream := grpc_middleware.WrapServerStream(ss)
		newStream.WrappedContext, err = ctxWrapper(ss.Context())
		if err != nil {
			return err
		}
		return handler(srv, ss)
	}
}

func oneStrIf(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
