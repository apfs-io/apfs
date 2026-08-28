package client

// Preview is the display asset for an object: original image bytes or an SVG icon.
type Preview struct {
	ContentType string
	Content     []byte
}
