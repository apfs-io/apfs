package client

import (
	"context"
	"errors"
	"io"

	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
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
	Refresh(ctx context.Context, id *protocol.ObjectID, opts ...RequestOption) error

	// Get file object from storage
	Head(ctx context.Context, id *protocol.ObjectID, opts ...RequestOption) (*models.Object, error)

	// Read object with body
	Get(ctx context.Context, id *protocol.ObjectID, opts ...RequestOption) (*models.Object, io.ReadCloser, error)

	// UploadFile object into storage
	UploadFile(ctx context.Context, filepath string, opts ...RequestOption) (*models.Object, error)

	// Upload file object into storage
	Upload(ctx context.Context, data io.Reader, opts ...RequestOption) (*models.Object, error)

	// Delete object from storage
	Delete(ctx context.Context, id any, opts ...RequestOption) error
}

// MetadataManagerClient interface represents interaction with metadata storage
type MetadataManagerClient interface {
	// SetManifest of the group
	SetManifest(ctx context.Context, manifest *models.Manifest, opts ...RequestOption) error

	// GetManifest of the group
	GetManifest(ctx context.Context, opts ...RequestOption) (*models.Manifest, error)
}

// Client interface accessor to the Disk API
type Client interface {
	io.Closer
	ObjectManagerClient
	MetadataManagerClient
	WithGroup(name string) Client
}
