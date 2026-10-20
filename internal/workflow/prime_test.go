package workflow

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apfs-io/apfs/models"
)

// pngHeader is enough for http.DetectContentType to report image/png.
const pngHeader = "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"

// freshRunner returns a new reader on every Run so multiple calls see data.
type freshRunner struct {
	usesPrefix string
	data       string
	calls      int
}

func (r *freshRunner) CanRun(step *models.WorkflowStep) bool {
	return strings.HasPrefix(step.Uses, r.usesPrefix)
}

func (r *freshRunner) Run(_ context.Context, step *models.WorkflowStep, _ StepInput) (StepOutput, error) {
	r.calls++
	out := StepOutput{Writer: strings.NewReader(r.data), Outputs: map[string]any{}}
	if target, ok := step.With["target"].(string); ok && target != "" {
		out.TargetPath = target
		out.ItemMeta = &models.ItemMeta{}
	}
	return out, nil
}

func primeWorkflow() *models.Workflow {
	return &models.Workflow{
		Version: "2",
		Prime: &models.WorkflowJob{
			Steps: []*models.WorkflowStep{{Name: "encode", Uses: "proc/encode"}},
		},
		Jobs: map[string]*models.WorkflowJob{
			"thumb": {Steps: []*models.WorkflowStep{
				{Name: "thumb", Uses: "proc/thumb", With: map[string]any{"target": "thumb.jpg"}},
			}},
		},
	}
}

func TestProcessObject_SkipByMainHeight(t *testing.T) {
	store := newFakeStorage()
	store.meta = &models.Meta{Main: models.ItemMeta{Name: "prim", NameExt: "mp4", Height: 480, Width: 854}}
	runner := &freshRunner{usesPrefix: "proc/", data: "video"}
	reg := NewRunnerRegistry()
	reg.Register(runner)
	exec := NewExecutor(store, reg)
	wf := &models.Workflow{
		Version: "6",
		Jobs: map[string]*models.WorkflowJob{
			"mp4-720p": {If: "${{ main.height > 720 }}", Steps: []*models.WorkflowStep{
				{Uses: "proc/transcode", With: map[string]any{"target": "720p.mp4"}},
			}},
			"mp4-360p": {If: "${{ main.height > 360 }}", Steps: []*models.WorkflowStep{
				{Uses: "proc/transcode", With: map[string]any{"target": "360p.mp4"}},
			}},
		},
	}

	complete, err := exec.ProcessObject(context.Background(), wf, "obj-1", nil, 0)
	require.NoError(t, err)
	assert.True(t, complete, "a skipped target must not block completion")
	assert.Equal(t, models.JobStatusSkipped, store.state.Jobs["mp4-720p"].Status)
	assert.Equal(t, models.JobStatusCompleted, store.state.Jobs["mp4-360p"].Status)
	assert.Equal(t, []string{"mp4-720p"}, store.meta.SkippedJobs)
	assert.Empty(t, store.meta.MissingJobTargets(wf))
	assert.True(t, store.meta.IsConsistent(wf))
	assert.Contains(t, store.written, "360p.mp4")
	assert.NotContains(t, store.written, "720p.mp4")
	assert.Equal(t, 1, runner.calls)

	// A taller original on re-processing runs the job and clears the skip.
	store.meta.Main.Height = 1080
	store.state = models.NewProcessingState("obj-1", wf.Version, wf.JobIDs())
	complete, err = exec.ProcessObject(context.Background(), wf, "obj-1", nil, 0)
	require.NoError(t, err)
	assert.True(t, complete)
	assert.Equal(t, models.JobStatusCompleted, store.state.Jobs["mp4-720p"].Status)
	assert.Empty(t, store.meta.SkippedJobs)
}

func TestBuildDAG_PrimeRunsFirst(t *testing.T) {
	wf := primeWorkflow()
	wf.Jobs["other"] = &models.WorkflowJob{}
	dag, err := BuildDAG(wf)
	require.NoError(t, err)
	assert.Equal(t, []string{models.PrimeJobID, "other", "thumb"}, dag.TopologicalOrder())
	assert.Equal(t, []string{"other", "thumb"}, dag.Downstream(models.PrimeJobID))

	state := models.NewProcessingState("obj", wf.Version, wf.JobIDs())
	assert.Equal(t, []string{models.PrimeJobID}, dag.ReadyJobs(state, nil))
	state.Jobs[models.PrimeJobID].Status = models.JobStatusCompleted
	assert.Equal(t, []string{"other", "thumb"}, dag.ReadyJobs(state, nil))
}

func TestBuildDAG_PrimeValidation(t *testing.T) {
	wf := primeWorkflow()
	wf.Jobs["thumb"].Needs = []string{models.PrimeJobID}
	_, err := BuildDAG(wf)
	assert.Error(t, err)

	wf = primeWorkflow()
	wf.Prime.Steps[0].With = map[string]any{"target": "x.mp4"}
	_, err = BuildDAG(wf)
	assert.Error(t, err)
}

func TestExecuteJob_PrimeReplacesOriginal(t *testing.T) {
	store := newFakeStorage()
	store.meta = &models.Meta{Main: models.ItemMeta{Name: "prim", NameExt: "jpg", Path: "prim.jpg"}}
	runner := &freshRunner{usesPrefix: "proc/", data: pngHeader + "payload"}
	reg := NewRunnerRegistry()
	reg.Register(runner)
	exec := NewExecutor(store, reg)
	wf := primeWorkflow()

	complete, err := exec.ProcessObject(context.Background(), wf, "obj-1", nil, 0)
	require.NoError(t, err)
	assert.True(t, complete)

	assert.Equal(t, []byte(pngHeader+"payload"), store.written["prim.png"])
	assert.Equal(t, models.OriginalFilename, store.readNames[0])
	assert.Equal(t, "prim.png", store.meta.Main.Fullname())
	assert.Equal(t, models.PrimeJobID, store.meta.Main.Role)
	require.Len(t, store.meta.Items, 1, "the original stays in Main, not Items")
	assert.Equal(t, "thumb", store.meta.Items[0].Name)
	assert.Equal(t, wf.PrimeHash(), store.meta.PrimeHash)
	assert.Equal(t, models.JobStatusCompleted, store.state.Jobs[models.PrimeJobID].Status)
	assert.Equal(t, models.JobStatusCompleted, store.state.Jobs["thumb"].Status)
	assert.Equal(t, 2, runner.calls)

	// Re-processing with the same prime config skips the encode.
	store.state = models.NewProcessingState("obj-1", wf.Version, wf.JobIDs())
	require.NoError(t, exec.ExecuteJob(context.Background(), wf, "obj-1", models.PrimeJobID, nil))
	assert.Equal(t, 2, runner.calls)
	assert.Equal(t, "already-encoded", store.state.Jobs[models.PrimeJobID].Outputs["skipped"])

	// A changed prime config re-encodes.
	wf.Prime.Steps[0].With = map[string]any{"crf": "20"}
	store.state = models.NewProcessingState("obj-1", wf.Version, wf.JobIDs())
	require.NoError(t, exec.ExecuteJob(context.Background(), wf, "obj-1", models.PrimeJobID, nil))
	assert.Equal(t, 3, runner.calls)
	assert.Equal(t, wf.PrimeHash(), store.meta.PrimeHash)
}
