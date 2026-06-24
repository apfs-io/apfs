// Package driver provides a StateStore that delegates to an ObjectStateAccessor
// (i.e. the storage driver). Use this when you want state.json persisted in
// the same bucket/object tree as the other object files.
package driver

import (
	"context"

	"github.com/apfs-io/apfs/internal/storage/statestore"
	storio "github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/models"
)

// Store wraps an ObjectStateAccessor as a StateStore.
type Store struct {
	accessor storio.ObjectStateAccessor
}

// New creates a driver-backed StateStore.
func New(accessor storio.ObjectStateAccessor) *Store {
	return &Store{accessor: accessor}
}

// Get implements StateStore.
func (s *Store) Get(ctx context.Context, id string) (*models.ProcessingState, error) {
	return s.accessor.ReadState(ctx, storio.ObjectIDType(id))
}

// Set implements StateStore.
func (s *Store) Set(ctx context.Context, id string, state *models.ProcessingState) error {
	return s.accessor.WriteState(ctx, storio.ObjectIDType(id), state)
}

// Delete implements StateStore.
// Not all drivers support deletion of state.json; this is a best-effort no-op.
func (s *Store) Delete(ctx context.Context, id string) error {
	return s.accessor.WriteState(ctx, storio.ObjectIDType(id), nil)
}

var _ statestore.StateStore = (*Store)(nil)
