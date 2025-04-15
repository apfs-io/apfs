//
// @project apfs 2018 - 2022, 2025
// @author Dmitry Ponomarev <demdxx@gmail.com> 2018 - 2022, 2025
//

package apfs

import (
	"context"

	"google.golang.org/grpc"

	"github.com/apfs-io/apfs/libs/client"
	"github.com/apfs-io/apfs/libs/storerrors"
)

// Connect new client to disk service
// address should be in format tcp://host:port/default-group-name
// Scheme tcp:// or dns:// is required
func Connect(ctx context.Context, address string, opts ...grpc.DialOption) (Client, error) {
	return client.Connect(ctx, address, opts...)
}

// FromContext returns API client object
func FromContext(ctx context.Context) Client {
	return client.FromContext(ctx)
}

// WithClient puts client object to context
func WithClient(ctx context.Context, cli Client) context.Context {
	return client.WithClient(ctx, cli)
}

// IsNotFound error object
func IsNotFound(err error) bool {
	return storerrors.IsNotFound(err)
}
