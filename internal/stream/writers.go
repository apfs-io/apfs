package stream

import (
	"context"
	"errors"
	"strings"
	"time"

	nc "github.com/geniusrabbit/notificationcenter/v2"
	"github.com/geniusrabbit/notificationcenter/v2/kafka"
	"github.com/geniusrabbit/notificationcenter/v2/nats"
	natsio "github.com/nats-io/nats.go"
)

// Error list...
var (
	ErrUndefinedStreamScheme = errors.New(`[stream] invalid scheme`)
)

// NewWriter publisher interface
func NewWriter(ctx context.Context, urlStr string) (nc.Publisher, error) {
	switch {
	case strings.HasPrefix(urlStr, "nats://"):
		// nats://broker1:9092,broker2:9092/group?client_id={service_name}&topics={topic_name1},{topic_name2}
		return nats.NewPublisher(nats.WithNatsURL(urlStr), nats.WithNatsOptions(natsio.ReconnectWait(time.Second*5)))
	case strings.HasPrefix(urlStr, "kafka://"):
		// kafka://broker1:9092,broker2:9092/group?client_id={service_name}&topics={topic_name1},{topic_name2}
		return kafka.NewPublisher(ctx, kafka.WithKafkaURL(urlStr))
	}
	return nil, ErrUndefinedStreamScheme
}
