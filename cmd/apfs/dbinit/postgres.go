//go:build postgres || alldb
// +build postgres alldb

package dbinit

import (
	_ "github.com/jinzhu/gorm/dialects/postgres"
)
