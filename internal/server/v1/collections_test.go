package v1

import (
	"context"
	"errors"
	"testing"

	nc "github.com/geniusrabbit/notificationcenter/v2"
	"github.com/stretchr/testify/assert"

	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
)

func TestCollections(t *testing.T) {
	accessor, err := newKVAccessor("memory")
	assert.NoError(t, err)
	assert.NotNil(t, accessor)
}

func TestSendUploadEventSkipsNilObject(t *testing.T) {
	var published int
	s := &server{
		eventStream: nc.FuncPublisher(func(context.Context, ...any) error {
			published++
			return nil
		}),
	}
	s.sendUploadEvent(context.Background(), nil, errors.New("upload failed"))
	assert.Equal(t, 0, published)

	s.sendUploadEvent(context.Background(), &protocol.Object{Id: "abc"}, nil)
	assert.Equal(t, 1, published)
}
