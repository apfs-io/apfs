//go:build sqlite || alldb
// +build sqlite alldb

package gorm

import (
	// _ "gorm.io/gorm/dialects/sqlite"
	// _ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
)

func init() {
	dialectors["sqlite"] = sqlite.Open
}
