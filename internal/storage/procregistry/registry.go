// Package procregistry tracks objects whose processing is still in flight.
package procregistry

import (
	"context"
	"strings"
	"time"
)

const inflightKey = "apfs:inflight"

// Registry is an index of objects currently being processed.
type Registry interface {
	// Add records objectID as in-flight (creates or refreshes the timestamp).
	Add(ctx context.Context, objectID string) error
	// Remove drops objectID from the index.
	Remove(ctx context.Context, objectID string) error
	// ListOlderThan returns object IDs whose last Add is older than age.
	ListOlderThan(ctx context.Context, age time.Duration) ([]string, error)
	// TryBegin marks objectID as actively processed in this process.
	// Returns false if another goroutine already holds it.
	TryBegin(objectID string) bool
	// End clears the in-process busy flag (does not Remove from the index).
	End(objectID string)
	// IsBusy reports whether this process currently holds objectID.
	IsBusy(objectID string) bool
}

// New builds a Registry from STORAGE_STATE_CONNECT.
// "memory" or empty → in-process map. redis:// / tcp:// → Redis HASH index.
func New(connect string) (Registry, error) {
	switch {
	case connect == "" || connect == "memory":
		return NewMemory(), nil
	case strings.HasPrefix(connect, "redis://") || strings.HasPrefix(connect, "tcp://"):
		return NewRedis(connect)
	default:
		return NewMemory(), nil
	}
}
