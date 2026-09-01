package appinit

import (
	"context"

	nc "github.com/geniusrabbit/notificationcenter/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/apfs-io/apfs/cmd/apfs/appcontext"
	api "github.com/apfs-io/apfs/internal/server/v1"
	"github.com/apfs-io/apfs/internal/stream"
)

// EventStreamName for income events
const EventStreamName = "events"

// ProtocolAPIObject inites the API implementation
func ProtocolAPIObject(
	ctx context.Context,
	eventsConf *appcontext.EventstreamConfig,
	storageConf *appcontext.StorageConfig,
	workflowsConf *appcontext.WorkflowsConfig,
	processingConf *appcontext.ProcessingConfig,
	workerTags []string,
	logger *zap.Logger,
) (api.ServiceServer, error) {
	// Register the notification stream
	events, err := registerStream(ctx, EventStreamName, eventsConf.Connect)
	if err != nil {
		return nil, err
	}

	opts := []api.Option{
		api.WithTaskProcessingLimit(processingConf.TaskLimit),
		api.WithEventstream(events),
		api.WithUpdateState(updateLocker(processingConf)),
		api.WithWorkflowExecutor(StepRunners(ctx, storageConf, logger)),
		api.WithWorkerTags(workerTags),
		api.WithWorkflowsBootstrap(workflowsConf.Dir, workflowsConf.Reconfigure),
		api.WithEnsureBucket(storageConf.EnsureBucket),
	}

	// Connect the optional status stream for per-task progress events.
	if processingConf.StatusStream.Connect != "" {
		statusStream, err := stream.NewWriter(ctx, processingConf.StatusStream.Connect)
		if err != nil {
			return nil, errors.Wrap(err, "connect to status stream: "+processingConf.StatusStream.Connect)
		}
		opts = append(opts, api.WithStatusStream(statusStream))
		logger.Info("processing status stream enabled",
			zap.String("url", processingConf.StatusStream.Connect))
	} else {
		logger.Info("processing status stream disabled")
	}

	srvLogic, err := api.NewServer(ctx,
		storageConf.MetadbConnect,
		storageConf.Connect,
		storageConf.StateConnect,
		opts...,
	)
	if err != nil {
		return nil, errors.Wrap(err, "server create")
	}
	return srvLogic, nil
}

func registerStream(ctx context.Context, name, connect string) (nc.Publisher, error) {
	stream, err := stream.NewWriter(ctx, connect)
	if err != nil {
		return nil, errors.Wrap(err, "connect to: "+connect)
	}
	if err = nc.Register(name, stream); err != nil {
		return nil, err
	}
	return stream, nil
}
