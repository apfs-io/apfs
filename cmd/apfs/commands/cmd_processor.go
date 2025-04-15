package commands

import (
	"context"
	"fmt"
	"runtime"

	nc "github.com/geniusrabbit/notificationcenter/v2"
	"github.com/geniusrabbit/notificationcenter/v2/wrappers/concurrency"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/apfs-io/apfs/cmd/apfs/appcontext"
	"github.com/apfs-io/apfs/cmd/apfs/appinit"
	"github.com/apfs-io/apfs/internal/context/ctxlogger"
	"github.com/apfs-io/apfs/internal/stream"
)

// Event stream name initialized from appinit.
var eventStream = appinit.EventStreamName

// processorConfig holds the configuration for the processor command.
type processorConfig struct {
	Storage     appcontext.StorageConfig     `json:"storage" yaml:"storage"`
	Eventstream appcontext.EventstreamConfig `json:"eventstream" yaml:"eventstream"`
}

// ProcessorCommand defines the CLI command for running the processor.
var ProcessorCommand = &Command[processorConfig]{
	Name:     "processor",
	HelpDesc: "Run processor",
	Exec:     processorCommandExec,
}

// processorCommandExec is the main execution function for the processor command.
func processorCommandExec(ctx context.Context, args []string, config *processorConfig) error {
	// Retrieve the logger from the context.
	logger := ctxlogger.Get(ctx)

	// Initialize the protocol API object with eventstream, storage, and logger configurations.
	protoAPI, err := appinit.ProtocolAPIObject(ctx,
		&config.Eventstream, &config.Storage, logger)
	fatalError(err, "protocol initialization")

	// Execute the processor logic.
	return runProcessor(ctx, &config.Eventstream, &config.Storage,
		protoAPI.(nc.Receiver), logger)
}

// runProcessor handles the core processing logic for the processor command.
func runProcessor(
	ctx context.Context,
	eventsConf *appcontext.EventstreamConfig,
	storageConf *appcontext.StorageConfig,
	reveiver nc.Receiver,
	logger *zap.Logger,
) error {
	// Log the start of the processor.
	fmt.Println(" ⇛ Run apfs file processor")
	fmt.Println("storage connect:", storageConf.Connect)

	// Register the notification stream.
	events, err := stream.NewReader(ctx, eventsConf.Connect)
	if err != nil {
		return errors.Wrap(err, "connect to: "+eventsConf.Connect)
	}
	if err = nc.Register(eventStream, events); err != nil {
		return err
	}

	// Set default concurrency if not configured.
	if eventsConf.Concurrency <= 0 {
		eventsConf.Concurrency = runtime.NumCPU()
	}

	// Create a concurrent pool wrapper for the receiver.
	conncurrencyReceiver := concurrency.WithWorkers(
		reveiver, eventsConf.Concurrency, nil,
		concurrency.WithWorkerPoolSize(eventsConf.PoolSize),
		concurrency.WithRecoverHandler(func(err any) {
			// Handle errors during processing.
			switch e := err.(type) {
			case error:
				logger.Error(`processing pool recovery`, zap.Error(e))
			default:
				logger.Error(`processing pool recovery`, zap.Any(`error`, err))
			}
		}),
	)

	// Subscribe to the event stream.
	if err = nc.Subscribe(ctx, eventStream, conncurrencyReceiver); err != nil {
		return errors.Wrap(err, "subscribe processor handler")
	}

	// Start listening for events.
	fmt.Println("Run listener:", eventsConf.Connect)
	return nc.Listen(ctx)
}
