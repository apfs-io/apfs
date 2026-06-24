package models

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── NewProcessingState ────────────────────────────────────────────────────────

func TestNewProcessingState(t *testing.T) {
	ps := NewProcessingState("obj-1", "2", []string{"encode", "thumbnail"})
	require.NotNil(t, ps)
	assert.Equal(t, "obj-1", ps.ObjectID)
	assert.Equal(t, "2", ps.ManifestVersion)
	assert.Equal(t, ProcessingStatusPending, ps.Status)
	assert.Len(t, ps.Jobs, 2)
	for _, id := range []string{"encode", "thumbnail"} {
		require.Contains(t, ps.Jobs, id)
		assert.Equal(t, JobStatusPending, ps.Jobs[id].Status)
	}
	assert.False(t, ps.StartedAt.IsZero())
}

func TestNewProcessingState_NoJobs(t *testing.T) {
	ps := NewProcessingState("obj-2", "2", nil)
	assert.Empty(t, ps.Jobs)
}

// ── ComputeProgress ───────────────────────────────────────────────────────────

func TestProcessingState_ComputeProgress(t *testing.T) {
	ps := NewProcessingState("obj-1", "2", []string{"a", "b", "c", "d"})

	ps.ComputeProgress()
	assert.Equal(t, 0.0, ps.Progress, "all pending → 0%%")

	ps.Jobs["a"].Status = JobStatusCompleted
	ps.Jobs["b"].Status = JobStatusSkipped
	ps.ComputeProgress()
	assert.Equal(t, 0.5, ps.Progress, "2 of 4 terminal → 50%%")

	for _, j := range ps.Jobs {
		j.Status = JobStatusCompleted
	}
	ps.ComputeProgress()
	assert.Equal(t, 1.0, ps.Progress, "all completed → 100%%")
}

func TestProcessingState_ComputeProgress_Nil(t *testing.T) {
	(*ProcessingState)(nil).ComputeProgress() // must not panic
}

// ── ComputeStatus ─────────────────────────────────────────────────────────────

func TestProcessingState_ComputeStatus_AllPending(t *testing.T) {
	ps := NewProcessingState("x", "2", []string{"a", "b"})
	ps.ComputeStatus()
	assert.Equal(t, ProcessingStatusPending, ps.Status)
}

func TestProcessingState_ComputeStatus_Running(t *testing.T) {
	ps := NewProcessingState("x", "2", []string{"a", "b"})
	ps.Jobs["a"].Status = JobStatusRunning
	ps.ComputeStatus()
	assert.Equal(t, ProcessingStatusRunning, ps.Status)
}

func TestProcessingState_ComputeStatus_Completed(t *testing.T) {
	ps := NewProcessingState("x", "2", []string{"a", "b"})
	ps.Jobs["a"].Status = JobStatusCompleted
	ps.Jobs["b"].Status = JobStatusCompleted
	ps.ComputeStatus()
	assert.Equal(t, ProcessingStatusCompleted, ps.Status)
}

func TestProcessingState_ComputeStatus_Partial(t *testing.T) {
	ps := NewProcessingState("x", "2", []string{"a", "b"})
	ps.Jobs["a"].Status = JobStatusCompleted
	ps.Jobs["b"].Status = JobStatusFailed
	ps.ComputeStatus()
	assert.Equal(t, ProcessingStatusPartial, ps.Status)
}

func TestProcessingState_ComputeStatus_PartialWithSkipped(t *testing.T) {
	// Skipped counts as terminal; if no failures, it's completed, not partial.
	ps := NewProcessingState("x", "2", []string{"a", "b"})
	ps.Jobs["a"].Status = JobStatusCompleted
	ps.Jobs["b"].Status = JobStatusSkipped
	ps.ComputeStatus()
	assert.Equal(t, ProcessingStatusCompleted, ps.Status)
}

func TestProcessingState_ComputeStatus_MixedRunning(t *testing.T) {
	ps := NewProcessingState("x", "2", []string{"a", "b", "c"})
	ps.Jobs["a"].Status = JobStatusCompleted
	ps.Jobs["b"].Status = JobStatusPending // still waiting
	ps.Jobs["c"].Status = JobStatusFailed
	ps.ComputeStatus()
	// pending + failed + completed → still running (pipeline not done)
	assert.Equal(t, ProcessingStatusRunning, ps.Status)
}

