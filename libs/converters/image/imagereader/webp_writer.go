//go:build !gc && !webp
// +build !gc,!webp

package imagereader

import (
	"fmt"
	"image"
	"io"
)

func encodeWebp(out io.Writer, img image.Image) error {
	return fmt.Errorf("[image] webp saving is not supported")
}
