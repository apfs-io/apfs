package storio

import (
	"context"

	"github.com/apfs-io/apfs/models"
)

// ObjectStateAccessor persists processing state (state.json) alongside the
// object's data files. It is the low-level storage layer; the high-level
// StateStore (internal/storage/statestore) provides caching on top.
type ObjectStateAccessor interface {
	// ReadState loads the current ProcessingState for the object.
	// Returns nil, nil when no state file exists yet.
	ReadState(ctx context.Context, id ObjectID) (*models.ProcessingState, error)

	// WriteState persists a ProcessingState for the object.
	WriteState(ctx context.Context, id ObjectID, state *models.ProcessingState) error
}
