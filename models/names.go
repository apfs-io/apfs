//
// @project apfs 2025
// @author Dmitry Ponomarev <demdxx@gmail.com> 2025
//

package models

import (
	"strings"
)

// Consts ...
const (
	OriginalFilename = "original"
)

// IsOriginal file name
func IsOriginal(name string) bool {
	return name == "" || name == "@" || name == OriginalFilename ||
		strings.HasPrefix(name, OriginalFilename+".")
}

// SourceFilename prepare ” -> @ = oroginal filename
func SourceFilename(name, ext string) string {
	if name == "" || name == "@" {
		name = OriginalFilename
	}
	if !strings.ContainsRune(name, '.') && ext != "" {
		name += "." + strings.TrimPrefix(ext, ".")
	}
	return name
}
