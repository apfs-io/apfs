package processor

import (
	"context"

	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/models"
)

// Storage interface defines methods for interacting with storage objects
type Storage interface {
	// Object returns the object from the storage
	Object(ctx context.Context, obj any) (npio.Object, error)

	// Retrieve the object's manifest
	ObjectManifest(ctx context.Context, obj npio.Object) *models.Manifest

	// Update object metadata
	UpdateObjectInfo(ctx context.Context, obj npio.Object) error
}
