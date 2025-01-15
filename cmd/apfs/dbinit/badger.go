//go:build badger || alldb
// +build badger alldb

package dbinit

import (
	_ "github.com/apfs-io/apfs/internal/storage/database/badger"
)
