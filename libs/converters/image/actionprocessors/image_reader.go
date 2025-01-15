package actionprocessors

import (
	"image"
	"io"
)

// ImageReader basic image reading desctiption
type ImageReader interface {
	io.ReadSeekCloser
	Image() image.Image
	SetImage(img image.Image)
}
