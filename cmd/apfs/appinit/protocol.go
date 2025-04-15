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
func ProtocolAPIObject(ctx context.Context, eventsConf *appcontext.EventstreamConfig, storageConf *appcontext.StorageConfig, logger *zap.Logger) (api.ServiceServer, error) {
	// Register the notification stream
	events, err := registerStream(ctx, EventStreamName, eventsConf.Connect)
	if err != nil {
		return nil, err
	}
	srvLogic, err := api.NewServer(
		storageConf.MetadbConnect,
		storageConf.Connect,
		storageConf.StateConnect,
		api.WithStageProcessingLimit(storageConf.ProcessingStageLimit),
		api.WithTaskProcessingLimit(storageConf.ProcessingTaskLimit),
		api.WithEventstream(events),
		api.WithUpdateState(updateLocker(storageConf)),
		api.WithStorageConverters(Converters(storageConf, logger)),
		api.WithRetries(storageConf.ProcessingMaxRetries),
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
