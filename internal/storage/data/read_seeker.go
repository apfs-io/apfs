package data

import (
	"io"
)

// ToReadSeeker extends reader with seeker
func ToReadSeeker(reader io.Reader) (io.ReadSeeker, error) {
	if seeker, ok := reader.(io.ReadSeeker); ok {
		return seeker, nil
	}
	buff := &buffer{}
	_, err := io.Copy(buff, reader)
	return buff, err
}

type buffer struct {
	data   []byte
	offset int64
}

func (buff *buffer) Reset() {
	buff.data = buff.data[:0]
}

func (buff *buffer) Len() int64 {
	return int64(len(buff.data))
}

func (buff *buffer) Write(p []byte) (n int, err error) {
	buff.data = append(buff.data, p...)
	return len(p), nil
}

func (buff *buffer) Read(p []byte) (n int, err error) {
	if buff.offset >= buff.Len() {
		return 0, io.EOF
	}
	for n = 0; buff.offset < buff.Len() && n < len(p); n++ {
		p[n] = buff.data[buff.offset]
		buff.offset++
	}
	return n, err
}

func (buff *buffer) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset = buff.offset + offset
	case io.SeekEnd:
		offset = buff.Len() + offset
	}
	if buff.offset = offset; offset > buff.Len() {
		buff.offset = buff.Len()
	}
	return buff.offset, nil
}

var (
	_ io.Reader = (*buffer)(nil)
	_ io.Seeker = (*buffer)(nil)
	_ io.Writer = (*buffer)(nil)
)
