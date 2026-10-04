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

func TestSnapshotDerivedItemsKeepsNames(t *testing.T) {
	meta := &models.Meta{
		Items: []*models.ItemMeta{
			{Name: "1080p", NameExt: "mp4"},
			{Name: "preview", NameExt: "jpg"},
		},
		Attributes: map[string]any{"key": "val"},
	}
	items := snapshotDerivedItems(meta)
	assert.Equal(t, []string{"1080p", "preview"}, []string{items[0].Name, items[1].Name})
	assert.Empty(t, meta.Items)
	assert.Nil(t, meta.Attributes)
	assert.Nil(t, snapshotDerivedItems(nil))
}

func TestNewRefreshProcessingStateResetsJobs(t *testing.T) {
	wf := &models.Workflow{
		Version: "3",
		Jobs: map[string]*models.WorkflowJob{
			"mp4-1080p": {},
			"preview":   {},
		},
	}
	state := newRefreshProcessingState("video/obj", wf)
	assert.Equal(t, "video/obj", state.ObjectID)
	assert.Equal(t, "3", state.ManifestVersion)
	assert.Equal(t, models.ProcessingStatusPending, state.Status)
	assert.Equal(t, models.JobStatusPending, state.Jobs["mp4-1080p"].Status)
	assert.Equal(t, models.JobStatusPending, state.Jobs["preview"].Status)

	empty := newRefreshProcessingState("video/obj", nil)
	assert.Equal(t, models.ProcessingStatusPending, empty.Status)
	assert.Empty(t, empty.ManifestVersion)
	assert.Empty(t, empty.Jobs)
}

func TestDerivedItemsForEventRefreshVsUpdate(t *testing.T) {
	wf := &models.Workflow{
		Version: "3",
		Jobs: map[string]*models.WorkflowJob{
			"mp4": {
				Steps: []*models.WorkflowStep{
					{With: map[string]any{"target": "1080p.mp4"}},
				},
			},
		},
	}
	prior := &models.ProcessingState{
		Status:          models.ProcessingStatusPartial,
		ManifestVersion: "3",
	}
	meta := &models.Meta{
		Items: []*models.ItemMeta{
			{Name: "1080p", NameExt: "mp4"},
		},
	}

	updateItems := derivedItemsForEvent(models.UpdateEventType, meta, wf, prior)
	assert.Empty(t, updateItems, "successful terminal Update must not strip workflow targets")
	assert.Len(t, meta.Items, 1, "Update must leave meta items in place")

	refreshItems := derivedItemsForEvent(models.RefreshEventType, meta, wf, prior)
	assert.Len(t, refreshItems, 1)
	assert.Equal(t, "1080p.mp4", refreshItems[0].Fullname())
	assert.Empty(t, meta.Items, "Refresh clears derived items after the snapshot")
}
