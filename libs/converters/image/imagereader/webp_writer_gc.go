//go:build gc || webp
// +build gc webp

package imagereader

import (
	"fmt"
	"image"
	"io"
	// 	"github.com/harukasan/go-libwebp/webp"
)

func encodeWebp(out io.Writer, img image.Image) error {
	// 	return webp.EncodeRGBA(w, img.(*image.RGBA), config)
	return fmt.Errorf("[image] webp saving is not supported")
}
