// Package statestore provides a persistence layer for ProcessingState.
// It sits between the workflow executor and the underlying storage driver.
package statestore

import (
	"context"

	"github.com/apfs-io/apfs/models"
)

// StateStore reads and writes ProcessingState by object ID.
type StateStore interface {
	// Get returns the current state, or (nil, nil) if no state exists.
	Get(ctx context.Context, id string) (*models.ProcessingState, error)

	// Set persists a ProcessingState.
	Set(ctx context.Context, id string, state *models.ProcessingState) error

	// Delete removes the state for the given object.
	Delete(ctx context.Context, id string) error
}
