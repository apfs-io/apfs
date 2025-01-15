package stream

import (
	"context"
	"strings"
	"time"

	nc "github.com/geniusrabbit/notificationcenter/v2"
	"github.com/geniusrabbit/notificationcenter/v2/kafka"
	"github.com/geniusrabbit/notificationcenter/v2/nats"
	natsio "github.com/nats-io/nats.go"
)

// NewReader stream
func NewReader(ctx context.Context, urlStr string) (nc.Subscriber, error) {
	switch {
	case strings.HasPrefix(urlStr, "nats://"):
		// nats://localhost:4222/group?topics={topic_name1},{topic_name2}
		return nats.NewSubscriber(nats.WithNatsURL(urlStr), nats.WithNatsOptions(natsio.ReconnectWait(time.Second*5)))
	case strings.HasPrefix(urlStr, "kafka://"):
		// kafka://broker1:9092,broker2:9092/group?client_id={service_name}&topics={topic_name1},{topic_name2}
		return kafka.NewSubscriber(kafka.WithKafkaURL(urlStr))
	}
	return nil, ErrUndefinedStreamScheme
}
