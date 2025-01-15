package imagereader

import (
	"image"
	"io"
	"strings"

	"github.com/disintegration/imaging"
)

// Encode image into the target type
func Encode(img image.Image, wr io.Writer, target string, quality int) (err error) {
	switch strings.ToLower(target) {
	case "image/png":
		err = imaging.Encode(wr, img, imaging.PNG)
	case "image/gif":
		err = imaging.Encode(wr, img, imaging.GIF)
	case "image/tiff":
		err = imaging.Encode(wr, img, imaging.TIFF)
	case "image/bmp":
		err = imaging.Encode(wr, img, imaging.BMP)
	case "image/webp":
		err = encodeWebp(wr, img)
	default: // .jpg, .jpeg
		if quality > 0 {
			err = imaging.Encode(wr, img, imaging.JPEG, imaging.JPEGQuality(quality))
		} else {
			err = imaging.Encode(wr, img, imaging.JPEG)
		}
	}
	return err
}
