package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/apfs-io/apfs/internal/middleware"
	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	v1 "github.com/apfs-io/apfs/internal/server/v1"
	"github.com/apfs-io/apfs/internal/server/v1/tools"
)

type contextWrapper func(context.Context) context.Context

// GRPCServer wrapper object
type GRPCServer struct {
	API         *v1.ServerHTTPWrapper
	ContextWrap contextWrapper
	Logger      *zap.Logger

	Concurrency       uint32
	RequestTimeout    time.Duration
	ConnectionTimeout time.Duration

	// Secure connection certificates
	CertFile string
	KeyFile  string
}

// RunGRPC server
func (s *GRPCServer) RunGRPC(ctx context.Context, listen string) error {
	network, address := parseNetwork(listen)
	s.Logger.Info("Start GRPC API",
		zap.String("network", network),
		zap.String("address", address))

	lis, err := (&net.ListenConfig{}).
		Listen(ctx, network, address)
	if err != nil {
		return err
	}

	zapOpts := []grpc_zap.Option{
		grpc_zap.WithDurationField(func(duration time.Duration) zapcore.Field {
			return zap.Int64("grpc.time_ns", duration.Nanoseconds())
		}),
	}

	// Init certification
	creds, err := loadCreds(s.CertFile, s.KeyFile)
	if err != nil {
		if closeErr := lis.Close(); err != nil {
			s.Logger.Error("failed to close",
				zap.String(`network`, network),
				zap.String(`address`, address),
				zap.Error(closeErr))
		}
		return errors.Wrap(err, `failed to setup TLS:`)
	}

	// Create server instance
	srv := grpc.NewServer(
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(
			middleware.GRPCErrorUnaryWrapper,
			grpc_zap.UnaryServerInterceptor(s.Logger, zapOpts...),
			middleware.GRPCContextUnaryWrapper(s.contextWrapFnk()),
		),
		grpc.ChainStreamInterceptor(
			middleware.GRPCErrorStreamWrapper,
			grpc_zap.StreamServerInterceptor(s.Logger, zapOpts...),
			middleware.GRPCContextStreamWrapper(s.contextWrapFnk()),
		),
		grpc.MaxConcurrentStreams(s.Concurrency),
		grpc.ConnectionTimeout(s.ConnectionTimeout),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: s.ConnectionTimeout * 2,
			Time:              s.ConnectionTimeout * 2,
			Timeout:           s.ConnectionTimeout,
		}),
	)

	// Register service API
	protocol.RegisterServiceAPIServer(srv, s.API)

	go func() {
		<-ctx.Done()
		srv.GracefulStop()
		if closeErr := lis.Close(); err != nil {
			s.Logger.Error("failed to close",
				zap.String(`network`, network),
				zap.String(`address`, address),
				zap.Error(closeErr))
		}
	}()

	s.Logger.Info("Starting GRPC listening", zap.String("listen", listen))
	return srv.Serve(lis)
}

// RunHTTP server
func (s *GRPCServer) RunHTTP(ctx context.Context, address string) error {
	s.Logger.Info("Start HTTP API: " + address)

	gw := runtime.NewServeMux()
	if err := protocol.RegisterServiceAPIHandlerServer(ctx, gw, s.API); err != nil {
		return err
	}

	mux := chi.NewRouter()
	mux.Use(chimiddleware.RequestID)
	mux.Use(chimiddleware.RealIP)
	mux.Use(chimiddleware.Logger)
	mux.Use(chimiddleware.Recoverer)

	mux.Handle("/*", gw)
	mux.Get("/object", s.API.GetHTTPHandler)
	mux.Get("/object/*", s.API.GetHTTPHandler)
	mux.Post("/object", s.API.UploadHTTPHandler)
	mux.Post("/object/{group}", s.API.UploadHTTPHandler)
	mux.Handle("/swagger/", s.swaggerHandler())
	mux.HandleFunc("/health", tools.HealthCheck)
	mux.Handle("/metrics", promhttp.Handler())

	// Context and metrics wrapper
	h := middleware.HTTPContextWrapper(mux, s.ContextWrap)

	srv := &http.Server{
		Addr:    address,
		Handler: h,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			return s.ContextWrap(ctx)
		},
	}

	go func() {
		<-ctx.Done()
		s.Logger.Info("Shutting down the HTTP server")
		if err := srv.Shutdown(context.Background()); err != nil {
			s.Logger.Error("Failed to shutdown HTTP server", zap.Error(err))
		}
	}()

	s.Logger.Info(fmt.Sprintf("Starting listening at %s", address))
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		s.Logger.Error("Failed to listen and serve", zap.Error(err))
		return err
	}
	return nil
}

func (s *GRPCServer) swaggerHandler() http.Handler {
	return http.StripPrefix("/swagger/",
		tools.SwaggerServer("/swagger/swagger.json", true),
	)
}

func (s *GRPCServer) contextWrapFnk() func(ctx context.Context) (context.Context, error) {
	return func(ctx context.Context) (context.Context, error) {
		return s.ContextWrap(ctx), nil
	}
}

func parseNetwork(uri string) (network, address string) {
	if !strings.Contains(uri, "://") {
		return "tcp", uri
	}
	addr := strings.SplitN(uri, "://", 2)
	return addr[0], strings.TrimLeft(addr[1], "/")
}

func loadCreds(crt, key string) (credentials.TransportCredentials, error) {
	if crt == `` {
		return insecure.NewCredentials(), nil
	}
	return credentials.NewServerTLSFromFile(crt, key)
}
