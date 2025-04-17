package badger

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/demdxx/gocast/v2"
	badger "github.com/dgraph-io/badger/v4"

	"github.com/apfs-io/apfs/internal/storage"
	"github.com/apfs-io/apfs/models"
)

// connector represents a BadgerDB connection wrapper implementing the storage.DB interface.
type connector struct {
	conn *badger.DB // Badger database connection
}

// Connect initializes a BadgerDB connection using the provided URL.
func Connect(_ context.Context, connectURL string) (storage.DB, error) {
	// Parse the connection URL
	u, err := url.Parse(connectURL)
	if err != nil {
		return nil, err
	}

	// Configure BadgerDB options
	opts := badger.
		DefaultOptions(strings.TrimPrefix(connectURL, `badger://`)).
		WithValueLogFileSize(1 * 1024 * 1024) // Set value log file size to 1MB

	// Apply optional configurations based on URL query parameters
	if gocast.Bool(u.Query().Get(`sync`)) {
		opts = opts.WithSyncWrites(true) // Enable synchronous writes
	}
	if gocast.Bool(u.Query().Get(`readonly`)) {
		opts = opts.WithReadOnly(true) // Open database in read-only mode
	}
	if gocast.Bool(u.Query().Get(`inmemory`)) {
		opts = opts.WithInMemory(true) // Use in-memory database
	}

	// Open the BadgerDB connection
	conn, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &connector{conn: conn}, nil
}

// Get retrieves an object from the database by its ID.
func (db *connector) Get(objID string) (obj *models.Object, err error) {
	err = db.conn.View(func(txn *badger.Txn) error {
		// Fetch the item from the database
		item, err := txn.Get([]byte(objID))
		if err != nil {
			return err
		}
		// Decode the item's value into the object
		return item.Value(func(data []byte) error {
			return json.Unmarshal(data, &obj)
		})
	})
	return obj, err
}

// Set stores an object in the database.
func (db *connector) Set(obj *models.Object) error {
	return db.conn.Update(func(txn *badger.Txn) error {
		// Serialize the object to JSON
		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		// Store the serialized object in the database
		return txn.Set([]byte(obj.ID), data)
	})
}

// Delete removes an object from the database by its path.
func (db *connector) Delete(objID string) error {
	return db.conn.Update(func(txn *badger.Txn) error {
		// Delete the object from the database
		return txn.Delete([]byte(objID))
	})
}

// Close closes the BadgerDB connection.
func (db *connector) Close() error {
	return db.conn.Close()
}
