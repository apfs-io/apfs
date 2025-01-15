package utils

import (
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/apfs-io/apfs/models"

	"github.com/stretchr/testify/assert"
)

func Test_trimExt(t *testing.T) {
	var tests = []struct {
		name   string
		target string
	}{
		{
			name:   "file.ext",
			target: "file",
		},
		{
			name:   "file",
			target: "file",
		},
		{
			name:   "file.jpeg",
			target: "file",
		},
		{
			name:   "/file.jpeg",
			target: "/file",
		},
		{
			name:   "/base/path/file.jpeg",
			target: "/base/path/file",
		},
		{
			name:   "19bff28990b058b56f70202278c8cf6b.jpg",
			target: "19bff28990b058b56f70202278c8cf6b",
		},
		{
			name:   "19bff28990b058b56f70202278c8cf6b.jpg.bak",
			target: "19bff28990b058b56f70202278c8cf6b.jpg",
		},
	}

	for _, test := range tests {
		if res := trimExt(test.name); res != test.target {
			t.Errorf("'%s' != '%s'", res, test.target)
		}
	}
}

func Test_CollectFileInfo(t *testing.T) {
	_, fileName, _, _ := runtime.Caller(0)
	filePath := filepath.Join(fileName, "../../../testdata/cat.jpg")
	meta, err := CollectFileInfo(nil, filePath, "")
	if assert.NoError(t, err) {
		assert.Equal(t, "jpg", meta.NameExt)
		assert.Equal(t, int64(27878), meta.Size)
		assert.Equal(t, models.ObjectType("image"), meta.Type)
		assert.Equal(t, "image/jpeg", meta.ContentType)
		assert.Equal(t, 320, meta.Width)
		assert.Equal(t, 419, meta.Height)
	}
}
