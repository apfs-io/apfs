package streamreader

import (
	"io"

	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
)

// ClientStreamReader provides wrapping to read data stream from GRPC stream
type ClientStreamReader struct {
	client protocol.ServiceAPI_GetClient
	buffer []byte
	offset int
}

// NewClientStreamReader creates new service client stream wrapper
func NewClientStreamReader(client protocol.ServiceAPI_GetClient, data []byte) *ClientStreamReader {
	return &ClientStreamReader{
		client: client,
		buffer: data,
	}
}

// Read implements io.Reader interface
func (c *ClientStreamReader) Read(p []byte) (n int, err error) {
	var obj *protocol.ObjectResponse
	switch {
	case c.offset < len(c.buffer):
		for i := 0; i < len(p); i++ {
			if c.offset >= len(c.buffer) {
				break
			}
			p[i] = c.buffer[c.offset]
			c.offset++
			n++
		}
	case c.client != nil:
		if obj, err = c.client.Recv(); obj != nil {
			content := obj.GetContent().GetContent()
			for i := 0; i < len(p); i++ {
				if len(content) <= i {
					break
				}
				p[i] = content[i]
				n++
			}
			// Save reveived content into buffer
			if n < len(content) {
				c.buffer = append(c.buffer[:0], content[n:]...)
				c.offset = 0
			}
		}
	default:
		err = io.EOF
	}
	return n, err
}

// Close implements io.Closer interface
func (c *ClientStreamReader) Close() (err error) {
	if c.client != nil {
		err = c.client.CloseSend()
		c.client = nil
	}
	return err
}

var _ io.ReadCloser = (*ClientStreamReader)(nil)
