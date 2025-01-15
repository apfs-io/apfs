package streamreader

import (
	"bytes"
	"io"
	"testing"

	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
)

type dummyDataStream struct {
	DummyClientStream
	buffer []byte
	idx    int
}

func (s *dummyDataStream) Recv() (*protocol.ObjectResponse, error) {
	if s.idx >= len(s.buffer) {
		return nil, io.EOF
	}
	s.idx++
	return &protocol.ObjectResponse{
		Object: &protocol.ObjectResponse_Content{
			Content: &protocol.DataContent{
				Content: []byte{s.buffer[s.idx-1]},
			},
		},
	}, nil
}

func TestStreamReader(t *testing.T) {
	stream := &dummyDataStream{buffer: []byte("abcdef")}
	reader := NewClientStreamReader(stream, []byte("x-"))
	data, err := io.ReadAll(reader)

	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal([]byte("x-abcdef"), data) {
		t.Error("invalid read response")
	}

	if err = reader.Close(); err != nil {
		t.Error(err)
	}
}
