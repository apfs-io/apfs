package preview

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsImage(t *testing.T) {
	assert.True(t, IsImage("image/jpeg"))
	assert.True(t, IsImage("image/png"))
	assert.True(t, IsImage("image/svg+xml"))
	assert.True(t, IsImage("IMAGE/WEBP"))
	assert.True(t, IsImage("image"))
	assert.False(t, IsImage("video/mp4"))
	assert.False(t, IsImage("application/pdf"))
	assert.False(t, IsImage(""))
}

func TestKindFromContentType(t *testing.T) {
	cases := []struct {
		ct   string
		kind string
	}{
		{"video/mp4", KindVideo},
		{"video/webm", KindVideo},
		{"audio/mpeg", KindAudio},
		{"audio/ogg", KindAudio},
		{"text/html", KindHTML},
		{"application/xhtml+xml", KindHTML},
		{"htmlarch", KindHTML},
		{"application/pdf", KindPDF},
		{"application/zip", KindArchive},
		{"application/gzip", KindArchive},
		{"application/x-tar", KindArchive},
		{"application/x-7z-compressed", KindArchive},
		{"application/vnd.rar", KindArchive},
		{"application/json", KindFile},
		{"text/plain", KindFile},
		{"application/octet-stream", KindFile},
		{"", KindFile},
		{"video/mp4; codecs=avc1", KindVideo},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.kind, KindFromContentType(tc.ct), tc.ct)
	}
}

func TestIconEmbed(t *testing.T) {
	kinds := []string{KindVideo, KindAudio, KindHTML, KindPDF, KindArchive, KindFile}
	for _, kind := range kinds {
		ct, data := Icon(kind)
		assert.Equal(t, SVGContentType, ct, kind)
		require.NotEmpty(t, data, kind)
		assert.True(t, strings.Contains(string(data), "<svg"), kind)
	}
	ct, data := Icon("unknown-kind")
	assert.Equal(t, SVGContentType, ct)
	require.NotEmpty(t, data)
}
