package utils

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

const ffprobeTimeout = 30 * time.Second

// errFFProbeUnavailable is returned when ffprobe is not on PATH.
var errFFProbeUnavailable = errors.New("ffprobe is not available")

var (
	ffprobeOnce sync.Once
	ffprobeBin  string
)

// videoInfo is the subset of ffprobe output stored in ItemMeta.
type videoInfo struct {
	Width    int
	Height   int
	Duration int // seconds
	Bitrate  string
	Codec    string
}

// ffprobePath returns the ffprobe binary path, or "" when it is not installed.
func ffprobePath() string {
	ffprobeOnce.Do(func() {
		ffprobeBin, _ = exec.LookPath("ffprobe")
	})
	return ffprobeBin
}

// probeVideo runs ffprobe on the reader's content. A non-file reader is
// spooled to a temp file because MP4 may keep its index at the end (no pipe).
// The reader is rewound before returning.
func probeVideo(reader io.ReadSeeker) (*videoInfo, error) {
	bin := ffprobePath()
	if bin == "" {
		return nil, errFFProbeUnavailable
	}
	defer func() { _, _ = reader.Seek(0, io.SeekStart) }()

	filename := ""
	if f, ok := reader.(*os.File); ok {
		filename = f.Name()
	} else {
		tmp, err := os.CreateTemp("", "apfs-probe-*")
		if err != nil {
			return nil, err
		}
		defer func() { _ = os.Remove(tmp.Name()) }()
		if _, err = reader.Seek(0, io.SeekStart); err == nil {
			_, err = io.Copy(tmp, reader)
		}
		if cerr := tmp.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return nil, err
		}
		filename = tmp.Name()
	}

	ctx, cancel := context.WithTimeout(context.Background(), ffprobeTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,codec_name,bit_rate:stream_tags=rotate:stream_side_data=rotation:format=duration,bit_rate",
		"-of", "json",
		filename,
	).Output()
	if err != nil {
		return nil, err
	}
	return parseFFProbe(out)
}

type ffprobeOutput struct {
	Streams []struct {
		Width     int               `json:"width"`
		Height    int               `json:"height"`
		CodecName string            `json:"codec_name"`
		BitRate   string            `json:"bit_rate"`
		Tags      map[string]string `json:"tags"`
		SideData  []struct {
			Rotation *float64 `json:"rotation"`
		} `json:"side_data_list"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
	} `json:"format"`
}

// parseFFProbe extracts video metadata from ffprobe JSON output. Width and
// Height are the displayed dimensions (swapped for a 90/270 degree rotation).
func parseFFProbe(data []byte) (*videoInfo, error) {
	var out ffprobeOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if len(out.Streams) == 0 {
		return nil, errors.New("ffprobe: no video stream")
	}
	stream := out.Streams[0]
	info := &videoInfo{
		Width:   stream.Width,
		Height:  stream.Height,
		Codec:   stream.CodecName,
		Bitrate: out.Format.BitRate,
	}
	if info.Bitrate == "" {
		info.Bitrate = stream.BitRate
	}
	if d, err := strconv.ParseFloat(out.Format.Duration, 64); err == nil && d > 0 {
		info.Duration = int(math.Round(d))
	}

	rotation := 0.0
	if r, err := strconv.ParseFloat(stream.Tags["rotate"], 64); err == nil {
		rotation = r
	}
	for _, sd := range stream.SideData {
		if sd.Rotation != nil {
			rotation = *sd.Rotation
			break
		}
	}
	if quarter := int(math.Round(rotation/90)) % 2; quarter != 0 {
		info.Width, info.Height = info.Height, info.Width
	}
	return info, nil
}
