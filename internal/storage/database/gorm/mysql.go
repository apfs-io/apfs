//go:build mysql || alldb
// +build mysql alldb

package gorm

import (
	// _ "gorm.io/gorm/dialects/mysql"
	"gorm.io/driver/mysql"
)

func init() {
	dialectors["mysql"] = mysql.Open
}
