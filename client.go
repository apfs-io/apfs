//
// @project apfs 2018 - 2022
// @author Dmitry Ponomarev <demdxx@gmail.com> 2018 - 2022
//

package apfs

import (
	"context"

	"google.golang.org/grpc"

	"github.com/apfs-io/apfs/libs/client"
	"github.com/apfs-io/apfs/libs/storerrors"
)

// NewClient to apfs
func NewClient(ctx context.Context, address string, opts ...grpc.DialOption) (Client, error) {
	return client.Open(ctx, address, opts...)
}

// NewGroupClient wrapper
func NewGroupClient(name string, cli Client) *GroupClient {
	return client.NewGroupClient(name, cli)
}

// ContextGet returns API client object
func ContextGet(ctx context.Context) Client {
	return client.ContextGet(ctx)
}

// WithClient puts client object to context
func WithClient(ctx context.Context, cli Client) context.Context {
	return client.WithClient(ctx, cli)
}

// IsNotFound error object
func IsNotFound(err error) bool {
	return storerrors.IsNotFound(err)
}
