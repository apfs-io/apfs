//go:build (sqlite || alldb) && !nosqlite
// +build sqlite alldb
// +build !nosqlite

package dbinit

import (
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)
