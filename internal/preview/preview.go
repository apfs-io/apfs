package preview

import (
	"embed"
	"strings"

	"github.com/apfs-io/apfs/models"
)

//go:embed icons/*.svg
var icons embed.FS

const (
	KindVideo   = "video"
	KindAudio   = "audio"
	KindHTML    = "html"
	KindPDF     = "pdf"
	KindArchive = "archive"
	KindFile    = "file"

	SVGContentType = "image/svg+xml"
)

// IsImage reports whether contentType is an image (including SVG).
func IsImage(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	return ct == "image" || strings.HasPrefix(ct, "image/")
}

// KindFromContentType maps a MIME type to an icon kind.
func KindFromContentType(contentType string) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}

	switch {
	case ct == "application/pdf":
		return KindPDF
	case isArchive(ct):
		return KindArchive
	case ct == "text/html" || ct == "application/xhtml+xml" || ct == "htmlarch" ||
		strings.Contains(ct, "html"):
		return KindHTML
	}

	switch models.ObjectTypeByContentType(ct) {
	case models.TypeVideo:
		return KindVideo
	case models.TypeAudio:
		return KindAudio
	case models.TypeHTMLArchType:
		return KindHTML
	default:
		return KindFile
	}
}

func isArchive(ct string) bool {
	switch ct {
	case "application/zip",
		"application/x-zip-compressed",
		"application/gzip",
		"application/x-gzip",
		"application/x-tar",
		"application/x-gtar",
		"application/x-7z-compressed",
		"application/x-rar-compressed",
		"application/vnd.rar",
		"application/x-bzip2",
		"application/x-xz":
		return true
	}
	return strings.Contains(ct, "zip") || strings.Contains(ct, "compressed")
}

// Icon returns the embedded SVG for kind and image/svg+xml.
// Unknown kinds fall back to file.svg.
func Icon(kind string) (contentType string, data []byte) {
	name := kind
	if name == "" {
		name = KindFile
	}
	data, err := icons.ReadFile("icons/" + name + ".svg")
	if err != nil {
		data, err = icons.ReadFile("icons/" + KindFile + ".svg")
		if err != nil {
			return SVGContentType, []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"/>`)
		}
	}
	return SVGContentType, data
}

// IconForContentType returns the SVG icon for a non-image content type.
func IconForContentType(contentType string) (string, []byte) {
	return Icon(KindFromContentType(contentType))
}
