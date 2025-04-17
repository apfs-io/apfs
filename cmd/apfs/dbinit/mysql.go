//go:build mysql || alldb
// +build mysql alldb

package dbinit

import (
	"github.com/apfs-io/apfs/internal/storage/database"
	"github.com/apfs-io/apfs/internal/storage/database/gorm"
)

func init() {
	database.Register(`mysql`, gorm.Connect)
}
