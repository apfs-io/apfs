package client

import (
	"context"
	"io"
	"strings"

	"google.golang.org/grpc"

	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	"github.com/apfs-io/apfs/models"
)

// GroupClient storage wrapper
type GroupClient struct {
	groupName string
	client    Client
}

// ConnectGroupClient wrapper
func ConnectGroupClient(ctx context.Context, name, address string, opts ...grpc.DialOption) (*GroupClient, error) {
	client, err := Open(ctx, address, opts...)
	if err != nil {
		return nil, err
	}
	return NewGroupClient(name, client), nil
}

// NewGroupClient wrapper
func NewGroupClient(name string, client Client) *GroupClient {
	return &GroupClient{
		groupName: name,
		client:    client,
	}
}

// Refresh returns information about header without content
func (gr *GroupClient) Refresh(ctx context.Context, id *protocol.ObjectID, opts ...grpc.CallOption) error {
	return gr.client.Refresh(ctx, id, opts...)
}

// Head returns information about header without content
func (gr *GroupClient) Head(ctx context.Context, id *protocol.ObjectID, opts ...grpc.CallOption) (*models.Object, error) {
	if !strings.HasPrefix(id.Id, gr.groupName) {
		id = &protocol.ObjectID{
			Id:   gr.groupName + "/" + strings.TrimLeft(id.Id, "/"),
			Name: id.Name,
		}
	}
	return gr.client.Head(ctx, id, opts...)
}

// Get file object from storage
func (gr *GroupClient) Get(ctx context.Context, id *protocol.ObjectID, opts ...grpc.CallOption) (obj *models.Object, reader io.ReadCloser, err error) {
	if !strings.HasPrefix(id.Id, gr.groupName) {
		id = &protocol.ObjectID{
			Id:   gr.groupName + "/" + strings.TrimLeft(id.Id, "/"),
			Name: id.Name,
		}
	}
	return gr.client.Get(ctx, id, opts...)
}

// SetManifest of the group
func (gr *GroupClient) SetManifest(ctx context.Context, manifest *models.Manifest, opts ...grpc.CallOption) error {
	return gr.client.SetManifest(ctx, gr.groupName, manifest, opts...)
}

// GetManifest of the group
func (gr *GroupClient) GetManifest(ctx context.Context, opts ...grpc.CallOption) (*models.Manifest, error) {
	return gr.client.GetManifest(ctx, gr.groupName, opts...)
}

// UploadFile object into storage
func (gr *GroupClient) UploadFile(ctx context.Context, id, filepath string, tags []string, overwrite bool, opts ...grpc.CallOption) (*models.Object, error) {
	return gr.client.UploadFile(ctx, gr.groupName, id, filepath, tags, overwrite, opts...)
}

// Upload file object into storage
func (gr *GroupClient) Upload(ctx context.Context, id string, data io.Reader, tags []string, overwrite bool, opts ...grpc.CallOption) (*models.Object, error) {
	return gr.client.Upload(ctx, gr.groupName, id, data, tags, overwrite, opts...)
}

// Delete object from storage
func (gr *GroupClient) Delete(ctx context.Context, id any, opts ...grpc.CallOption) error {
	switch v := id.(type) {
	case string:
		id = prepareID(v, gr.groupName)
	case *protocol.ObjectIDNames:
		if !strings.HasPrefix(v.Id, gr.groupName) {
			id = &protocol.ObjectIDNames{
				Id:    prepareID(v.Id, gr.groupName),
				Names: v.Names,
			}
		}
	case *protocol.ObjectID:
		if !strings.HasPrefix(v.Id, gr.groupName) {
			id = &protocol.ObjectID{
				Id:   gr.groupName + "/" + strings.TrimLeft(v.Id, "/"),
				Name: v.Name,
			}
		}
	case *protocol.Object:
		if !strings.HasPrefix(v.Id, gr.groupName) && v.Bucket == "" {
			v.Bucket = gr.groupName
		}
	default:
		return ErrInvalidParams
	}
	return gr.client.Delete(ctx, id, opts...)
}

// Close group wrapper
func (gr *GroupClient) Close() error {
	return gr.client.Close()
}

func prepareID(id, bucket string) string {
	if !strings.HasPrefix(id, bucket) {
		return bucket + "/" + strings.TrimLeft(id, "/")
	}
	return id
}
