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

func TestProcessFollowupAfter_TerminalDoesNotRequeue(t *testing.T) {
	partial := &models.ProcessingState{Status: models.ProcessingStatusPartial}
	completed := &models.ProcessingState{Status: models.ProcessingStatusCompleted}
	failed := &models.ProcessingState{Status: models.ProcessingStatusFailed}
	running := &models.ProcessingState{Status: models.ProcessingStatusRunning}

	assert.Equal(t, processMarkComplete, processFollowupAfter(true, partial),
		"complete + partial (pending artifacts after reload) must mark done, not Update")
	assert.Equal(t, processMarkComplete, processFollowupAfter(false, partial),
		"incomplete + partial must not livelock on next-step Update")
	assert.Equal(t, processMarkComplete, processFollowupAfter(true, completed))
	assert.Equal(t, processMarkComplete, processFollowupAfter(true, nil),
		"jobless complete with nil state")
	assert.Equal(t, processStopFailed, processFollowupAfter(false, failed))
	assert.Equal(t, processStopFailed, processFollowupAfter(true, failed))
	assert.Equal(t, processRequeueUpdate, processFollowupAfter(false, running))
	assert.Equal(t, processRequeueUpdate, processFollowupAfter(false, nil))
	assert.Equal(t, processRequeueUpdate, processFollowupAfter(true, running),
		"complete claimed but state still running → keep Update")
}

func TestShouldEnqueueUpdateFromHead(t *testing.T) {
	completed := &models.ProcessingState{Status: models.ProcessingStatusCompleted}
	partial := &models.ProcessingState{Status: models.ProcessingStatusPartial}
	running := &models.ProcessingState{Status: models.ProcessingStatusRunning}

	assert.False(t, shouldEnqueueUpdateFromHead(true, nil))
	assert.False(t, shouldEnqueueUpdateFromHead(true, running))
	assert.False(t, shouldEnqueueUpdateFromHead(false, completed),
		"inconsistent meta after successful complete must not re-queue")
	assert.False(t, shouldEnqueueUpdateFromHead(false, partial))
	assert.True(t, shouldEnqueueUpdateFromHead(false, running))
	assert.True(t, shouldEnqueueUpdateFromHead(false, nil))
}

func TestShouldDeleteExcessOnUpdate(t *testing.T) {
	wf := &models.Workflow{Version: "5"}
	completed := &models.ProcessingState{Status: models.ProcessingStatusCompleted, ManifestVersion: "5"}
	partial := &models.ProcessingState{Status: models.ProcessingStatusPartial, ManifestVersion: "5"}
	running := &models.ProcessingState{Status: models.ProcessingStatusRunning, ManifestVersion: "5"}
	stale := &models.ProcessingState{Status: models.ProcessingStatusCompleted, ManifestVersion: "4"}

	assert.True(t, shouldDeleteExcessOnUpdate(wf, nil))
	assert.True(t, shouldDeleteExcessOnUpdate(wf, running))
	assert.False(t, shouldDeleteExcessOnUpdate(wf, completed))
	assert.False(t, shouldDeleteExcessOnUpdate(wf, partial))
	assert.True(t, shouldDeleteExcessOnUpdate(wf, stale),
		"manifest revision bump still strips leftovers")
}
