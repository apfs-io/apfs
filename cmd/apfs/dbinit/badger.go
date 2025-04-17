//go:build badger || alldb
// +build badger alldb

package dbinit

import (
	"github.com/apfs-io/apfs/internal/storage/database"
	"github.com/apfs-io/apfs/internal/storage/database/badger"
)

func init() {
	database.Register(`badger`, badger.Connect)
}
