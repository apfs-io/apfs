package database

import (
	"net/url"

	"github.com/pkg/errors"

	"github.com/apfs-io/apfs/internal/storage"
)

var errUnsupportedDatabaseDriver = errors.New(`unsupported database driver`)

var registry = map[string]Connector{}

// Connector database type
type Connector func(connect string) (storage.DB, error)

// Register database connector
func Register(driver string, connector Connector) {
	registry[driver] = connector
}

// Open new database connection
func Open(connect string) (storage.DB, error) {
	u, err := url.Parse(connect)
	if err != nil {
		return nil, err
	}
	connector := registry[u.Scheme]
	if connector == nil {
		return nil, errors.Wrap(errUnsupportedDatabaseDriver, u.Scheme)
	}
	return connector(connect)
}
