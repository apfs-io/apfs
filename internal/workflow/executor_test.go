package workflow

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storio "github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/models"
)

// ── Fake ExecutorStorage ──────────────────────────────────────────────────────

type fakeStorage struct {
	state     *models.ProcessingState
	meta      *models.Meta
	readErr   error
	writeErr  error
	written   map[string][]byte // path → content
	source    []byte
	readNames []string
	writeMeta []*models.ItemMeta
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{
		written: map[string][]byte{},
		source:  []byte("fake-image-data"),
	}
}

func (f *fakeStorage) ReadState(_ context.Context, _ storio.ObjectID) (*models.ProcessingState, error) {
	return f.state, f.readErr
}

func (f *fakeStorage) WriteState(_ context.Context, _ storio.ObjectID, s *models.ProcessingState) error {
	f.state = s
	return f.writeErr
}

func (f *fakeStorage) ReadMeta(_ context.Context, _ storio.ObjectID) (*models.Meta, error) {
	return f.meta, f.readErr
}

func (f *fakeStorage) WriteMeta(_ context.Context, _ storio.ObjectID, m *models.Meta) error {
	f.meta = m
	return f.writeErr
}

func (f *fakeStorage) WriteFile(_ context.Context, _ storio.ObjectID, path string, data interface{ Read([]byte) (int, error) }, im *models.ItemMeta) error {
	b, _ := io.ReadAll(data)
	f.written[path] = b
	f.writeMeta = append(f.writeMeta, im)
	return f.writeErr
}

func (f *fakeStorage) ReadFile(_ context.Context, _ storio.ObjectID, name string) (io.ReadCloser, error) {
	f.readNames = append(f.readNames, name)
	if f.readErr != nil {
		return nil, f.readErr
	}
	return io.NopCloser(bytes.NewReader(f.source)), nil
}

// ── Fake StepRunner ───────────────────────────────────────────────────────────

type fakeRunner struct {
	usesPrefix string
	output     StepOutput
	err        error
	callCount  int
	errUntil   int // return err for the first errUntil calls
}

func (r *fakeRunner) CanRun(step *models.WorkflowStep) bool {
	return strings.HasPrefix(step.Uses, r.usesPrefix)
}

