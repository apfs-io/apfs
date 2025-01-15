package actionprocessors

import "strings"

// ContentTypeFromExt from file extension
func ContentTypeFromExt(fileExt string) string {
	switch strings.TrimPrefix(strings.ToLower(fileExt), ".") {
	case "jpeg", "jpg", "jpe":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "tiff":
		return "image/tiff"
	case "bmp":
		return "image/bmp"
	case "webp":
		return "image/webp"
	}
	return "image/png"
}
