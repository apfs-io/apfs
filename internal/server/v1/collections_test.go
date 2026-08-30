package v1

import (
	"context"
	"errors"
	"testing"

	nc "github.com/geniusrabbit/notificationcenter/v2"
	"github.com/stretchr/testify/assert"

	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	"github.com/apfs-io/apfs/internal/workflow"
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

func TestUseWorkflowExecutor(t *testing.T) {
	exec := workflow.NewExecutor(nil, nil)
	jobs := map[string]*models.WorkflowJob{
		"thumb": {Steps: []*models.WorkflowStep{{Name: "Thumbnail", Uses: "procedure"}}},
	}

	assert.True(t, useWorkflowExecutor(&models.Workflow{Version: "3", Jobs: jobs}, exec),
		"version 3 with jobs must use the executor, not v1 ProcessTasks")
	assert.True(t, useWorkflowExecutor(&models.Workflow{Version: "2", Jobs: jobs}, exec))
	assert.True(t, useWorkflowExecutor(&models.Workflow{Version: "3.1", Jobs: jobs}, exec))
	assert.False(t, useWorkflowExecutor(&models.Workflow{Version: "3", Jobs: jobs}, nil),
		"no executor → fall back to v1")
	assert.False(t, useWorkflowExecutor(&models.Workflow{Version: "3"}, exec),
		"jobless workflow stays on v1")
	assert.False(t, useWorkflowExecutor(nil, exec))
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
