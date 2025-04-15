package commands

import (
	"context"
	"fmt"

	nc "github.com/geniusrabbit/notificationcenter/v2"

	"github.com/apfs-io/apfs/cmd/apfs/appcontext"
	"github.com/apfs-io/apfs/cmd/apfs/appinit"
	"github.com/apfs-io/apfs/cmd/apfs/server"
	"github.com/apfs-io/apfs/internal/context/ctxlogger"
	v1 "github.com/apfs-io/apfs/internal/server/v1"
)

// serverConfig defines the configuration structure for the server.
type serverConfig struct {
	Processing bool `cli:"processing"`

	Server      appcontext.ServerConfig      `json:"server" yaml:"server"`
	Storage     appcontext.StorageConfig     `json:"storage" yaml:"storage"`
	Eventstream appcontext.EventstreamConfig `json:"eventstream" yaml:"eventstream"`
}

// ServerCommand defines the CLI command for running the server.
var ServerCommand = &Command[serverConfig]{
	Name:     "server",
	HelpDesc: "Run server",
	Exec:     serverCommandExec,
}

// serverCommandExec is the main execution function for the server command.
func serverCommandExec(ctx context.Context, args []string, config *serverConfig) error {
	// Retrieve the logger from the context.
	logger := ctxlogger.Get(ctx)

	// Initialize the protocol API object with eventstream, storage, and logger configurations.
	protoAPI, err := appinit.ProtocolAPIObject(ctx,
		&config.Eventstream, &config.Storage, logger)
	fatalError(err, "protocol initialization")

	// Run the processor if the Processing flag is set.
	if config.Processing {
		go func() {
			fatalError(runProcessor(ctx,
				&config.Eventstream, &config.Storage, protoAPI.(nc.Receiver), logger))
		}()
	}

	// Wrap the protocol API with an HTTP wrapper.
	srvAPI := v1.NewHTTPWrapper(protoAPI)

	// Print server initialization messages.
	fmt.Println(" ⇛ Run apfs gRPC server")
	fmt.Println(" ⇛ storage connect:", config.Storage.Connect)

	// Define a context wrapper to inject the logger into the context.
	ctxWrp := func(ctx context.Context) context.Context {
		return ctxlogger.WithLogger(ctx, logger)
	}

	// Initialize the gRPC server with the provided configurations.
	srv := &server.GRPCServer{
		API:               srvAPI,
		ContextWrap:       ctxWrp,
		Logger:            logger,
		Concurrency:       config.Server.GRPC.Concurrency, // Set gRPC concurrency level.
		RequestTimeout:    config.Server.GRPC.Timeout,     // Set request timeout.
		ConnectionTimeout: config.Server.GRPC.Timeout,     // Set connection timeout.
		CertFile:          "",                             // Placeholder for certificate file.
		KeyFile:           "",                             // Placeholder for key file.
	}

	// Run the server with the specified HTTP and gRPC listen addresses.
	return srv.Run(ctx, config.Server.HTTP.Listen, config.Server.GRPC.Listen)
}
