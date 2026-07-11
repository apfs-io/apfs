package client

import (
	"context"
	"encoding/json"
	"io"

	nc "github.com/geniusrabbit/notificationcenter/v2"

	"github.com/apfs-io/apfs/internal/context/ctxstatusstream"
	"github.com/apfs-io/apfs/internal/stream"
)

// ProcessingStatusEvent is the per-task progress message emitted by the
// processing pipeline. Fields mirror state.proto counters + object_id.
type ProcessingStatusEvent = ctxstatusstream.ProcessingStatusEvent

// ProcessingHandler is called for each status event received from the stream.
type ProcessingHandler func(ctx context.Context, event *ProcessingStatusEvent) error

// ProcessingStream subscribes to the processing status stream published by
// the APFS processor.
type ProcessingStream struct {
	sub nc.Subscriber
}

// NewProcessingStream connects to the status stream at connURL and returns
// a ProcessingStream ready to subscribe.
func NewProcessingStream(ctx context.Context, connURL string) (*ProcessingStream, error) {
	sub, err := stream.NewReader(ctx, connURL)
	if err != nil {
		return nil, err
	}
	return &ProcessingStream{sub: sub}, nil
}

// Subscribe registers h to be called for every ProcessingStatusEvent.
// The subscription is active until ctx is cancelled or Listen returns.
func (ps *ProcessingStream) Subscribe(ctx context.Context, h ProcessingHandler) error {
	return ps.sub.Subscribe(ctx, nc.FuncReceiver(func(msg nc.Message) error {
		var event ProcessingStatusEvent
		if err := json.Unmarshal(msg.Body(), &event); err != nil {
			return err
		}
		if err := h(msg.Context(), &event); err != nil {
			return err
		}
		return msg.Ack()
	}))
}

// Listen blocks until ctx is cancelled, processing incoming events.
func (ps *ProcessingStream) Listen(ctx context.Context) error {
	return ps.sub.Listen(ctx)
}

// Close releases the underlying subscriber connection.
func (ps *ProcessingStream) Close() error {
	if closer, _ := ps.sub.(io.Closer); closer != nil {
		return closer.Close()
	}
	return nil
}

// SubscribeProcessing is a convenience function that connects, subscribes,
// and listens in one call. It blocks until ctx is cancelled or an error occurs.
//
//	err := client.SubscribeProcessing(ctx, "nats://host:4222/apfs?topics=status",
//	    func(ctx context.Context, e *client.ProcessingStatusEvent) error {
//	        fmt.Println(e.ObjectID, e.Completed, "/", e.Total)
//	        return nil
//	    })
func SubscribeProcessing(ctx context.Context, connURL string, handlers ...ProcessingHandler) error {
	if len(handlers) == 0 {
		return nil
	}
	ps, err := NewProcessingStream(ctx, connURL)
	if err != nil {
		return err
	}
	defer func() { _ = ps.Close() }()
	for _, h := range handlers {
		if err := ps.Subscribe(ctx, h); err != nil {
			return err
		}
	}
	return ps.Listen(ctx)
}
