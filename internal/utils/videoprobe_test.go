package utils

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFFProbe(t *testing.T) {
	tests := []struct {
		name string
		json string
		want videoInfo
	}{
		{
			name: "landscape h264",
			json: `{"streams":[{"width":1920,"height":1080,"codec_name":"h264","bit_rate":"4000000"}],
				"format":{"duration":"12.6","bit_rate":"4128000"}}`,
			want: videoInfo{Width: 1920, Height: 1080, Codec: "h264", Duration: 13, Bitrate: "4128000"},
		},
		{
			name: "rotate tag",
			json: `{"streams":[{"width":1920,"height":1080,"codec_name":"h264","tags":{"rotate":"90"}}],
				"format":{"duration":"3.0","bit_rate":"1000"}}`,
			want: videoInfo{Width: 1080, Height: 1920, Codec: "h264", Duration: 3, Bitrate: "1000"},
		},
		{
			name: "rotation side data",
			json: `{"streams":[{"width":1280,"height":720,"codec_name":"hevc",
				"side_data_list":[{"rotation":-90}]}],"format":{"duration":"1.2"}}`,
			want: videoInfo{Width: 720, Height: 1280, Codec: "hevc", Duration: 1},
		},
		{
			name: "180 keeps dimensions, stream bitrate fallback, no duration",
			json: `{"streams":[{"width":640,"height":360,"codec_name":"vp9","bit_rate":"900",
				"side_data_list":[{"rotation":180}]}],"format":{}}`,
			want: videoInfo{Width: 640, Height: 360, Codec: "vp9", Bitrate: "900"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info, err := parseFFProbe([]byte(tc.json))
			require.NoError(t, err)
			assert.Equal(t, tc.want, *info)
		})
	}

	_, err := parseFFProbe([]byte(`{}`))
	assert.Error(t, err)
	_, err = parseFFProbe([]byte(`not json`))
	assert.Error(t, err)
}

func TestCollectFileInfo_Video(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil || ffprobePath() == "" {
		t.Skip("ffmpeg/ffprobe not installed")
	}
	path := filepath.Join(t.TempDir(), "clip.mp4")
	cmd := exec.Command(ffmpeg, "-v", "error", "-f", "lavfi", "-i", "testsrc=duration=1:size=320x240:rate=10",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-y", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg cannot encode test clip: %v %s", err, out)
	}

	meta, err := CollectFileInfo(nil, path, "video/mp4")
	require.NoError(t, err)
	assert.Equal(t, 320, meta.Width)
	assert.Equal(t, 240, meta.Height)
	assert.Equal(t, "h264", meta.Codec)
	assert.Equal(t, 1, meta.Duration)
	assert.NotEmpty(t, meta.Bitrate)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	meta, err = CollectReadSeekerInfo(nil, bytes.NewReader(data), "clip.mp4", "video/mp4")
	require.NoError(t, err)
	assert.Equal(t, 320, meta.Width)
	assert.Equal(t, 240, meta.Height)
	assert.Equal(t, "h264", meta.Codec)
}
