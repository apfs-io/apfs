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
	// Refresh triggers reprocessing of the named object.
	Refresh(ctx context.Context, id *ObjectID, opts ...RequestOption) error

	// Head returns the object descriptor. Pass WithWorkflow(), WithState(), or
	// WithFullState() to include additional data in the single response.
	Head(ctx context.Context, id *ObjectID, opts ...RequestOption) (*Object, error)

	// Get returns the object descriptor and a content stream.
	Get(ctx context.Context, id *ObjectID, opts ...RequestOption) (*Object, io.ReadCloser, error)

	// UploadFile uploads a file from disk to storage.
	UploadFile(ctx context.Context, filepath string, opts ...RequestOption) (*Object, error)

	// Upload streams data to storage and returns the resulting object.
	Upload(ctx context.Context, data io.Reader, opts ...RequestOption) (*Object, error)

	// Delete removes an object (or named sub-items) from storage.
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
