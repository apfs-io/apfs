// Package metacache provides a caching layer for object metadata (meta.json).
// It sits between the high-level storage layer and the underlying driver to
// avoid repeated serialisation/deserialisation on every Head/Get request.
package metacache

import (
	"context"
	"time"

	"github.com/apfs-io/apfs/models"
)

// DefaultTTL is the default cache entry lifetime.
const DefaultTTL = 5 * time.Minute

// MetaCache stores and retrieves *models.Meta by object ID string.
type MetaCache interface {
	// Get returns the cached Meta, or (nil, nil) on a cache miss.
	Get(ctx context.Context, id string) (*models.Meta, error)

	// Set stores a Meta entry with the given TTL.
	// A zero TTL uses the implementation's default.
	Set(ctx context.Context, id string, meta *models.Meta, ttl time.Duration) error

	// Invalidate removes the cached entry for id.
	Invalidate(ctx context.Context, id string) error
}
