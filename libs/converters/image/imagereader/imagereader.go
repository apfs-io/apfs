package imagereader

import (
	"bytes"
	"image"
	"io"

	"github.com/pkg/errors"

	"github.com/apfs-io/apfs/internal/bytebufferpool"
)

var (
	errInvalidImageDecode = errors.New(`invalid image decode`)
)

type imageReader struct {
	contentType string
	quality     int
	img         image.Image
	buff        *bytes.Buffer
	readBuffer  *bytes.Reader
}

func NewImageReader(img image.Image, contentType string, quality int) *imageReader {
	return &imageReader{img: img, contentType: contentType, quality: quality}
}

func (ir *imageReader) Clone() *imageReader {
	return &imageReader{
		contentType: ir.contentType,
		quality:     ir.quality,
		img:         ir.img,
	}
}

func (ir *imageReader) Image() image.Image {
	return ir.img
}

func (ir *imageReader) SetImage(img image.Image) {
	ir.img = img
	bytebufferpool.Release(ir.buff)
	ir.buff = nil
	ir.readBuffer = nil
}

func (ir *imageReader) Read(p []byte) (int, error) {
	if err := ir.readBufferInit(); err != nil {
		return 0, err
	}
	if ir.readBuffer == nil {
		return 0, io.EOF
	}
	numReaded, err := ir.readBuffer.Read(p)
	if err != nil || numReaded <= 0 {
		ir.readBuffer = nil
	}
	return numReaded, err
}

func (ir *imageReader) Seek(offset int64, whence int) (int64, error) {
	if err := ir.readBufferInit(); err != nil {
		return 0, err
	}
	if ir.readBuffer == nil {
		return 0, io.EOF
	}
	return ir.readBuffer.Seek(offset, whence)
}

func (ir *imageReader) readBufferInit() error {
	if (ir.buff == nil || ir.buff.Len() == 0) && ir.img != nil {
		if ir.buff == nil {
			ir.buff = bytebufferpool.Acquire()
		} else {
			ir.buff.Reset()
		}
		err := Encode(ir.img, ir.buff, ir.contentType, ir.quality)
		if err != nil {
			return err
		}
	}
	if ir.readBuffer == nil && ir.buff != nil {
		ir.readBuffer = bytes.NewReader(ir.buff.Bytes())
	}
	return nil
}

func (ir *imageReader) Close() error {
	ir.img = nil
	ir.readBuffer = nil
	bytebufferpool.Release(ir.buff)
	return nil
}

var (
	_ io.ReadCloser = (*imageReader)(nil)
	_ io.ReadSeeker = (*imageReader)(nil)
)
