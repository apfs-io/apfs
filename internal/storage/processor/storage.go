package processor

import (
	"context"

	storio "github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/models"
)

// Storage interface defines methods for interacting with storage objects
type Storage interface {
	// Object returns the object from the storage
	Object(ctx context.Context, obj any) (storio.Object, error)

	// ObjectWorkflow returns the effective workflow for the object,
	// falling back to the bucket-level workflow if the object has none.
	ObjectWorkflow(ctx context.Context, obj storio.Object) *models.Workflow

	// Update object metadata
	UpdateObjectInfo(ctx context.Context, obj storio.Object) error
}