func (r *fakeRunner) Run(_ context.Context, step *models.WorkflowStep, _ StepInput) (StepOutput, error) {
	r.callCount++
	if r.err != nil {
		// errUntil=0 means "always fail"; errUntil>0 means "fail for first N calls"
		if r.errUntil <= 0 || r.callCount <= r.errUntil {
			return StepOutput{}, r.err
		}
	}
	out := r.output
	if target, ok := step.With["target"].(string); ok && target != "" {
		out.TargetPath = target
		if out.ItemMeta == nil {
			out.ItemMeta = &models.ItemMeta{}
		}
	}
	return out, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func singleJobWorkflow(jobID string, uses string, opts ...func(*models.WorkflowJob)) *models.Workflow {
	job := &models.WorkflowJob{
		Steps: []*models.WorkflowStep{
			{Name: "step1", Uses: uses, With: map[string]any{"target": "out.jpg"}},
		},
	}
	for _, o := range opts {
		o(job)
	}
	return &models.Workflow{
		Version: "2",
		Jobs:    map[string]*models.WorkflowJob{jobID: job},
	}
}

func withOnFailure(policy string) func(*models.WorkflowJob) {
	return func(j *models.WorkflowJob) { j.OnFailure = policy }
}

func withIf(expr string) func(*models.WorkflowJob) {
	return func(j *models.WorkflowJob) { j.If = expr }
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestExecuteJob_HappyPath(t *testing.T) {
	store := newFakeStorage()
	runner := &fakeRunner{
		usesPrefix: "image/",
		output: StepOutput{
			Writer:     strings.NewReader("fake-image-data"),
			TargetPath: "out.jpg",
			ItemMeta:   &models.ItemMeta{},
			Outputs:    map[string]any{"width": 640},
		},
	}
	reg := NewRunnerRegistry()
	reg.Register(runner)

	wf := singleJobWorkflow("thumbnail", "image/resize")
	exec := NewExecutor(store, reg)
	err := exec.ExecuteJob(context.Background(), wf, "obj-1", "thumbnail", []string{"worker-a"})

	require.NoError(t, err)
	require.NotNil(t, store.state)
	assert.Equal(t, models.JobStatusCompleted, store.state.Jobs["thumbnail"].Status)
	assert.Equal(t, models.ProcessingStatusCompleted, store.state.Status)
	assert.Equal(t, 1.0, store.state.Progress)
	assert.Equal(t, []byte("fake-image-data"), store.written["out.jpg"])
}

func TestExecuteJob_SkippedByIfCondition(t *testing.T) {
	store := newFakeStorage()
	runner := &fakeRunner{usesPrefix: "image/"}
	reg := NewRunnerRegistry()
	reg.Register(runner)

	wf := singleJobWorkflow("thumbnail", "image/resize", withIf("false"))
	exec := NewExecutor(store, reg)
	err := exec.ExecuteJob(context.Background(), wf, "obj-1", "thumbnail", []string{"worker-a"})

	require.NoError(t, err)
	assert.Equal(t, models.JobStatusSkipped, store.state.Jobs["thumbnail"].Status)
	assert.Equal(t, 0, runner.callCount, "runner should not be called for skipped job")
}

func TestExecuteJob_AlreadyTerminal(t *testing.T) {
	store := newFakeStorage()
	store.state = &models.ProcessingState{
		ObjectID: "obj-1",
		Status:   models.ProcessingStatusCompleted,
		Jobs: map[string]*models.JobState{
			"thumbnail": {Status: models.JobStatusCompleted},
		},
	}
	runner := &fakeRunner{usesPrefix: "image/"}
	reg := NewRunnerRegistry()
	reg.Register(runner)

	wf := singleJobWorkflow("thumbnail", "image/resize")
	exec := NewExecutor(store, reg)
	err := exec.ExecuteJob(context.Background(), wf, "obj-1", "thumbnail", []string{"worker-a"})

	require.NoError(t, err)
	assert.Equal(t, 0, runner.callCount, "runner must not run again for a completed job")
}

func TestExecuteJob_MissingRunner(t *testing.T) {
	store := newFakeStorage()
	reg := NewRunnerRegistry() // empty registry

	wf := singleJobWorkflow("thumbnail", "image/resize")
	exec := NewExecutor(store, reg)
	// on-failure:fail handles the error internally; ExecuteJob returns nil
	err := exec.ExecuteJob(context.Background(), wf, "obj-1", "thumbnail", []string{"worker-a"})

	require.NoError(t, err)
	require.NotNil(t, store.state)
	assert.Equal(t, models.JobStatusFailed, store.state.Jobs["thumbnail"].Status)
}

func TestExecuteJob_MissingJobInWorkflow(t *testing.T) {
	store := newFakeStorage()
	reg := NewRunnerRegistry()

	wf := &models.Workflow{Version: "2", Jobs: map[string]*models.WorkflowJob{}}
	exec := NewExecutor(store, reg)
	err := exec.ExecuteJob(context.Background(), wf, "obj-1", "nonexistent", []string{"worker-a"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestStepSourceName(t *testing.T) {
	large := &models.ItemMeta{}
	large.UpdateName("large.jpg")
	meta := &models.Meta{Items: []*models.ItemMeta{large}}

	tests := []struct {
		name string
		step *models.WorkflowStep
		meta *models.Meta
		want string
	}{
		{
			name: "nil step",
			want: models.OriginalFilename,
		},
		{
			name: "first step omitted source",
			step: &models.WorkflowStep{With: map[string]any{"target": "large.jpg"}},
			want: models.OriginalFilename,
		},
		{
			name: "follow-up omitted source uses existing target",
			step: &models.WorkflowStep{With: map[string]any{"target": "large.jpg"}},
			meta: meta,
			want: "large.jpg",
		},
		{
			name: "explicit source",
			step: &models.WorkflowStep{With: map[string]any{"source": "large.jpg", "target": "clean.jpg"}},
			meta: meta,
			want: "large.jpg",
		},
		{
			name: "explicit source missing item",
			step: &models.WorkflowStep{With: map[string]any{"source": "missing.jpg"}},
			want: "missing.jpg",
		},
		{
			name: "original source stays prim",
			step: &models.WorkflowStep{With: map[string]any{"source": "prim", "target": "large.jpg"}},
			meta: meta,
			want: models.OriginalFilename,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, stepSourceName(tt.step, tt.meta))
		})
	}
}

func TestExecuteJob_ChainsSourceToExistingTarget(t *testing.T) {
	store := newFakeStorage()
	runner := &fakeRunner{
		usesPrefix: "procedure",
		output: StepOutput{
			Writer:   strings.NewReader("artifact"),
			ItemMeta: &models.ItemMeta{},
		},
	}
	reg := NewRunnerRegistry()
	reg.Register(runner)

	wf := &models.Workflow{
		Version: "2",
		Jobs: map[string]*models.WorkflowJob{
			"large": {
				Steps: []*models.WorkflowStep{
					{Name: "resize", Uses: "procedure", With: map[string]any{"target": "large.jpg"}},
					{Name: "strip", Uses: "procedure", With: map[string]any{"target": "large.jpg"}},
				},
			},
		},
	}
	exec := NewExecutor(store, reg)
	err := exec.ExecuteJob(context.Background(), wf, "obj-1", "large", nil)
	require.NoError(t, err)
	assert.Equal(t, []string{models.OriginalFilename, "large.jpg"}, store.readNames)
	assert.Equal(t, models.JobStatusCompleted, store.state.Jobs["large"].Status)
}

func TestExecuteJob_OnFailureContinue(t *testing.T) {
	store := newFakeStorage()
	runner := &fakeRunner{
		usesPrefix: "image/",
		err:        errors.New("conversion failed"),
	}
	reg := NewRunnerRegistry()
	reg.Register(runner)

	wf := singleJobWorkflow("thumbnail", "image/resize", withOnFailure("continue"))
	exec := NewExecutor(store, reg)
	err := exec.ExecuteJob(context.Background(), wf, "obj-1", "thumbnail", []string{"worker-a"})

	// on-failure:continue → executor swallows the error
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusFailed, store.state.Jobs["thumbnail"].Status)
	// A single failed job with on-failure:continue → partial overall
	assert.Equal(t, models.ProcessingStatusPartial, store.state.Status)
	assert.Nil(t, store.meta, "no artifacts written → skip WriteMeta")
}

func TestExecuteJob_OnFailureContinueWritesMeta(t *testing.T) {
	store := newFakeStorage()
	resize := &fakeRunner{
		usesPrefix: "procedure/resize",
		output: StepOutput{
			Writer:   strings.NewReader("resized"),
			ItemMeta: &models.ItemMeta{},
		},
	}
	strip := &fakeRunner{
		usesPrefix: "procedure/strip",
		err:        errors.New("strip failed"),
	}
	reg := NewRunnerRegistry()
	reg.Register(resize)
	reg.Register(strip)

	wf := &models.Workflow{
		Version: "2",
		Jobs: map[string]*models.WorkflowJob{
			"large": {
				OnFailure: "continue",
				Steps: []*models.WorkflowStep{
					{Name: "resize", Uses: "procedure/resize", With: map[string]any{"target": "large.jpg"}},
					{Name: "strip", Uses: "procedure/strip", With: map[string]any{"target": "large.jpg"}},
				},
			},
		},
	}
	exec := NewExecutor(store, reg)
	err := exec.ExecuteJob(context.Background(), wf, "obj-1", "large", nil)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusFailed, store.state.Jobs["large"].Status)
	assert.Equal(t, models.ProcessingStatusPartial, store.state.Status)
	require.NotNil(t, store.meta)
	assert.Equal(t, "2", store.meta.ManifestVersion)
	require.NotNil(t, store.meta.ItemByName("large.jpg"))
	assert.Equal(t, []byte("resized"), store.written["large.jpg"])
}

func TestExecuteJob_OnFailureFail_DownstreamSkipped(t *testing.T) {
	store := newFakeStorage()
	runner := &fakeRunner{
		usesPrefix: "image/",
		err:        errors.New("step error"),
	}
	reg := NewRunnerRegistry()
	reg.Register(runner)

	// Two jobs: "source" → "downstream" (needs source)
	wf := &models.Workflow{
		Version: "2",
		Jobs: map[string]*models.WorkflowJob{
			"source": {
				OnFailure: "fail",
				Steps:     []*models.WorkflowStep{{Name: "s", Uses: "image/resize", With: map[string]any{}}},
			},
			"downstream": {
				Needs:     []string{"source"},
				OnFailure: "fail",
				Steps:     []*models.WorkflowStep{{Name: "s", Uses: "image/resize", With: map[string]any{}}},
			},
		},
	}

	// Pre-seed state so downstream is visible
	store.state = models.NewProcessingState("obj-1", "2", []string{"source", "downstream"})

	exec := NewExecutor(store, reg)
	// on-failure:fail handles failures internally; ExecuteJob returns nil
	err := exec.ExecuteJob(context.Background(), wf, "obj-1", "source", []string{"worker-a"})

	require.NoError(t, err)
	assert.Equal(t, models.JobStatusFailed, store.state.Jobs["source"].Status)
	assert.Equal(t, models.JobStatusSkipped, store.state.Jobs["downstream"].Status)
	assert.Equal(t, models.ProcessingStatusFailed, store.state.Status)
}

func TestExecuteJob_OnFailureRetry_ExhaustedRetries(t *testing.T) {
	store := newFakeStorage()
	runner := &fakeRunner{
		usesPrefix: "image/",
		err:        errors.New("transient error"),
	}
	reg := NewRunnerRegistry()
	reg.Register(runner)

	wf := singleJobWorkflow("encode", "image/resize", withOnFailure("retry:2"))
	exec := NewExecutor(store, reg)

	// Simulate 2 previous attempts already recorded.
	store.state = models.NewProcessingState("obj-1", "2", []string{"encode"})
	store.state.Jobs["encode"].Attempts = 2

	err := exec.ExecuteJob(context.Background(), wf, "obj-1", "encode", []string{"worker-a"})

	// max retries (2) reached → final failure, no retry error
	require.NoError(t, err, "after max retries, on-failure:retry falls back to marking failed")
	assert.Equal(t, models.JobStatusFailed, store.state.Jobs["encode"].Status)
}

func TestExecuteJob_OnFailureRetry_SucceedsEventually(t *testing.T) {
	store := newFakeStorage()
	runner := &fakeRunner{
		usesPrefix: "image/",
		err:        errors.New("transient"),
		errUntil:   1, // fail only first call
		output:     StepOutput{Outputs: map[string]any{"done": true}},
	}
	reg := NewRunnerRegistry()
	reg.Register(runner)

	wf := singleJobWorkflow("encode", "image/resize", withOnFailure("retry:3"))
	exec := NewExecutor(store, reg)

	// First attempt: runner returns error → executor returns retry sentinel
	err := exec.ExecuteJob(context.Background(), wf, "obj-1", "encode", []string{"worker-a"})
	require.Error(t, err, "first attempt should return retry error")
	assert.Contains(t, err.Error(), "retry")
	// After ResetForRetry the job is pending again, ready for reschedule
	assert.Equal(t, models.JobStatusPending, store.state.Jobs["encode"].Status)

	// Second attempt: runner succeeds (callCount > errUntil=1)
	err = exec.ExecuteJob(context.Background(), wf, "obj-1", "encode", []string{"worker-a"})
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusCompleted, store.state.Jobs["encode"].Status)
}

