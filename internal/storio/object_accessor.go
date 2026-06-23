package storio

import (
	"context"
	"io"
	"net/url"

	"github.com/apfs-io/apfs/models"
)

// ObjectAccessor manages the lifecycle of top-level objects in the storage.
type ObjectAccessor interface {
	// Create allocates a new object in the given bucket.
	// id is optional; pass nil to generate one automatically.
	// Set overwrite to replace an existing object with the same id.
	Create(ctx context.Context, bucket string, id ObjectID, overwrite bool, params url.Values) (Object, error)

	// UpdateParams applies key/value params to the object (or a named subfile).
	UpdateParams(ctx context.Context, id ObjectID, params url.Values) error

	// Open returns a handle for an existing object. Returns a not-found error
	// when the object does not exist.
	Open(ctx context.Context, id ObjectID) (Object, error)

	// Read returns a reader for the named subfile within the object.
	// Use "@" or "" for the original/primary file.
	Read(ctx context.Context, id ObjectID, name string) (io.ReadCloser, error)

	// Update writes data for the named subfile and updates its ItemMeta.
	Update(ctx context.Context, id ObjectID, name string, data io.Reader, meta *models.ItemMeta) error

	// UpdateMeta updates only the ItemMeta for the named subfile without
	// touching its content.
	UpdateMeta(ctx context.Context, id ObjectID, name string, meta *models.ItemMeta) error

	// Clean removes all derived subfiles from the object, leaving only the
	// original file and the meta/state records.
	Clean(ctx context.Context, id ObjectID) error

	// Remove deletes the object and optionally specific named subfiles.
	// With no names the entire object directory is removed.
	Remove(ctx context.Context, id ObjectID, names ...string) error
}
