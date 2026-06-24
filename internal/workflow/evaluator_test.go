package workflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/apfs-io/apfs/models"
)

func TestEvaluateIf_Empty(t *testing.T) {
	skip, err := EvaluateIf("", nil)
	assert.NoError(t, err)
	assert.False(t, skip)
}

func TestEvaluateIf_Literals(t *testing.T) {
	skip, err := EvaluateIf("${{ true }}", nil)
	assert.NoError(t, err)
	assert.False(t, skip) // true → do not skip

	skip, err = EvaluateIf("${{ false }}", nil)
	assert.NoError(t, err)
	assert.True(t, skip) // false → skip
}

func TestEvaluateIf_NumericComparison(t *testing.T) {
	now := time.Now()
	state := &models.ProcessingState{
		Jobs: map[string]*models.JobState{
			"probe": {
				Status:     models.JobStatusCompleted,
				Outputs:    map[string]any{"duration": float64(1800)},
				FinishedAt: &now,
			},
		},
	}

	// 1800 < 3600 → true → do not skip
	skip, err := EvaluateIf("${{ probe.outputs.duration < 3600 }}", state)
	assert.NoError(t, err)
	assert.False(t, skip)

	// 1800 > 3600 → false → skip
	skip, err = EvaluateIf("${{ probe.outputs.duration > 3600 }}", state)
	assert.NoError(t, err)
	assert.True(t, skip)
}

func TestEvaluateIf_StatusCheck(t *testing.T) {
	now := time.Now()
	state := &models.ProcessingState{
		Jobs: map[string]*models.JobState{
			"validate": {
				Status:     models.JobStatusCompleted,
				FinishedAt: &now,
			},
		},
	}

	skip, err := EvaluateIf("${{ validate.status == 'completed' }}", state)
	assert.NoError(t, err)
	assert.False(t, skip)

	skip, err = EvaluateIf("${{ validate.status == 'failed' }}", state)
	assert.NoError(t, err)
	assert.True(t, skip)
}
