package badger

import (
	"encoding/json"
	"strings"

	badger "github.com/dgraph-io/badger/v3"

	"github.com/apfs-io/apfs/internal/storage"
	"github.com/apfs-io/apfs/models"
)

// Connector go sqlite3 object
type connector struct {
	conn *badger.DB
}

// Connect database by URL
func Connect(connectURL string) (storage.DB, error) {
	conn, err := badger.Open(badger.DefaultOptions(
		strings.TrimPrefix(connectURL, `badger://`),
	).WithValueLogFileSize(1 * 1024 * 1024))
	if err != nil {
		return nil, err
	}
	return &connector{conn: conn}, nil
}

// Get file base object
func (db *connector) Get(objID string) (obj *models.Object, err error) {
	err = db.conn.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(objID))
		if err != nil {
			return err
		}
		return item.Value(func(data []byte) error {
			return json.Unmarshal(data, &obj)
		})
	})
	return obj, err
}

// Set file base object
func (db *connector) Set(obj *models.Object) error {
	return db.conn.Update(func(txn *badger.Txn) error {
		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		return txn.Set([]byte(obj.ID), data)
	})
}

// Delete file base object
func (db *connector) Delete(path string) error {
	return db.conn.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(path))
	})
}
