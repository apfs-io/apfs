// Package ctxstatusstream provides context-based access to the processing
// status stream publisher. The publisher is optional — when not configured
// all Publish calls are no-ops, so callers need no nil guards.
package ctxstatusstream

import (
	"context"

	nc "github.com/geniusrabbit/notificationcenter/v2"

	"github.com/apfs-io/apfs/models"
)

type ctxKey struct{}

// ProcessingStatusEvent is the compact progress message published to the
// status stream after each processing task and as a final summary when all
// tasks complete. Fields mirror state.proto: object_id + ProcessingCounters
// + optional error.
type ProcessingStatusEvent struct {
	ObjectID  string                  `json:"object_id"`
	Status    models.ProcessingStatus `json:"status"`
	Progress  float64                 `json:"progress"` // 0.0–1.0
	Total     int                     `json:"total"`
	Completed int                     `json:"completed"`
	Failed    int                     `json:"failed"`
	Skipped   int                     `json:"skipped"`
	Pending   int                     `json:"pending"`
	Error     string                  `json:"error,omitempty"`
	// Final is true for the last event of an object (processing finished or failed).
	Final bool `json:"final"`
}

// Get returns the status stream publisher stored in ctx, or nil if not set.
func Get(ctx context.Context) nc.Publisher {
	if v := ctx.Value(ctxKey{}); v != nil {
		return v.(nc.Publisher)
	}
	return nil
}

// WithPublisher stores pub in ctx and returns the derived context.
func WithPublisher(ctx context.Context, pub nc.Publisher) context.Context {
	return context.WithValue(ctx, ctxKey{}, pub)
}

// Publish sends event to the status stream. It is a no-op when no publisher
// is stored in ctx (optional connection).
func Publish(ctx context.Context, event *ProcessingStatusEvent) error {
	pub := Get(ctx)
	if pub == nil {
		return nil
	}
	return pub.Publish(ctx, event)
}
