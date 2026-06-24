package workflow

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/apfs-io/apfs/internal/context/ctxlogger"
	storio "github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/models"
)

// ExecutorStorage is the minimal storage interface required by the Executor.
type ExecutorStorage interface {
	// ReadState reads the current ProcessingState for an object.
	ReadState(ctx context.Context, id storio.ObjectID) (*models.ProcessingState, error)
	// WriteState persists a ProcessingState for an object.
	WriteState(ctx context.Context, id storio.ObjectID, state *models.ProcessingState) error
	// WriteFile writes data to a relative path inside an object scope.
	WriteFile(ctx context.Context, id storio.ObjectID, path string, data interface{ Read([]byte) (int, error) }, meta *models.ItemMeta) error
	// ReadFile opens a named subfile inside an object scope for reading.
	ReadFile(ctx context.Context, id storio.ObjectID, name string) (io.ReadCloser, error)
	// ReadMeta reads the current Meta for an object.
	ReadMeta(ctx context.Context, id storio.ObjectID) (*models.Meta, error)
	// WriteMeta persists updated Meta for an object.
	WriteMeta(ctx context.Context, id storio.ObjectID, meta *models.Meta) error
}

// Executor runs a single job within a workflow for a given object.
// It is intended to be called by a worker process after receiving a job-dispatch
// event from notificationcenter.
type Executor struct {
	storage  ExecutorStorage
	registry *RunnerRegistry
}

// NewExecutor creates an Executor with the given storage and runner registry.
func NewExecutor(storage ExecutorStorage, registry *RunnerRegistry) *Executor {
	return &Executor{storage: storage, registry: registry}
}

