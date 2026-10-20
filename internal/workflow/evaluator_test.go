package workflow

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apfs-io/apfs/models"
)

func stateCtx(state *models.ProcessingState) EvalContext {
	return EvalContext{State: state}
}

func TestEvaluateIf_Empty(t *testing.T) {
	skip, err := EvaluateIf("", EvalContext{})
	assert.NoError(t, err)
	assert.False(t, skip)
}

func TestEvaluateIf_Literals(t *testing.T) {
	skip, err := EvaluateIf("${{ true }}", EvalContext{})
	assert.NoError(t, err)
	assert.False(t, skip) // true → do not skip

	skip, err = EvaluateIf("${{ false }}", EvalContext{})
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
	skip, err := EvaluateIf("${{ probe.outputs.duration < 3600 }}", stateCtx(state))
	assert.NoError(t, err)
	assert.False(t, skip)

	// 1800 > 3600 → false → skip
	skip, err = EvaluateIf("${{ probe.outputs.duration > 3600 }}", stateCtx(state))
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

	skip, err := EvaluateIf("${{ validate.status == 'completed' }}", stateCtx(state))
	assert.NoError(t, err)
	assert.False(t, skip)

	skip, err = EvaluateIf("${{ validate.status == 'failed' }}", stateCtx(state))
	assert.NoError(t, err)
	assert.True(t, skip)
}

func TestEvaluateIf_Conditions(t *testing.T) {
	ctx := EvalContext{
		State: &models.ProcessingState{Jobs: map[string]*models.JobState{
			"probe":   {Status: models.JobStatusCompleted, Outputs: map[string]any{"meta": json.RawMessage(`{"a":1}`), "fps": json.RawMessage(`30`)}},
			"1080p":   {Status: models.JobStatusSkipped},
			"mp4-720": {Status: models.JobStatusFailed},
		}},
		Meta: &models.Meta{Main: models.ItemMeta{
			Width: 1280, Height: 720, Duration: 30, Size: 1024, Bitrate: "2500000",
			Codec: "h264", ContentType: "video/mp4", Type: models.TypeVideo, NameExt: "mp4",
			Attributes: map[string]any{"lang": "en"},
		}},
	}
	tests := []struct {
		expr string
		run  bool
	}{
		{"${{ main.height > 720 }}", false},
		{"${{ main.height >= 720 }}", true},
		{"${{ main.height }} > 480 && ${{ main.codec }} == 'h264'", true},
		{"${{ main.height > 1080 || main.codec != 'h264' }}", false},
		{"${{ main.height > 1080 || main.width == 1280 }}", true},
		{"${{ !(main.height > 480) }}", false},
		{"${{ !main.missing }}", true},
		{"${{ main.width > main.height && (main.duration < 60 || main.size > 1000000000) }}", true},
		{"${{ main.bitrate > 2000000 }}", true},
		{"${{ main.type == 'video' && main.ext == 'mp4' && main.content_type == 'video/mp4' }}", true},
		{"${{ main.attributes.lang == 'en' }}", true},
		{"${{ main.codec == 'a<b' }}", false},
		{"${{ jobs.1080p.status == 'skipped' }}", true},
		{"${{ jobs.mp4-720.status != 'failed' }}", false},
		{"${{ probe.outputs.fps >= 30 }}", true},
		{"${{ jobs.unknown.status == null }}", true},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			skip, err := EvaluateIf(tc.expr, ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.run, !skip)
		})
	}
}

func TestEvaluateIf_NilMetaIsFalse(t *testing.T) {
	skip, err := EvaluateIf("${{ main.height > 720 }}", EvalContext{})
	require.NoError(t, err)
	assert.True(t, skip)

	skip, err = EvaluateIf("${{ main.height <= 720 }}", EvalContext{Meta: &models.Meta{}})
	require.NoError(t, err)
	assert.True(t, skip, "unknown height must not pass either comparison")
}

func TestEvaluateIf_SyntaxErrors(t *testing.T) {
	for _, expr := range []string{
		"${{ main.height > }}",
		"${{ (main.height > 1 }}",
		"${{ main.height > 1 ) }}",
		"${{ main.codec == 'h264 }}",
		"${{ unknownref }}",
		"${{ main.height = 1 }}",
	} {
		_, err := EvaluateIf(expr, EvalContext{Meta: &models.Meta{}})
		assert.Error(t, err, expr)
	}
}
