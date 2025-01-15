package fs

import (
	"io"
	"os"
	// "github.com/coocood/freecache"
)

// FileCacher accessor
type FileCacher interface {
	Read(filepath string) (io.ReadSeekCloser, error)
	// Delete - clear cache
	Delete(filepath string) error
}

type dummyFileCache struct{}

func (c *dummyFileCache) Read(filepath string) (io.ReadSeekCloser, error) {
	return os.Open(filepath)
}

// Delete - clear cache
func (c *dummyFileCache) Delete(filepath string) error {
	return nil
}
