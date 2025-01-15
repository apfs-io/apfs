package memory

import (
	"errors"
	"sync"

	"github.com/apfs-io/apfs/internal/storage"
	"github.com/apfs-io/apfs/models"
)

var (
	errNotFound = errors.New("not found")
)

// Connector go sqlite3 object
type connector struct {
	mx  sync.RWMutex
	mem map[string]*models.Object
}

// Connect database by URL
func Connect(connectURL string) (storage.DB, error) {
	return &connector{mem: map[string]*models.Object{}}, nil
}

// Get file base object
func (db *connector) Get(objID string) (*models.Object, error) {
	db.mx.RLock()
	defer db.mx.RUnlock()
	if obj, ok := db.mem[objID]; ok {
		return obj, nil
	}
	return nil, errNotFound
}

// Set file base object
func (db *connector) Set(obj *models.Object) error {
	db.mx.Lock()
	defer db.mx.Unlock()
	db.mem[obj.ID] = obj
	return nil
}

// Delete file base object
func (db *connector) Delete(id string) error {
	db.mx.Lock()
	defer db.mx.Unlock()
	delete(db.mem, id)
	return nil
}
