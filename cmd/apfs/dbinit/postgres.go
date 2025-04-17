//go:build postgres || alldb
// +build postgres alldb

package dbinit

import (
	"github.com/apfs-io/apfs/internal/storage/database"
	"github.com/apfs-io/apfs/internal/storage/database/gorm"
)

func init() {
	database.Register(`postgres`, gorm.Connect)
}
