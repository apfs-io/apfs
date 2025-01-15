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
func ProtocolAPIObject(ctx context.Context, conf *appcontext.ConfigType, logger *zap.Logger) (api.ServiceServer, error) {
	// Register the notification stream
	events, err := registerStream(ctx, EventStreamName, conf.Eventstream.Connect)
	if err != nil {
		return nil, err
	}
	srvLogic, err := api.NewServer(
		conf.Storage.MetadbConnect,
		conf.Storage.Connect,
		conf.Storage.StateConnect,
		api.WithStageProcessingLimit(conf.Storage.ProcessingStageLimit),
		api.WithTaskProcessingLimit(conf.Storage.ProcessingTaskLimit),
		api.WithEventstream(events),
		api.WithUpdateState(updateLocker(conf)),
		api.WithStorageConverters(Converters(conf, logger)),
		api.WithRetries(conf.Storage.ProcessingMaxRetries),
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
