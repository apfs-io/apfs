package storage

import (
	"strings"

	storio "github.com/apfs-io/apfs/internal/storio"
)

func objcID(obj any) string {
	switch v := obj.(type) {
	case string:
		return v
	case storio.Object:
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
