package procregistry

import (
	"context"
	"sync"
	"time"
)

type memoryRegistry struct {
	mu    sync.Mutex
	items map[string]time.Time
	busy  map[string]struct{}
}

// NewMemory returns an in-process inflight registry.
func NewMemory() Registry {
	return &memoryRegistry{
		items: make(map[string]time.Time),
		busy:  make(map[string]struct{}),
	}
}

func (r *memoryRegistry) Add(_ context.Context, objectID string) error {
	if objectID == "" {
		return nil
	}
	r.mu.Lock()
	r.items[objectID] = time.Now()
	r.mu.Unlock()
	return nil
}

func (r *memoryRegistry) Remove(_ context.Context, objectID string) error {
	r.mu.Lock()
	delete(r.items, objectID)
	delete(r.busy, objectID)
	r.mu.Unlock()
	return nil
}

func (r *memoryRegistry) ListOlderThan(_ context.Context, age time.Duration) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cutoff := time.Now().Add(-age)
	out := make([]string, 0, len(r.items))
	for id, ts := range r.items {
		if ts.Before(cutoff) || ts.Equal(cutoff) {
			out = append(out, id)
		}
	}
	return out, nil
}

func (r *memoryRegistry) TryBegin(objectID string) bool {
	if objectID == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.busy[objectID]; ok {
		return false
	}
	r.busy[objectID] = struct{}{}
	return true
}

func (r *memoryRegistry) End(objectID string) {
	r.mu.Lock()
	delete(r.busy, objectID)
	r.mu.Unlock()
}

func (r *memoryRegistry) IsBusy(objectID string) bool {
	r.mu.Lock()
	_, ok := r.busy[objectID]
	r.mu.Unlock()
	return ok
}
