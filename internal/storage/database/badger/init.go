package badger

import (
	"github.com/apfs-io/apfs/internal/storage/database"
)

func init() {
	database.Register(`badger`, Connect)
}
