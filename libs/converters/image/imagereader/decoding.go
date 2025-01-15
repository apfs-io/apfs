package imagereader

import (
	"io"

	"github.com/disintegration/imaging"
	"github.com/pkg/errors"
)

// Decode image from some image type
func Decode(in io.Reader, contentType string, quality int) (*imageReader, error) {
	switch v := in.(type) {
	case *imageReader:
		return v.Clone(), nil
	default:
		img, err := imaging.Decode(in, imaging.AutoOrientation(true))
		if err != nil {
			return nil, errors.Wrap(errInvalidImageDecode, err.Error())
		}
		return NewImageReader(img, contentType, quality), nil
	}
}
