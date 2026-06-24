package workflow

import (
	"context"
	"fmt"

	storio "github.com/apfs-io/apfs/internal/storio"
	"github.com/apfs-io/apfs/models"
)

// HasPendingArtifacts reports whether the workflow still has job targets
// that are not present in meta.Items.
func HasPendingArtifacts(w *models.Workflow, meta *models.Meta) bool {
	if meta == nil {
		return w != nil && len(w.Jobs) > 0
	}
	return len(meta.MissingJobTargets(w)) > 0
}

// ProcessObject runs ready workflow jobs for objectID until no more jobs can
// start in this iteration or maxJobs is reached (0 = unlimited).
// complete is true when all jobs are terminal and processing succeeded.
func (e *Executor) ProcessObject(
	ctx context.Context,
	w *models.Workflow,
	objectID string,
	workerTags []string,
	maxJobs int,
) (complete bool, err error) {
	if w == nil || len(w.Jobs) == 0 {
		return true, nil
	}
	if e == nil || e.registry == nil {
		return false, fmt.Errorf("workflow executor not configured")
	}

	dag, err := BuildDAG(w)
	if err != nil {
		return false, err
	}

	id := storio.ObjectIDType(objectID)
	jobsRun := 0

	for {
		state, err := e.storage.ReadState(ctx, id)
		if err != nil {
			return false, fmt.Errorf("process object: load state: %w", err)
		}
		if state == nil {
			state = models.NewProcessingState(objectID, w.Version, w.JobIDs())
			if err := e.storage.WriteState(ctx, id, state); err != nil {
				return false, fmt.Errorf("process object: init state: %w", err)
			}
		}

		ready := dag.ReadyJobs(state, workerTags)
		if len(ready) == 0 {
			state.ComputeProgress()
			state.ComputeStatus()
			return state.Status.IsTerminal() && state.Status.IsSuccess() && !HasPendingArtifacts(w, loadMetaForCheck(ctx, e, id)), nil
		}

		ran := false
		for _, jobID := range ready {
			if maxJobs > 0 && jobsRun >= maxJobs {
				return allJobsTerminal(dag, state) && !HasPendingArtifacts(w, loadMetaForCheck(ctx, e, id)), nil
			}
			if execErr := e.ExecuteJob(ctx, w, objectID, jobID, workerTags); execErr != nil {
				// Retryable failures are reported by ExecuteJob; continue with other ready jobs.
				_ = execErr
			}
			jobsRun++
			ran = true
		}
		if !ran {
			break
		}
	}

	state, err := e.storage.ReadState(ctx, id)
	if err != nil {
		return false, err
	}
	if state == nil {
		return false, nil
	}
	meta, _ := e.storage.ReadMeta(ctx, id)
	return state.Status.IsTerminal() && state.Status.IsSuccess() && !HasPendingArtifacts(w, meta), nil
}

func loadMetaForCheck(ctx context.Context, e *Executor, id storio.ObjectID) *models.Meta {
	meta, err := e.storage.ReadMeta(ctx, id)
	if err != nil {
		return nil
	}
	return meta
}

func allJobsTerminal(dag *DAG, state *models.ProcessingState) bool {
	if dag == nil || state == nil {
		return false
	}
	for _, jobID := range dag.TopologicalOrder() {
		js, ok := state.Jobs[jobID]
		if !ok || js == nil || !js.Status.IsTerminal() {
			return false
		}
	}
	return true
}
