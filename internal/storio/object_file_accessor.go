package storio

import (
	"context"
	"io"

	"github.com/apfs-io/apfs/models"
)

// FileInfo describes a single file inside an object scope.
type FileInfo struct {
	// Path is relative to the object root: "thumbs/1.jpg", "main.mp4"
	Path string
	// Size in bytes; 0 if unknown.
	Size int64
	// Meta is the item metadata if available; may be nil.
	Meta *models.ItemMeta
}

// ObjectFileAccessor provides hierarchical file access within a single object
// scope. All paths are relative to the object directory:
//
//	"main.mp4"       — top-level file
//	"thumbs/1.jpg"   — nested file
//
// Drivers (FS, S3) translate these paths to absolute filesystem paths or
// S3 key prefixes transparently.
type ObjectFileAccessor interface {
	// WriteFile writes data to the given relative path inside the object.
	// It creates intermediate directories as needed.
	WriteFile(ctx context.Context, id ObjectID, path string, data io.Reader, meta *models.ItemMeta) error

	// ReadFile returns a reader for the given relative path.
	ReadFile(ctx context.Context, id ObjectID, path string) (io.ReadCloser, error)

	// ListFiles lists all files inside the object whose path matches pattern.
	// pattern follows filepath.Match semantics; "" or "*" matches everything.
	// Results are sorted by path.
	ListFiles(ctx context.Context, id ObjectID, pattern string) ([]*FileInfo, error)

	// DeleteFiles removes the given relative paths from the object.
	// Paths that do not exist are silently skipped.
	DeleteFiles(ctx context.Context, id ObjectID, paths ...string) error

	// MoveFile renames srcPath to dstPath within the same object scope.
	MoveFile(ctx context.Context, id ObjectID, srcPath, dstPath string) error
}
