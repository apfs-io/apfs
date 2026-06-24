package storage

import (
	"context"
	"io"

	storio "github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/models"
)

// WorkflowStorage adapts *Storage to workflow.ExecutorStorage.
type WorkflowStorage struct {
	*Storage
}

// NewWorkflowStorage wraps store for use by the workflow executor.
func NewWorkflowStorage(store *Storage) *WorkflowStorage {
	return &WorkflowStorage{Storage: store}
}

func (s *WorkflowStorage) ReadState(ctx context.Context, id storio.ObjectID) (*models.ProcessingState, error) {
	return s.GetProcessingState(ctx, id.ID().String())
}

func (s *WorkflowStorage) WriteState(ctx context.Context, id storio.ObjectID, state *models.ProcessingState) error {
	return s.SetProcessingState(ctx, id.ID().String(), state)
}

func (s *WorkflowStorage) WriteFile(
	ctx context.Context,
	id storio.ObjectID,
	path string,
	data interface{ Read([]byte) (int, error) },
	meta *models.ItemMeta,
) error {
	return s.driver.Update(ctx, id, path, data, meta)
}

func (s *WorkflowStorage) ReadFile(ctx context.Context, id storio.ObjectID, name string) (io.ReadCloser, error) {
	return s.driver.Read(ctx, id, name)
}
