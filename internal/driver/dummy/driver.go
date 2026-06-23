// Package dummy provides a no-op StorageAccessor implementation useful for
// testing and as a placeholder for new driver development.
package dummy

import (
	"context"
	"io"
	"net/url"

	storio "github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/models"
)

// Storage is a no-op StorageAccessor that silently discards all operations.
type Storage struct{}

// Create implements ObjectAccessor.
func (d *Storage) Create(ctx context.Context, bucket string, id storio.ObjectID, overwrite bool, params url.Values) (storio.Object, error) {
	return nil, nil
}

// UpdateParams implements ObjectAccessor.
func (d *Storage) UpdateParams(ctx context.Context, id storio.ObjectID, params url.Values) error {
	return nil
}

// Open implements ObjectAccessor.
func (d *Storage) Open(ctx context.Context, id storio.ObjectID) (storio.Object, error) {
	return nil, nil
}

// Read implements ObjectAccessor.
func (d *Storage) Read(ctx context.Context, id storio.ObjectID, name string) (io.ReadCloser, error) {
	return nil, nil
}

// Update implements ObjectAccessor.
func (d *Storage) Update(ctx context.Context, id storio.ObjectID, name string, data io.Reader, meta *models.ItemMeta) error {
	return nil
}

// UpdateMeta implements ObjectAccessor.
func (d *Storage) UpdateMeta(ctx context.Context, id storio.ObjectID, name string, meta *models.ItemMeta) error {
	return nil
}

// Clean implements ObjectAccessor.
func (d *Storage) Clean(ctx context.Context, id storio.ObjectID) error {
	return nil
}

// Remove implements ObjectAccessor.
func (d *Storage) Remove(ctx context.Context, id storio.ObjectID, names ...string) error {
	return nil
}

// WriteFile implements ObjectFileAccessor.
func (d *Storage) WriteFile(ctx context.Context, id storio.ObjectID, path string, data io.Reader, meta *models.ItemMeta) error {
	return nil
}

// ReadFile implements ObjectFileAccessor.
func (d *Storage) ReadFile(ctx context.Context, id storio.ObjectID, path string) (io.ReadCloser, error) {
	return nil, nil
}

// ListFiles implements ObjectFileAccessor.
func (d *Storage) ListFiles(ctx context.Context, id storio.ObjectID, pattern string) ([]*storio.FileInfo, error) {
	return nil, nil
}

// DeleteFiles implements ObjectFileAccessor.
func (d *Storage) DeleteFiles(ctx context.Context, id storio.ObjectID, paths ...string) error {
	return nil
}

// MoveFile implements ObjectFileAccessor.
func (d *Storage) MoveFile(ctx context.Context, id storio.ObjectID, srcPath, dstPath string) error {
	return nil
}

// Scan implements ObjectScanner.
func (d *Storage) Scan(ctx context.Context, pattern string, walkf storio.WalkStorageFunc) error {
	return nil
}

// ReadWorkflow implements WorkflowAccessor.
func (d *Storage) ReadWorkflow(ctx context.Context, bucket string) (*models.Workflow, error) {
	return nil, nil
}

// UpdateWorkflow implements WorkflowAccessor.
func (d *Storage) UpdateWorkflow(ctx context.Context, bucket string, workflow *models.Workflow) error {
	return nil
}

// ReadState implements ObjectStateAccessor.
func (d *Storage) ReadState(ctx context.Context, id storio.ObjectID) (*models.ProcessingState, error) {
	return nil, nil
}

// WriteState implements ObjectStateAccessor.
func (d *Storage) WriteState(ctx context.Context, id storio.ObjectID, state *models.ProcessingState) error {
	return nil
}

var _ storio.StorageAccessor = (*Storage)(nil)
