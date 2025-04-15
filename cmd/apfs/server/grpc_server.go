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

// GRPCServer is a wrapper object that encapsulates the gRPC and HTTP server configurations and logic.
type GRPCServer struct {
	API         *v1.ServerHTTPWrapper // API implementation for handling requests.
	ContextWrap contextWrapper        // Function to wrap context with additional data.
	Logger      *zap.Logger           // Logger for logging server events.

	Concurrency       uint32        // Maximum number of concurrent streams.
	RequestTimeout    time.Duration // Timeout for individual requests.
	ConnectionTimeout time.Duration // Timeout for connections.

	// Secure connection certificates
	CertFile string // Path to the certificate file.
	KeyFile  string // Path to the key file.
}

// Run starts the gRPC and/or HTTP server based on the provided addresses.
func (s *GRPCServer) Run(ctx context.Context, httpAddr, grpcAddr string) error {
	// If both HTTP and gRPC addresses are provided, run them concurrently.
	if httpAddr != "" && grpcAddr != "" {
		go func() {
			if err := s.RunGRPC(ctx, grpcAddr); err != nil {
				s.Logger.Error("failed to run GRPC server",
					zap.String("address", grpcAddr),
					zap.Error(err))
			}
		}()
		return s.RunHTTP(ctx, httpAddr)
	}
	// Run only the HTTP server if the HTTP address is provided.
	if httpAddr != "" {
		return s.RunHTTP(ctx, httpAddr)
	}
	// Run only the gRPC server if the gRPC address is provided.
	if grpcAddr != "" {
		return s.RunGRPC(ctx, grpcAddr)
	}
	// Return an error if no address is provided.
	return errors.New("no server address provided")
}

// RunGRPC starts the gRPC server on the specified address.
func (s *GRPCServer) RunGRPC(ctx context.Context, listen string) error {
	// Parse the network and address from the listen string.
	network, address := parseNetwork(listen)
	s.Logger.Info("Start GRPC API",
		zap.String("network", network),
		zap.String("address", address))

	// Create a listener for the specified network and address.
	lis, err := (&net.ListenConfig{}).
		Listen(ctx, network, address)
	if err != nil {
		return err
	}

	// Configure logging options for gRPC.
	zapOpts := []grpc_zap.Option{
		grpc_zap.WithDurationField(func(duration time.Duration) zapcore.Field {
			return zap.Int64("grpc.time_ns", duration.Nanoseconds())
		}),
	}

	// Initialize TLS credentials if certificate and key files are provided.
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

	// Create the gRPC server instance with middleware and configurations.
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

	// Register the service API with the gRPC server.
	protocol.RegisterServiceAPIServer(srv, s.API)

	// Gracefully stop the server when the context is canceled.
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

// RunHTTP starts the HTTP server on the specified address.
func (s *GRPCServer) RunHTTP(ctx context.Context, address string) error {
	s.Logger.Info("Start HTTP API: " + address)

	// Create a new gRPC-Gateway multiplexer.
	gw := runtime.NewServeMux()
	if err := protocol.RegisterServiceAPIHandlerServer(ctx, gw, s.API); err != nil {
		return err
	}

	// Set up the HTTP router with middleware and routes.
	mux := chi.NewRouter()
	mux.Use(chimiddleware.RequestID)
	mux.Use(chimiddleware.RealIP)
	mux.Use(chimiddleware.Logger)
	mux.Use(chimiddleware.Recoverer)

	// Define routes for the API and additional endpoints.
	mux.Handle("/*", gw)
	mux.Get("/object", s.API.GetHTTPHandler)
	mux.Get("/object/*", s.API.GetHTTPHandler)
	mux.Post("/object", s.API.UploadHTTPHandler)
	mux.Post("/object/{group}", s.API.UploadHTTPHandler)
	mux.Handle("/swagger/", s.swaggerHandler())
	mux.HandleFunc("/health", tools.HealthCheck)
	mux.Handle("/metrics", promhttp.Handler())

	// Wrap the router with context and metrics middleware.
	handler := middleware.HTTPContextWrapper(mux, s.ContextWrap)

	// Create the HTTP server instance.
	srv := &http.Server{
		Addr:        address,
		Handler:     handler,
		BaseContext: func(l net.Listener) context.Context { return ctx },
		ConnContext: s.connContextWrapFnk(),
	}

	// Gracefully shut down the server when the context is canceled.
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

// swaggerHandler serves the Swagger UI for API documentation.
func (s *GRPCServer) swaggerHandler() http.Handler {
	return http.StripPrefix("/swagger/",
		tools.SwaggerServer("/swagger/swagger.json", true))
}

// contextWrapFnk wraps the context with additional data for gRPC.
func (s *GRPCServer) contextWrapFnk() func(ctx context.Context) (context.Context, error) {
	return func(ctx context.Context) (context.Context, error) {
		return s.ContextWrap(ctx), nil
	}
}

// connContextWrapFnk wraps the connection context with additional data.
func (s *GRPCServer) connContextWrapFnk() func(ctx context.Context, c net.Conn) context.Context {
	return func(ctx context.Context, c net.Conn) context.Context {
		return s.ContextWrap(ctx)
	}
}

// parseNetwork parses the network and address from a URI.
func parseNetwork(uri string) (network, address string) {
	if !strings.Contains(uri, "://") {
		return "tcp", uri
	}
	addr := strings.SplitN(uri, "://", 2)
	return addr[0], strings.TrimLeft(addr[1], "/")
}

// loadCreds loads TLS credentials from the provided certificate and key files.
func loadCreds(crt, key string) (credentials.TransportCredentials, error) {
	if crt == `` {
		return insecure.NewCredentials(), nil
	}
	return credentials.NewServerTLSFromFile(crt, key)
}
