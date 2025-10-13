package fs

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	npio "github.com/apfs-io/apfs/internal/io"
	"github.com/apfs-io/apfs/internal/object"
)

var (
	systemDirs = []string{
		"/",
		"/root",
		"/home",
		"/var",
		"/usr",
		"/etc",
		"/opt",
		"/private",
		"/users",
		"/System",
	}
)

func loadJSONFile(ch FileCacher, filepath string, target any) (err error) {
	var file io.ReadCloser
	if file, err = ch.Read(filepath); err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(target)
}

func saveJSONFile(ch FileCacher, dst string, object any) (err error) {
	// Create directory path
	if dir := filepath.Dir(dst); len(dir) > 0 {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	if ch != nil {
		_ = ch.Delete(dst)
	}
	var file *os.File
	if file, err = os.Create(dst); err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(object)
}

func prepareFileExt(name, ext string) (_name, _ext string) {
	nameExt := ""
	_ext = prepareExt(ext)
	_name = name
	if strings.ContainsAny(name, ".") {
		nameExt = filepath.Ext(name)
		if nameExt != "" && strings.HasSuffix(name, nameExt) {
			_name = name[:len(name)-len(nameExt)]
		}
	}
	if _ext == "" {
		_ext = prepareExt(nameExt)
	}
	return _name, _ext
}

func prepareExt(ext string) string {
	if ext == "" || ext == "." {
		return ""
	}
	return "." + strings.ToLower(strings.TrimPrefix(ext, "."))
}

func updateObjectFileInfo(obj *object.Object, info os.FileInfo) {
	meta := obj.MustMeta()
	modeTime := info.ModTime()

	if meta.UpdatedAt.IsZero() {
		meta.UpdatedAt = modeTime
	}

	obj.SetUpdatedAt(meta.UpdatedAt)
	if !meta.CreatedAt.IsZero() {
		obj.SetCreatedAt(meta.CreatedAt)
	}
}

func isValidID(id npio.ObjectID) bool {
	if id == nil || id.ID() == "" {
		return false
	}
	fspath := string(id.ID())
	if fspath == "." || fspath == ".." ||
		strings.HasPrefix(fspath, "/") ||
		strings.HasPrefix(fspath, "./") || strings.HasPrefix(fspath, "../") ||
		strings.HasSuffix(fspath, "/.") || strings.HasSuffix(fspath, "/..") ||
		strings.Contains(fspath, "/./") || strings.Contains(fspath, "/../") {
		return false // This is injection
	}
	return true
}

func objectFromID(id npio.ObjectID) *object.Object {
	filepath := strings.Trim(string(id.ID()), "/")
	splits := strings.SplitN(filepath, "/", 2)
	if len(splits) != 2 {
		return object.NewObject("", "", "")
	}
	return object.NewObject(id.ID(), splits[0], splits[1])
}

func isEmptyDir(dir string) bool {
	list, err := os.ReadDir(dir)
	return err == nil && len(list) == 0
}

func isSystemDir(dir string) bool {
	dir = strings.TrimSpace(dir)
	dir = strings.TrimRight(dir, "/")
	if dir == "" {
		return true
	}
	return slices.Contains(systemDirs, dir)
}
