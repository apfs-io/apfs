package client

import (
	"context"
	"errors"
	"io"

	"google.golang.org/grpc"

	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	"github.com/apfs-io/apfs/models"
)

// Error list...
var (
	ErrInvalidParams                 = errors.New("invalid parameter")
	ErrInvalidDeleteRequestArguments = errors.New(`invalid delete request arguments`)
)

// Client interface accessor to the Disk API
type Client interface {
	io.Closer

	// Refresg file object data
	Refresh(ctx context.Context, id *protocol.ObjectID, opts ...grpc.CallOption) error

	// Get file object from storage
	Head(ctx context.Context, id *protocol.ObjectID, opts ...grpc.CallOption) (*models.Object, error)

	// Read object with body
	Get(ctx context.Context, id *protocol.ObjectID, opts ...grpc.CallOption) (*models.Object, io.ReadCloser, error)

	// SetManifest of the group
	SetManifest(ctx context.Context, group string, manifest *models.Manifest, opts ...grpc.CallOption) error

	// GetManifest of the group
	GetManifest(ctx context.Context, group string, opts ...grpc.CallOption) (*models.Manifest, error)

	// UploadFile object into storage
	UploadFile(ctx context.Context, group, id, filepath string, tags []string, overwrite bool, opts ...grpc.CallOption) (*models.Object, error)

	// Upload file object into storage
	Upload(ctx context.Context, group, id string, data io.Reader, tags []string, overwrite bool, opts ...grpc.CallOption) (*models.Object, error)

	// Delete object from storage
	Delete(ctx context.Context, id any, opts ...grpc.CallOption) error
}
