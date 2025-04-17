//go:build memory || alldb
// +build memory alldb

package dbinit

import (
	"github.com/apfs-io/apfs/internal/storage/database"
	"github.com/apfs-io/apfs/internal/storage/database/memory"
)

func init() {
	// Register the memory database with the database package
	database.Register(`memory`, memory.Connect)
}
