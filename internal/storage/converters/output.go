package converters

import (
	"bytes"
	"errors"
	"io"
	"reflect"

	"github.com/apfs-io/apfs/models"
)

var errOutputStreamCantBeOverrited = errors.New(`output stream can't be overrited`)

type output struct {
	meta *models.ItemMeta
	out  io.Reader
}

// NewOutput interface wrapper
func NewOutput(meta *models.ItemMeta) Output {
	return &output{meta: meta}
}

func (out *output) Meta() *models.ItemMeta {
	if out.meta == nil {
		out.meta = &models.ItemMeta{}
	}
	return out.meta
}

func (out *output) SetOutput(outStream io.Reader) error {
	if out.out != nil {
		return errOutputStreamCantBeOverrited
	}
	out.out = outStream
	return nil
}

func (out *output) OutputWriter() (io.Writer, error) {
	if out.out == nil {
		out.out = &bytes.Buffer{}
	}
	switch w := out.out.(type) {
	case io.Writer:
		return w, nil
	}
	return nil, errOutputStreamCantBeOverrited
}

func (out *output) ObjectReader() io.Reader {
	if out.out == nil {
		return nil
	}
	return out.out
}

func (out *output) IsEqual(in Input) bool {
	if out.out == nil {
		return false
	}
	return ptr(in.ObjectReader()) != ptr(out.out)
}

func ptr(v any) uintptr {
	_v := reflect.ValueOf(v)
	for _v.IsValid() && (_v.Kind() == reflect.Interface || _v.Kind() == reflect.Pointer) && !_v.IsNil() {
		_v = _v.Elem()
	}
	if !_v.IsValid() || ((_v.Kind() == reflect.Interface || _v.Kind() == reflect.Pointer) && !_v.IsNil()) {
		return 0
	}
	if _v.CanAddr() {
		return _v.Addr().Pointer()
	}
	return 0
}
