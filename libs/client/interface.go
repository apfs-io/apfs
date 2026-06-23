package client

import (
	"context"
	"errors"
	"io"

	"github.com/apfs-io/apfs/models"
)

// Error list...
var (
	ErrInvalidParams                 = errors.New("invalid parameter")
	ErrInvalidDeleteRequestArguments = errors.New(`invalid delete request arguments`)
)

// ObjectManagerClient interface represents interaction with object storage
type ObjectManagerClient interface {
	// Refresg file object data
	Refresh(ctx context.Context, id *ObjectID, opts ...RequestOption) error

	// Get file object from storage
	Head(ctx context.Context, id *ObjectID, opts ...RequestOption) (*models.Object, error)

	// Read object with body
	Get(ctx context.Context, id *ObjectID, opts ...RequestOption) (*models.Object, io.ReadCloser, error)

	// UploadFile object into storage
	UploadFile(ctx context.Context, filepath string, opts ...RequestOption) (*models.Object, error)

	// Upload file object into storage
	Upload(ctx context.Context, data io.Reader, opts ...RequestOption) (*models.Object, error)

	// Delete object from storage
	Delete(ctx context.Context, id any, opts ...RequestOption) error
}

// MetadataManagerClient interface represents interaction with metadata storage
type MetadataManagerClient interface {
	// SetWorkflow stores the workflow manifest for the group.
	SetWorkflow(ctx context.Context, w *models.Workflow, opts ...RequestOption) error

	// GetWorkflow reads the workflow manifest for the group.
	GetWorkflow(ctx context.Context, opts ...RequestOption) (*models.Workflow, error)
}

// Client interface accessor to the Disk API
type Client interface {
	io.Closer
	ObjectManagerClient
	MetadataManagerClient
	// WithGroup returns a client scoped to the given group.
	WithGroup(name string) Client
	// Group returns a fluent Group client scoped to the given bucket.
	Group(name string) *Group
}
