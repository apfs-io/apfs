//go:build memory || alldb
// +build memory alldb

package dbinit

import (
	_ "github.com/apfs-io/apfs/internal/storage/database/memory"
)
