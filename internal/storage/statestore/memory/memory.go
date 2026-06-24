// Package memory provides an in-process StateStore backed by sync.Map.
package memory

import (
	"context"
	"sync"

	"github.com/apfs-io/apfs/internal/storage/statestore"
	"github.com/apfs-io/apfs/models"
)

// Store is a goroutine-safe in-memory StateStore.
type Store struct {
	mu   sync.RWMutex
	data map[string]*models.ProcessingState
}

// New creates an in-memory StateStore.
func New() *Store {
	return &Store{data: make(map[string]*models.ProcessingState)}
}

// Get implements StateStore.
func (s *Store) Get(_ context.Context, id string) (*models.ProcessingState, error) {
	s.mu.RLock()
	state := s.data[id]
	s.mu.RUnlock()
	return state, nil
}

// Set implements StateStore.
func (s *Store) Set(_ context.Context, id string, state *models.ProcessingState) error {
	s.mu.Lock()
	s.data[id] = state
	s.mu.Unlock()
	return nil
}

// Delete implements StateStore.
func (s *Store) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	delete(s.data, id)
	s.mu.Unlock()
	return nil
}

var _ statestore.StateStore = (*Store)(nil)
