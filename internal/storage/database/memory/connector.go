package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/apfs-io/apfs/internal/storage"
	"github.com/apfs-io/apfs/models"
	"golang.org/x/exp/maps"
)

var (
	// errNotFound is returned when an object is not found in the in-memory database.
	errNotFound = errors.New("not found")
)

// connector represents an in-memory database implementation of the storage.DB interface.
type connector struct {
	mx  sync.RWMutex              // Mutex to ensure thread-safe access to the in-memory map.
	mem map[string]*models.Object // In-memory storage for objects, keyed by their ID.
}

// Connect initializes a new in-memory database instance.
// The connectURL parameter is ignored as this is an in-memory implementation.
func Connect(_ context.Context, connectURL string) (storage.DB, error) {
	return &connector{mem: map[string]*models.Object{}}, nil
}

// Get retrieves an object from the in-memory database by its ID.
// Returns the object if found, or an error if the object does not exist.
func (db *connector) Get(objID string) (*models.Object, error) {
	db.mx.RLock() // Acquire a read lock for thread-safe access.
	defer db.mx.RUnlock()
	if obj, ok := db.mem[objID]; ok {
		return obj, nil
	}
	return nil, errNotFound
}

// Set stores or updates an object in the in-memory database.
// The object is indexed by its ID.
func (db *connector) Set(obj *models.Object) error {
	db.mx.Lock() // Acquire a write lock for thread-safe modification.
	defer db.mx.Unlock()
	db.mem[obj.ID] = obj
	return nil
}

// Delete removes an object from the in-memory database by its ID.
// If the object does not exist, the operation is a no-op.
func (db *connector) Delete(id string) error {
	db.mx.Lock() // Acquire a write lock for thread-safe modification.
	defer db.mx.Unlock()
	delete(db.mem, id)
	return nil
}

// Close clears all objects from the in-memory database.
// This method is typically called to release resources.
func (db *connector) Close() error {
	db.mx.Lock() // Acquire a write lock for thread-safe modification.
	defer db.mx.Unlock()
	maps.Clear(db.mem) // Clear the in-memory map.
	return nil
}
