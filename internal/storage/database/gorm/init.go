package gorm

import "github.com/apfs-io/apfs/internal/storage/database"

func init() {
	database.Register(`sqlite`, Connect)
	database.Register(`sqlite3`, Connect)
	database.Register(`postgres`, Connect)
	database.Register(`mysql`, Connect)
	database.Register(`mssql`, Connect)
}
