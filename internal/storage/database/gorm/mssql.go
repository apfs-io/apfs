//go:build mssql || alldb
// +build mssql alldb

package gorm

import (
	// _ "gorm.io/gorm/dialects/mssql"
	"gorm.io/driver/sqlserver"
)

func init() {
	dialectors["mssql"] = sqlserver.Open
	dialectors["sqlserver"] = sqlserver.Open
}
