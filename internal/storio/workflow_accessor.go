package storio

import (
	"context"

	"github.com/apfs-io/apfs/models"
)

// WorkflowAccessor describes bucket-level workflow manifest read/write.
type WorkflowAccessor interface {
	// ReadWorkflow returns the bucket-level workflow manifest.
	ReadWorkflow(ctx context.Context, bucket string) (*models.Workflow, error)

	// UpdateWorkflow persists the bucket-level workflow manifest.
	UpdateWorkflow(ctx context.Context, bucket string, workflow *models.Workflow) error
}