// ── RunnerRegistry ────────────────────────────────────────────────────────────

func TestRunnerRegistry_FindAndRegister(t *testing.T) {
	reg := NewRunnerRegistry()
	r1 := &fakeRunner{usesPrefix: "image/"}
	r2 := &fakeRunner{usesPrefix: "ffmpeg/"}

	reg.Register(r1)
	reg.Register(r2)

	step := &models.WorkflowStep{Uses: "image/resize"}
	assert.Same(t, r1, reg.Find(step))

	step.Uses = "ffmpeg/encode"
	assert.Same(t, r2, reg.Find(step))

	step.Uses = "unknown/tool"
	assert.Nil(t, reg.Find(step))
}

func TestProcessObject_Version3SetsManifestVersion(t *testing.T) {
	store := newFakeStorage()
	store.meta = &models.Meta{Main: models.ItemMeta{Name: "orig.jpg", Type: models.TypeImage}}
	runner := &fakeRunner{
		usesPrefix: "procedure",
		output: StepOutput{
			Writer:     strings.NewReader("thumb-data"),
			TargetPath: "thumb.jpg",
			ItemMeta:   &models.ItemMeta{},
		},
	}
	reg := NewRunnerRegistry()
	reg.Register(runner)

	wf := &models.Workflow{
		Version: "3",
		Jobs: map[string]*models.WorkflowJob{
			"thumb": {
				Steps: []*models.WorkflowStep{
					{Uses: "procedure", With: map[string]any{"target": "thumb.jpg"}},
				},
			},
		},
	}
	exec := NewExecutor(store, reg)
	complete, err := exec.ProcessObject(context.Background(), wf, "obj-1", []string{"image"}, 0)
	require.NoError(t, err)
	assert.True(t, complete)
	assert.Equal(t, "3", store.meta.ManifestVersion)
	assert.Equal(t, "3", store.state.ManifestVersion)
	assert.Equal(t, 1, runner.callCount)

	complete, err = exec.ProcessObject(context.Background(), wf, "obj-1", []string{"image"}, 0)
	require.NoError(t, err)
	assert.True(t, complete)
	assert.Equal(t, 1, runner.callCount, "same revision must not re-run jobs")
}

