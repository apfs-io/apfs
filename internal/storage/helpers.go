package storage

import (
	"bytes"
	"io"
	"reflect"
	"strings"

	"github.com/pkg/errors"

	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/models"
)

var errReaderResetPosition = errors.New(`reader can't reset position`)

func objcID(obj any) string {
	switch v := obj.(type) {
	case string:
		return v
	case npio.Object:
		return v.ID().String()
	}
	return ""
}

func splitPath(path string) (group, newpath string) {
	path = strings.TrimPrefix(path, "/")
	data := strings.SplitN(path, "/", 2)
	if len(data) == 1 {
		return "", path
	}
	return data[0], data[1]
}

func processingStatusBy(cObject npio.Object, manifest *models.Manifest, err error) models.ObjectStatus {
	if err != nil {
		return models.StatusError
	}
	updateProcessingState(cObject, manifest)
	return cObject.Status()
}

func resetReader(reader io.Reader) (out io.Reader, err error) {
	switch r := reader.(type) {
	case io.ReadSeeker:
		_, err = r.Seek(0, io.SeekStart)
		out = r
	case *bytes.Buffer:
		out = bytes.NewReader(r.Bytes())
	default:
		err = errReaderResetPosition
	}
	return out, err
}

func updateProcessingState(cObject npio.Object, manifest *models.Manifest) {
	meta := cObject.MustMeta()
	if meta.IsProcessingComplete(manifest) {
		if meta.IsComplete(manifest) {
			cObject.StatusUpdate(models.StatusOK)
		} else if meta.ErrorTaskCount() > 0 {
			cObject.StatusUpdate(models.StatusError)
		} else {
			cObject.StatusUpdate(models.StatusUndefined)
		}
	} else {
		cObject.StatusUpdate(models.StatusProcessing)
	}
}

func isNil(v any) bool {
	return v == nil || reflect.ValueOf(v).IsNil()
}

func defStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
