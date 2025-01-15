package io

import (
	"context"
	"time"

	"github.com/apfs-io/apfs/models"
)

// ScanObjInfo provides access to the object info or group
type ScanObjInfo interface {
	ObjectID
	Name() string
	IsGroup() bool
	Size() uint64
	Meta() *models.Meta
	Manifest() *models.Manifest
	CreatedAt() time.Time
	UpdatedAt() time.Time
}

// WalkStorageFnk defines the function which will be call by storage scanning process
type WalkStorageFnk func(path string, err error) error

// ObjectScanner of the structure accessor
type ObjectScanner interface {
	// Scan storage by pattern
	// 	pattern: search type equals to glob https://golang.org/pkg/path/filepath/#Glob
	Scan(ctx context.Context, pattern string, walkf WalkStorageFnk) error
}
