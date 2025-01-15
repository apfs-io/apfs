//go:build mssql || alldb
// +build mssql alldb

package dbinit

import (
	_ "github.com/jinzhu/gorm/dialects/mssql"
)
