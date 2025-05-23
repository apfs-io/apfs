package client

import (
	"context"
)

var (
	// CtxClientObject reference to the Client
	CtxClientObject = struct{ s string }{"apfs.client"}
)

// FromContext returns API client object
func FromContext(ctx context.Context) Client {
	if logObj := ctx.Value(CtxClientObject); logObj != nil {
		return logObj.(Client)
	}
	return nil
}

// WithClient puts client object to context
func WithClient(ctx context.Context, cli Client) context.Context {
	return context.WithValue(ctx, CtxClientObject, cli)
}
