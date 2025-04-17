//go:build mssql || alldb
// +build mssql alldb

package dbinit

import (
	"github.com/apfs-io/apfs/internal/storage/database"
	"github.com/apfs-io/apfs/internal/storage/database/gorm"
)

func init() {
	database.Register(`mssql`, gorm.Connect)
}