func TestProcessObject_VersionMismatchReprocesses(t *testing.T) {
	store := newFakeStorage()
	store.meta = &models.Meta{
		ManifestVersion: "2",
		Main:            models.ItemMeta{Name: "orig.jpg", Type: models.TypeImage},
	}
	runner := &fakeRunner{
		usesPrefix: "procedure",
		output: StepOutput{
			Writer:     strings.NewReader("thumb-data"),
			TargetPath: "thumb.jpg",
			ItemMeta:   &models.ItemMeta{},
		},
	}
	reg := NewRunnerRegistry()
	reg.Register(runner)

	wf2 := &models.Workflow{
		Version: "2",
		Jobs: map[string]*models.WorkflowJob{
			"thumb": {
				Steps: []*models.WorkflowStep{
					{Uses: "procedure", With: map[string]any{"target": "thumb.jpg"}},
				},
			},
		},
	}
	exec := NewExecutor(store, reg)
	complete, err := exec.ProcessObject(context.Background(), wf2, "obj-1", []string{"image"}, 0)
	require.NoError(t, err)
	assert.True(t, complete)
	assert.Equal(t, "2", store.meta.ManifestVersion)
	assert.Equal(t, 1, runner.callCount)

	wf3 := &models.Workflow{
		Version: "3",
		Jobs:    wf2.Jobs,
	}
	complete, err = exec.ProcessObject(context.Background(), wf3, "obj-1", []string{"image"}, 0)
	require.NoError(t, err)
	assert.True(t, complete)
	assert.Equal(t, "3", store.meta.ManifestVersion)
	assert.Equal(t, "3", store.state.ManifestVersion)
	assert.Equal(t, 2, runner.callCount, "revision bump must re-run jobs once")

	complete, err = exec.ProcessObject(context.Background(), wf3, "obj-1", []string{"image"}, 0)
	require.NoError(t, err)
	assert.True(t, complete)
	assert.Equal(t, 2, runner.callCount, "matching revision must not loop")
}

