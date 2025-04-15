package client

import (
	"context"
	"encoding/json"
	"io"

	nc "github.com/geniusrabbit/notificationcenter/v2"

	"github.com/apfs-io/apfs/internal/stream"
	"github.com/apfs-io/apfs/models"
)

// EventHandler function callback
type EventHandler func(models.EventType, *models.Object)

// Eventstream client to process events of of the service like delete, update
type Eventstream struct {
	sub nc.Subscriber
}

// NewEventStream event subscriber
func NewEventStream(ctx context.Context, connect string) (*Eventstream, error) {
	sub, err := stream.NewReader(ctx, connect)
	if err != nil {
		return nil, err
	}
	return &Eventstream{sub: sub}, nil
}

// Subscribe new event handler
func (es *Eventstream) Subscribe(ctx context.Context, h EventHandler) error {
	return es.sub.Subscribe(ctx, nc.FuncReceiver(func(msg nc.Message) error {
		var event models.Event
		if err := json.Unmarshal(msg.Body(), &event); err != nil {
			return err
		}
		h(event.Type, event.Object)
		return msg.Ack()
	}))
}

// Listen processing queue
func (es *Eventstream) Listen(ctx context.Context) error {
	return es.sub.Listen(ctx)
}

// Close the eventstream
func (es *Eventstream) Close() error {
	if closer, _ := es.sub.(io.Closer); closer != nil {
		return closer.Close()
	}
	return nil
}
