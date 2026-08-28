package ctxstatusstream

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apfs-io/apfs/models"
)

func TestEventFromState_Nil(t *testing.T) {
	assert.Nil(t, EventFromState(nil, true))
}

func TestEventFromState_CompletedForcesFinal(t *testing.T) {
	ps := models.NewProcessingState("obj-1", "2", []string{"a"})
	ps.Jobs["a"].Status = models.JobStatusCompleted
	ps.ComputeStatus()
	ps.ComputeProgress()

	ev := EventFromState(ps, false)
	require.NotNil(t, ev)
	assert.Equal(t, "obj-1", ev.ObjectID)
	assert.Equal(t, models.ProcessingStatusCompleted, ev.Status)
	assert.True(t, ev.Final)
	assert.Equal(t, 1, ev.Completed)
	assert.Equal(t, 1, ev.Total)
}

func TestEventFromState_FailedIncludesError(t *testing.T) {
	ps := models.NewProcessingState("obj-2", "2", []string{"a"})
	ps.Jobs["a"].MarkFailedCritical(assertErr("boom"))
	ps.ComputeStatus()

	ev := EventFromState(ps, false)
	require.NotNil(t, ev)
	assert.Equal(t, models.ProcessingStatusFailed, ev.Status)
	assert.Equal(t, "boom", ev.Error)
	assert.True(t, ev.Final)
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