// ── ProcessingStatus helpers ──────────────────────────────────────────────────

func TestProcessingStatus_IsTerminal(t *testing.T) {
	assert.False(t, ProcessingStatusPending.IsTerminal())
	assert.False(t, ProcessingStatusRunning.IsTerminal())
	assert.True(t, ProcessingStatusCompleted.IsTerminal())
	assert.True(t, ProcessingStatusPartial.IsTerminal())
	assert.True(t, ProcessingStatusFailed.IsTerminal())
}

func TestProcessingStatus_IsSuccess(t *testing.T) {
	assert.True(t, ProcessingStatusCompleted.IsSuccess())
	assert.True(t, ProcessingStatusPartial.IsSuccess())
	assert.False(t, ProcessingStatusFailed.IsSuccess())
	assert.False(t, ProcessingStatusPending.IsSuccess())
}

// ── JobStatus helpers ─────────────────────────────────────────────────────────

func TestJobStatus_IsTerminal(t *testing.T) {
	assert.False(t, JobStatusPending.IsTerminal())
	assert.False(t, JobStatusRunning.IsTerminal())
	assert.True(t, JobStatusCompleted.IsTerminal())
	assert.True(t, JobStatusFailed.IsTerminal())
	assert.True(t, JobStatusSkipped.IsTerminal())
}

// ── JobState transitions ──────────────────────────────────────────────────────

func TestJobState_MarkStarted(t *testing.T) {
	js := &JobState{Status: JobStatusPending}
	js.MarkStarted("worker-1")
	assert.Equal(t, JobStatusRunning, js.Status)
	assert.Equal(t, "worker-1", js.Worker)
	assert.NotNil(t, js.StartedAt)
}

func TestJobState_MarkCompleted(t *testing.T) {
	js := &JobState{Status: JobStatusRunning}
	js.MarkCompleted(map[string]any{"url": "https://example.com/out.mp4"})
	assert.Equal(t, JobStatusCompleted, js.Status)
	assert.Equal(t, 1.0, js.Progress)
	assert.NotNil(t, js.FinishedAt)
	assert.Equal(t, "https://example.com/out.mp4", js.Outputs["url"])
}

func TestJobState_MarkFailed(t *testing.T) {
	js := &JobState{Status: JobStatusRunning}
	js.MarkFailed(errors.New("disk full"))
	assert.Equal(t, JobStatusFailed, js.Status)
	assert.Equal(t, "disk full", js.Error)
	assert.Equal(t, 1, js.Attempts)
	assert.NotNil(t, js.FinishedAt)

	js.MarkFailed(errors.New("still broken"))
	assert.Equal(t, 2, js.Attempts)
}

func TestJobState_MarkFailed_NilError(t *testing.T) {
	js := &JobState{Status: JobStatusRunning}
	js.MarkFailed(nil)
	assert.Equal(t, JobStatusFailed, js.Status)
	assert.Empty(t, js.Error)
}

func TestJobState_MarkSkipped(t *testing.T) {
	js := &JobState{Status: JobStatusPending}
	js.MarkSkipped("upstream failed")
	assert.Equal(t, JobStatusSkipped, js.Status)
	assert.Equal(t, "upstream failed", js.Error)
	assert.NotNil(t, js.FinishedAt)
}

func TestJobState_ResetForRetry(t *testing.T) {
	js := &JobState{Status: JobStatusFailed, Worker: "w1", Error: "err", Attempts: 2}
	js.MarkFailed(errors.New("oops")) // increments Attempts to 3
	js.ResetForRetry()
	assert.Equal(t, JobStatusPending, js.Status)
	assert.Empty(t, js.Worker)
	assert.Empty(t, js.Error)
	assert.Nil(t, js.StartedAt)
	assert.Nil(t, js.FinishedAt)
	assert.Equal(t, 0.0, js.Progress)
}
