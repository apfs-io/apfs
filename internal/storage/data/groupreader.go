//
// @project apfs 2017 - 2018
// @author Dmitry Ponomarev <demdxx@gmail.com> 2017 - 2018
//

package data

import "io"

// GroupReader object
type GroupReader struct {
	skip    int
	readers []io.Reader
}

// NewGroupReader by group slice
func NewGroupReader(readers ...io.Reader) *GroupReader {
	return &GroupReader{
		skip:    0,
		readers: readers,
	}
}

// Read next block from several readers
func (gr *GroupReader) Read(p []byte) (n int, err error) {
	if gr.skip >= len(gr.readers) {
		return 0, io.EOF
	}

	for gr.skip < len(gr.readers) {
		if gr.readers[gr.skip] == nil {
			gr.skip++
			continue
		}
		n, err = gr.readers[gr.skip].Read(p)
		if err == io.EOF {
			gr.skip++
			if gr.skip < len(gr.readers) {
				err = nil
			}
		}
		if n > 0 {
			break
		}
	}
	return
}

// Close all readers which supports io.Closer interface
func (gr *GroupReader) Close() (err error) {
	for _, r := range gr.readers {
		if r == nil {
			continue
		}
		if cl, _ := r.(io.Closer); cl != nil {
			if err = cl.Close(); err != nil {
				break
			}
		}
	}
	return
}
