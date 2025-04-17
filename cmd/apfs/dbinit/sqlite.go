//go:build sqlite || sqlite3 || alldb
// +build sqlite sqlite3 alldb

package dbinit

import (
	"github.com/apfs-io/apfs/internal/storage/database"
	"github.com/apfs-io/apfs/internal/storage/database/gorm"
)

func init() {
	database.Register(`sqlite`, gorm.Connect)
	database.Register(`sqlite3`, gorm.Connect)
}
