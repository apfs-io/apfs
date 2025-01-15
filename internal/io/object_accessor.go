package io

import (
	"context"
	"io"
	"net/url"

	"github.com/apfs-io/apfs/models"
)

// ObjectAccessor to the objects storage
type ObjectAccessor interface {
	// Create new object object
	// ID is optional, can be defined the custom ID for strictly mapped data
	Create(ctx context.Context, bucket string, id ObjectID, overwrite bool, params url.Values) (Object, error)

	// UpdatePatams in the object. If name is present then update only params linked with the subobject
	UpdatePatams(ctx context.Context, id ObjectID, params url.Values) error

	// Open existing object
	// path example: images/a/b/c/d
	Open(ctx context.Context, id ObjectID) (Object, error)

	// Read returns reader of the specific internal object
	Read(ctx context.Context, id ObjectID, name string) (io.ReadCloser, error)

	// Update data in the storage
	Update(ctx context.Context, id ObjectID, name string, data io.Reader, meta *models.ItemMeta) error

	// Update data in the storage
	UpdateMeta(ctx context.Context, id ObjectID, name string, meta *models.ItemMeta) error

	// Clean removes all internal data from object except original
	Clean(ctx context.Context, id ObjectID) error

	// Remove object with ID
	Remove(ctx context.Context, id ObjectID, names ...string) error
}
