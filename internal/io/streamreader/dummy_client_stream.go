package streamreader

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// DummyClientStream wrapper
type DummyClientStream struct {
}

// Header dummy
func (d *DummyClientStream) Header() (metadata.MD, error) { return nil, nil }

// Trailer dummy
func (d *DummyClientStream) Trailer() metadata.MD { return nil }

// CloseSend dummy
func (d *DummyClientStream) CloseSend() error { return nil }

// Context dummy
func (d *DummyClientStream) Context() context.Context { return nil }

// SendMsg dummy
func (d *DummyClientStream) SendMsg(m any) error { return nil }

// RecvMsg dummy
func (d *DummyClientStream) RecvMsg(m any) error { return nil }

var _ grpc.ClientStream = (*DummyClientStream)(nil)
