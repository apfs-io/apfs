package v1

import (
	"context"
	"errors"
	"testing"

	nc "github.com/geniusrabbit/notificationcenter/v2"
	"github.com/stretchr/testify/assert"

	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	"github.com/apfs-io/apfs/models"
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

func TestV2StateBlocksCompletion(t *testing.T) {
	assert.False(t, v2StateBlocksCompletion(nil), "nil state after complete must not re-queue Update")

	completed := &models.ProcessingState{Status: models.ProcessingStatusCompleted}
	assert.False(t, v2StateBlocksCompletion(completed))

	partial := &models.ProcessingState{Status: models.ProcessingStatusPartial}
	assert.False(t, v2StateBlocksCompletion(partial))

	running := &models.ProcessingState{Status: models.ProcessingStatusRunning}
	assert.True(t, v2StateBlocksCompletion(running))

	pending := &models.ProcessingState{Status: models.ProcessingStatusPending}
	assert.True(t, v2StateBlocksCompletion(pending))

	failed := &models.ProcessingState{Status: models.ProcessingStatusFailed}
	assert.True(t, v2StateBlocksCompletion(failed))
}
