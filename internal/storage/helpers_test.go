package storage

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitPath(t *testing.T) {
	var (
		backet             = "images"
		path               = "04/12/c5f11b30d145a79a9b072a42625266e3"
		fullpath           = filepath.Join(backet, path)
		newBacket, newPath = splitPath(fullpath)
	)
	assert.Equal(t, backet, newBacket)
	assert.Equal(t, path, newPath)
}
