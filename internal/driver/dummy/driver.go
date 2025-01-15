package dummy

import (
	"context"
	"io"
	"net/url"

	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/models"
)

// Driver with fummy declaration
type Driver struct {
}

// Create new object object
// If manifest is Nil, than will be used dfault manifest
func (d *Driver) Create(ctx context.Context, bucket string, id npio.ObjectID, overwrite bool, params url.Values) (npio.Object, error) {
	return nil, nil
}

// UpdatePatams in the object. If name is present then update only params linked with the subobject
func (d *Driver) UpdatePatams(ctx context.Context, id npio.ObjectID, params url.Values) error {
	return nil
}

// Open existing object
// path example: images/a/b/c/d
func (d *Driver) Open(ctx context.Context, id npio.ObjectID) (npio.Object, error) {
	return nil, nil
}

// Read returns reader of the specific internal object
func (d *Driver) Read(ctx context.Context, id npio.ObjectID, name string) (io.ReadCloser, error) {
	return nil, nil
}

// Update data in the storage
func (d *Driver) Update(ctx context.Context, id npio.ObjectID, name string, data io.Reader, meta *models.ItemMeta) error {
	return nil
}

// Update data in the storage
func (d *Driver) UpdateMeta(ctx context.Context, id npio.ObjectID, name string, meta *models.ItemMeta) error {
	return nil
}

// Clean removes all internal data from object except original
func (d *Driver) Clean(ctx context.Context, id npio.ObjectID) error {
	return nil
}

// Remove object with ID
func (d *Driver) Remove(ctx context.Context, id npio.ObjectID, names ...string) error {
	return nil
}

// Scan storage by pattern
//
//	pattern: search type equals to glob https://golang.org/pkg/path/filepath/#Glob
func (d *Driver) Scan(ctx context.Context, pattern string, walkf npio.WalkStorageFnk) error {
	return nil
}

// ReadManifest information method
func (d *Driver) ReadManifest(ctx context.Context, bucket string) (*models.Manifest, error) {
	return nil, nil
}

// UpdateManifest information method
func (d *Driver) UpdateManifest(ctx context.Context, bucket string, manifest *models.Manifest) error {
	return nil
}

var _ npio.StorageAccessor = (*Driver)(nil)
