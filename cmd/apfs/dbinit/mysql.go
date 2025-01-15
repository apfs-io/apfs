//go:build mysql || alldb
// +build mysql alldb

package dbinit

import (
	_ "github.com/jinzhu/gorm/dialects/mysql"
)