func TestProcessObject_HappyPath(t *testing.T) {
	store := newFakeStorage()
	store.meta = &models.Meta{Main: models.ItemMeta{Name: "prim.jfif", Type: models.TypeImage}}
	runner := &fakeRunner{
		usesPrefix: "image/",
		output: StepOutput{
			Writer:     strings.NewReader("thumb-data"),
			TargetPath: "thumb.jpg",
			ItemMeta:   &models.ItemMeta{},
		},
	}
	reg := NewRunnerRegistry()
	reg.Register(runner)

	wf := &models.Workflow{
		Version: "2",
		Jobs: map[string]*models.WorkflowJob{
			"thumb": {
				Steps: []*models.WorkflowStep{
					{Uses: "image/resize", With: map[string]any{"target": "thumb"}},
				},
			},
			"small": {
				Steps: []*models.WorkflowStep{
					{Uses: "image/resize", With: map[string]any{"target": "small"}},
				},
			},
		},
	}
	exec := NewExecutor(store, reg)
	complete, err := exec.ProcessObject(context.Background(), wf, "obj-1", []string{"image"}, 0)
	require.NoError(t, err)
	assert.True(t, complete)
	assert.Equal(t, models.ProcessingStatusCompleted, store.state.Status)
	assert.NotNil(t, store.meta.ItemByName("thumb"))
	assert.NotNil(t, store.meta.ItemByName("small"))
	assert.Equal(t, 2, runner.callCount)
}

func TestProcessObject_EmptyJobsWritesCompletedState(t *testing.T) {
	store := newFakeStorage()
	exec := NewExecutor(store, NewRunnerRegistry())
	wf := &models.Workflow{Version: "2", Name: "default"}

	complete, err := exec.ProcessObject(context.Background(), wf, "obj-empty", nil, 0)
	require.NoError(t, err)
	assert.True(t, complete)
	require.NotNil(t, store.state)
	assert.Equal(t, "obj-empty", store.state.ObjectID)
	assert.Equal(t, "2", store.state.ManifestVersion)
	assert.Equal(t, models.ProcessingStatusCompleted, store.state.Status)
	assert.Equal(t, 1.0, store.state.Progress)
	assert.NotNil(t, store.state.FinishedAt)
}
