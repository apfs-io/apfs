package utils

import (
	"bytes"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/apfs-io/apfs/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		assert.Equal(t, "cat", meta.Name)
		assert.Equal(t, "jpg", meta.NameExt)
		assert.Equal(t, "cat.jpg", meta.Path)
		assert.Equal(t, int64(27878), meta.Size)
		assert.Equal(t, models.ObjectType("image"), meta.Type)
		assert.Equal(t, "image/jpeg", meta.ContentType)
		assert.Equal(t, 320, meta.Width)
		assert.Equal(t, 419, meta.Height)
	}
}

func TestCollectReadSeekerInfo_NonImageSetsNameAndPath(t *testing.T) {
	cases := []struct {
		path        string
		contentType string
		wantName    string
		wantExt     string
		wantPath    string
		wantType    models.ObjectType
	}{
		{"prim.mp4", "video/mp4", "prim", "mp4", "prim.mp4", models.TypeVideo},
		{"prim.mp3", "audio/mpeg", "prim", "mp3", "prim.mp3", models.TypeAudio},
		{"prim.html", "text/html", "prim", "html", "prim.html", models.TypeOther},
	}
	for _, tc := range cases {
		t.Run(tc.contentType, func(t *testing.T) {
			meta, err := CollectReadSeekerInfo(nil, bytes.NewReader([]byte("dummy-media")), tc.path, tc.contentType)
			require.NoError(t, err)
			assert.Equal(t, tc.wantName, meta.Name)
			assert.Equal(t, tc.wantExt, meta.NameExt)
			assert.Equal(t, tc.wantPath, meta.Path)
			assert.Equal(t, tc.wantType, meta.Type)
			assert.Equal(t, tc.contentType, meta.ContentType)
			assert.Equal(t, int64(len("dummy-media")), meta.Size)
		})
	}
}

func TestCollectReadSeekerInfo_OverwritesStaleFlatPath(t *testing.T) {
	_, fileName, _, _ := runtime.Caller(0)
	f, err := os.Open(filepath.Join(fileName, "../../../testdata/cat.jpg"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	meta, err := CollectReadSeekerInfo(&models.ItemMeta{Path: "prim.png"}, f, "w320.jpg", "image/jpeg")
	require.NoError(t, err)
	assert.Equal(t, "w320", meta.Name)
	assert.Equal(t, "jpg", meta.NameExt)
	assert.Equal(t, "w320.jpg", meta.Path)
}

func TestCollectReadSeekerInfo_KeepsNestedPath(t *testing.T) {
	meta, err := CollectReadSeekerInfo(
		&models.ItemMeta{Path: "thumbs/1.jpg"},
		bytes.NewReader([]byte("dummy-media")),
		"thumbs/1.jpg",
		"video/mp4",
	)
	require.NoError(t, err)
	assert.Equal(t, "1", meta.Name)
	assert.Equal(t, "thumbs/1.jpg", meta.Path)
}
