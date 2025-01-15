package bytebufferpool

import (
	"bytes"
	"sync"
)

var pool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

// Acquire buffer object
func Acquire() *bytes.Buffer {
	return pool.Get().(*bytes.Buffer)
}

// Release buffer object
func Release(buff *bytes.Buffer) {
	if buff == nil {
		return
	}
	buff.Reset()
	pool.Put(buff)
}
