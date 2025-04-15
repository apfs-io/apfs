package storage

import (
	"strings"

	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/models"
)

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
