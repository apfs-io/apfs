//go:build mysql || mssql || postgres || sqlite || alldb
// +build mysql mssql postgres sqlite alldb

package dbinit

import (
	_ "github.com/apfs-io/apfs/internal/storage/database/gorm"
)