// ExecuteJob runs the specified job for the object identified by objectID.
// workerTags identifies this worker instance (e.g. ["gpu","large"]) and is
// stored in the job state for observability. Pass nil or empty to indicate
// an untagged worker.
//
// It:
//  1. Loads the current ProcessingState and Meta.
//  2. Evaluates the job's if: condition (skips if false).
//  3. Evaluates the on-failure policy.
//  4. Runs each step in order.
//  5. Persists the updated state and meta.
func (e *Executor) ExecuteJob(ctx context.Context, w *models.Workflow, objectID string, jobID string, workerTags []string) error {
	workerLabel := strings.Join(workerTags, ",")
	log := ctxlogger.Get(ctx).With(
		zap.String("object_id", objectID),
		zap.String("job_id", jobID),
		zap.String("worker", workerLabel),
	)

	id := storio.ObjectIDType(objectID)

	// Load state
	state, err := e.storage.ReadState(ctx, id)
	if err != nil {
		return fmt.Errorf("executor: load state: %w", err)
	}
	if state == nil {
		state = models.NewProcessingState(objectID, w.Version, w.JobIDs())
	}

	js, ok := state.Jobs[jobID]
	if !ok || js == nil {
		js = &models.JobState{Status: models.JobStatusPending}
		state.Jobs[jobID] = js
	}

	// Already done?
	if js.Status.IsTerminal() {
		return nil
	}

	job, ok := w.Jobs[jobID]
	if !ok || job == nil {
		return fmt.Errorf("executor: job %q not found in workflow", jobID)
	}

	// Collect upstream outputs for evaluator and template resolution
	jobOutputs := collectOutputs(state)

	// Evaluate if: condition
	skip, err := EvaluateIf(job.If, state)
	if err != nil {
		log.Warn("if condition evaluation failed", zap.Error(err))
		skip = false
	}
	if skip {
		js.MarkSkipped("if condition evaluated to false")
		state.UpdatedAt = time.Now()
		state.ComputeProgress()
		state.ComputeStatus()
		return e.storage.WriteState(ctx, id, state)
	}

	// Mark started
	js.MarkStarted(workerLabel)
	state.Status = models.ProcessingStatusRunning
	state.UpdatedAt = time.Now()
	if err := e.storage.WriteState(ctx, id, state); err != nil {
		log.Warn("write state before job start", zap.Error(err))
	}

	// Load meta
	meta, err := e.storage.ReadMeta(ctx, id)
	if err != nil {
		return fmt.Errorf("executor: load meta: %w", err)
	}
	if meta == nil {
		meta = &models.Meta{}
	}

	// Apply timeout
	jobCtx := ctx
	if timeout := job.Timeout(); timeout > 0 {
		var cancel context.CancelFunc
		jobCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Execute steps
	jobErr := e.runSteps(jobCtx, job, jobID, id, meta, jobOutputs, js, log)

	// Handle failure policy
	fp := job.FailurePolicy()
	if jobErr != nil {
		switch fp & 0x0F {
		case models.FailurePolicyContinue:
			log.Warn("job failed (on-failure:continue)", zap.Error(jobErr))
			js.MarkFailed(jobErr)
		case models.FailurePolicyRetry:
			maxRetries := fp.MaxRetries()
			if js.Attempts < maxRetries {
				log.Warn("job failed, will retry",
					zap.Error(jobErr), zap.Int("attempts", js.Attempts), zap.Int("max", maxRetries))
				js.ResetForRetry()
				state.UpdatedAt = time.Now()
				_ = e.storage.WriteState(ctx, id, state)
				return fmt.Errorf("executor: retry job %q: %w", jobID, jobErr)
			}
			log.Error("job failed, max retries reached", zap.Error(jobErr))
			js.MarkFailed(jobErr)
		default: // FailurePolicyFail
			log.Error("job failed (on-failure:fail)", zap.Error(jobErr))
			js.MarkFailed(jobErr)
			// Mark downstream jobs as skipped
			dag, dagErr := BuildDAG(w)
			if dagErr == nil {
				for _, downID := range dag.Downstream(jobID) {
					if djs, ok := state.Jobs[downID]; ok && djs.Status == models.JobStatusPending {
						djs.MarkSkipped(fmt.Sprintf("upstream job %q failed", jobID))
					}
				}
			}
		}
	} else {
		js.MarkCompleted(js.Outputs)
		meta.ManifestVersion = w.Version
		if err := e.storage.WriteMeta(ctx, id, meta); err != nil {
			log.Warn("write meta after job complete", zap.Error(err))
		}
	}

	state.UpdatedAt = time.Now()
	state.ComputeProgress()
	state.ComputeStatus()
	if state.Status.IsTerminal() {
		now := time.Now()
		state.FinishedAt = &now
	}
	return e.storage.WriteState(ctx, id, state)
}

// runSteps executes all steps in the job in order.
func (e *Executor) runSteps(
	ctx context.Context,
	job *models.WorkflowJob,
	jobID string,
	id storio.ObjectID,
	meta *models.Meta,
	jobOutputs map[string]map[string]any,
	js *models.JobState,
	log *zap.Logger,
) error {
	if js.Outputs == nil {
		js.Outputs = map[string]any{}
	}
	js.Steps = make([]*models.StepState, 0, len(job.Steps))

	for _, step := range job.Steps {
		ss := &models.StepState{Name: step.Name, Status: models.StepStatusRunning}
		js.Steps = append(js.Steps, ss)

		runner := e.registry.Find(step)
		if runner == nil {
			ss.Status = models.StepStatusFailed
			ss.Error = fmt.Sprintf("no runner found for step uses=%q", step.Uses)
			return fmt.Errorf("no runner for step %q (uses=%q)", step.Name, step.Uses)
		}

		start := time.Now()
		sourceName := stepSourceName(step, meta)
		reader, err := e.storage.ReadFile(ctx, id, sourceName)
		if err != nil {
			ss.Status = models.StepStatusFailed
			ss.Error = err.Error()
			return fmt.Errorf("step %q read source %q: %w", step.Name, sourceName, err)
		}
		in := StepInput{
			ObjectID:   id.ID().String(),
			JobID:      jobID,
			Meta:       meta,
			JobOutputs: jobOutputs,
			Reader:     reader,
		}
		out, err := runner.Run(ctx, step, in)
		_ = reader.Close()
		ss.DurationMs = time.Since(start).Milliseconds()

		if err != nil {
			ss.Status = models.StepStatusFailed
			ss.Error = err.Error()
			return fmt.Errorf("step %q: %w", step.Name, err)
		}
		ss.Status = models.StepStatusCompleted

		// Merge step outputs into job outputs
		for k, v := range out.Outputs {
			js.Outputs[k] = v
		}

		// Write artifact if the step produced one
		if out.Writer != nil && out.TargetPath != "" {
			im := out.ItemMeta
			if im == nil {
				im = &models.ItemMeta{}
			}
			im.Role = jobID
			im.UpdateName(out.TargetPath)
			if err := e.storage.WriteFile(ctx, id, out.TargetPath, out.Writer, im); err != nil {
				return fmt.Errorf("step %q write artifact: %w", step.Name, err)
			}
			meta.SetItem(im)
			log.Debug("step artifact written",
				zap.String("path", out.TargetPath),
				zap.String("role", jobID))
		}
	}
	return nil
}

// collectOutputs builds a map[jobID]outputs from completed jobs in state.
func collectOutputs(state *models.ProcessingState) map[string]map[string]any {
	out := make(map[string]map[string]any, len(state.Jobs))
	for id, js := range state.Jobs {
		if js != nil && js.Status == models.JobStatusCompleted && js.Outputs != nil {
			out[id] = js.Outputs
		}
	}
	return out
}

func stepSourceName(step *models.WorkflowStep, meta *models.Meta) string {
	if step != nil {
		if src, ok := step.With["source"].(string); ok && src != "" && !models.IsOriginal(src) {
			if item := meta.ItemByName(src); item != nil && item.Fullname() != "" {
				return item.Fullname()
			}
			return src
		}
	}
	return models.OriginalFilename
}
